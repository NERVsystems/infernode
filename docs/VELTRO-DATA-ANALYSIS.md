# Veltro Data Analysis: Design Discussion

## Problem

Veltro needs to work with datasets — CSV files, tabular data, structured
data of any kind. "Here's a dataset, what's the average of column 3?"
"Find all rows where status is failed." "Correlate these two columns."
"Summarize this data."

Today, Veltro can `read` a file as raw text and have the LLM eyeball it.
That works for small files but is token-expensive, slow, and can't handle
computation. The agent needs actual data analysis capabilities.

## Current State

### What Veltro has

- `read` tool — line-oriented file reading (max 1000 lines)
- `json` tool — JSON parsing with path-based extraction
- `grep` / `search` — regex matching
- `exec` tool — shell command execution (5s timeout, 16KB output)
- Shell builtins: `getcsv` (CSV row parser), `expr`/`mpexpr` (arithmetic)
- CLI tools: `sort`, `uniq`, `wc`, `grep`, `sed`, `tr`, `comm`

### What's missing

- No `awk` or `cut` (upstream Inferno never had them either)
- No column extraction, aggregation, or statistics primitives
- No way to process more than ~1000 lines of data efficiently
- Shell `expr` does arithmetic but no accumulator pattern across rows

## Approaches

### Approach A: Containerized Python Environment via 9P

Run a Docker container with the Python data science stack (pandas, numpy,
scipy, matplotlib, scikit-learn, etc.) and expose it to Inferno as a 9P
filesystem. The agent writes Python code, sends it through the filesystem
interface, and reads back results.

#### Architecture

```
┌─────────────────────────────────────────────┐
│  Docker Container                           │
│                                             │
│  Python 3.x + pandas + numpy + scipy + ...  │
│  pyroute2 Plan9ServerSocket (9P2000)        │
│  ┌─────────────────────────────────────┐    │
│  │  /n/python/                         │    │
│  │  ├── new        → session ID        │    │
│  │  └── N/                             │    │
│  │      ├── eval   → write code, read  │    │
│  │      │            stdout/result     │    │
│  │      ├── data   → write CSV/JSON,   │    │
│  │      │            loads as DataFrame│    │
│  │      ├── env    → read/write vars   │    │
│  │      ├── plot   → read last plot as │    │
│  │      │            PNG/SVG           │    │
│  │      └── error  → read stderr       │    │
│  └─────────────────────────────────────┘    │
│                    TCP :5642                 │
└──────────────────────┬──────────────────────┘
                       │ 9P2000
┌──────────────────────┴──────────────────────┐
│  Inferno (emu)                              │
│  mount -A tcp!127.0.0.1!5642 /n/python      │
│                                             │
│  Veltro agent:                              │
│    echo 'import pandas as pd              ' │
│         'df = pd.read_csv("/data/sales.csv")│
│         'print(df.describe())             ' │
│         > /n/python/0/eval                  │
│    cat /n/python/0/eval                     │
└─────────────────────────────────────────────┘
```

#### 9P Server Implementation

**pyroute2** (actively maintained, PyPI) has a rewritten 9P2000 server:

```python
from pyroute2.plan9.server import Plan9ServerSocket

server = Plan9ServerSocket(address='0.0.0.0', port=5642)
server.filesystem.create('eval')
server.register_function('eval', execute_python)
server.serve()
```

Note: pyroute2's 9P is basic — "no 9p2000.L yet" — but 9P2000 is what
Inferno speaks natively (Styx), so this is actually fine. pyroute2 already
supports registering Python functions as synthetic files that accept JSON
input and return JSON output — this maps directly to what we need.

Other Python 9P options (all less suitable):
- **py9p** (deprecated, folded into pyroute2)
- **python-styx** (Plan9-Archive, historical — but has a `dictserver.py`
  reference for serving Python data structures as 9P files)
- **ixpy** — newer, simpler 9P virtual filesystem library
- **objectfs/PyVFS** — pluggable VFS with 9p2000.u backend

Alternatively, a **Go shim** using knusbaum/go9p could serve 9P and pipe
code to a Python subprocess. This gives a more mature 9P implementation
while keeping Python for the actual computation.

No existing container or turnkey solution combines a Python data science
environment with a 9P filesystem interface. This would be novel but the
pieces are all available.

#### Docker Image

A standard data science container:

```dockerfile
FROM python:3.12-slim
RUN pip install pandas numpy scipy matplotlib scikit-learn pyroute2
COPY serve9p.py /app/
EXPOSE 5642
CMD ["python", "/app/serve9p.py"]
```

Or use an existing image like `jupyter/scipy-notebook` and add the 9P
server layer.

#### Pros

- Full Python data science ecosystem — pandas, numpy, scipy, sklearn,
  matplotlib, seaborn, statsmodels, etc.
- LLMs are excellent at generating Python/pandas code
- Visualization via matplotlib (return plots as images)
- Huge community, endless documentation and examples
- Can install any pip package for specialized work
- Session state: DataFrames persist across calls within a session

#### Cons

- External dependency: Docker must be available on the host
- New infrastructure to build, deploy, and maintain
- Security surface: executing arbitrary Python code (sandboxed by container)
- Latency: 9P network round-trip + Python interpretation
- pyroute2's 9P server is basic; may need Go shim for robustness
- Doesn't follow Inferno's self-contained philosophy
- Container startup time (mitigated by keeping it running)

#### Effort: Medium

Build the 9P server (~200 lines Python or ~300 lines Go), Dockerfile,
mount integration in profile. The Python execution sandbox itself is
straightforward (exec with captured stdout/stderr).


### Approach B: Agent Composes Limbo Modules at Runtime

The agent writes custom Limbo code, compiles it using the in-Inferno
compiler, and executes it. This leverages Inferno's existing runtime
compilation capability (proven in limbtest.b) and the surprisingly rich
set of numerical/data libraries already present.

#### How It Works

```
┌─────────────────────────────────────────────┐
│  Veltro Agent                               │
│                                             │
│  1. LLM generates Limbo source code         │
│     tailored to the specific data task      │
│                                             │
│  2. write tool → /tmp/veltro/scratch/a.b    │
│                                             │
│  3. exec tool → limbo -I /module \          │
│       -o /tmp/veltro/scratch/a.dis \        │
│       /tmp/veltro/scratch/a.b               │
│                                             │
│  4. exec tool → /tmp/veltro/scratch/a.dis \ │
│       /path/to/data.csv                     │
│                                             │
│  5. read output, reason about results,      │
│     iterate if needed                       │
└─────────────────────────────────────────────┘
```

#### Available Libraries

Limbo has more numerical computing support than you'd expect:

| Module | Capabilities |
|--------|-------------|
| Math.m | trig, exp, log, sqrt, pow, erf, gamma, Bessel, floor/ceil, dot product, norms, BLAS gemm (matrix multiply), sort |
| LinAlg.m | LU decomposition (dgefa), linear system solving (dgesl), matrix printing |
| FFTs.m | Mixed-radix FFT, multivariate, real/imaginary |
| CSV.m | Parse CSV lines into `list of string` |
| JSON.m | Full JSON parser/generator with object/array navigation |
| Tables.m | Hash tables — `Table[T]` (int-keyed), `Strhash[T]` (string-keyed) |
| Lists.m | `map`, `filter`, `partition`, `allsat`, `anysat`, `pair`, `unpair` — with parametric polymorphism |
| Sort.m | Polymorphic sorting with custom comparators |
| String.m | Split, join, parse (toint, tobig, toreal), case conversion |
| Bufio.m | Buffered file I/O |
| IPints.m | Arbitrary precision integers |

This is enough for: descriptive statistics (mean, median, stddev, min, max),
filtering, grouping, sorting, correlation, linear regression, FFT, matrix
operations, and more.

#### Example: Agent-Generated Analytics Module

For a request like "what's the average revenue by region?", the agent
would generate:

```limbo
implement Analytics;

include "sys.m";
    sys: Sys;
include "draw.m";
include "bufio.m";
    bufio: Bufio;
    Iobuf: import bufio;
include "math.m";
    math: Math;
include "csv.m";
    csv: CSV;
include "string.m";
    str: String;
include "tables.m";
    tables: Tables;
    Strhash: import tables;

Analytics: module {
    init: fn(nil: ref Draw->Context, args: list of string);
};

init(nil: ref Draw->Context, args: list of string)
{
    sys = load Sys Sys->PATH;
    bufio = load Bufio Bufio->PATH;
    math = load Math Math->PATH;
    csv = load CSV CSV->PATH;
    csv->init(bufio);
    str = load String String->PATH;
    tables = load Tables Tables->PATH;

    path := "/data/sales.csv";
    if(args != nil && tl args != nil)
        path = hd tl args;

    fd := bufio->open(path, Sys->OREAD);
    if(fd == nil) {
        sys->fprint(sys->fildes(2), "cannot open %s: %r\n", path);
        raise "fail:open";
    }

    # Skip header
    header := csv->getline(fd);

    # Find column indices (region=0, revenue=2 for example)
    sums := Strhash[ref (real, int)].new(31, nil);

    for(;;) {
        fields := csv->getline(fd);
        if(fields == nil)
            break;

        # Extract region and revenue
        fl := fields;
        region := hd fl; fl = tl fl; fl = tl fl;
        revenue := real hd fl;

        entry := sums.find(region);
        if(entry == nil) {
            e := ref (0.0, 0);
            sums.add(region, e);
            entry = e;
        }
        (s, n) := *entry;
        *entry = (s + revenue, n + 1);
    }

    # Output results
    sys->print("%-20s %12s %8s\n", "region", "avg_revenue", "count");
    sys->print("%-20s %12s %8s\n", "------", "-----------", "-----");
    # iterate hash... (agent fills in iteration logic)
}
```

#### Reusable Module Templates

Rather than generating everything from scratch each time, the agent could
draw from a library of composable template modules:

- **csv_loader.b** — Load CSV into columnar arrays (string columns,
  real columns, with header parsing)
- **stats.b** — Descriptive statistics (mean, median, stddev, min, max,
  percentiles, variance, skewness)
- **groupby.b** — Group rows by a string column, apply aggregate functions
- **filter.b** — Filter rows by conditions on any column
- **sort_table.b** — Sort by column with ascending/descending
- **correlation.b** — Pearson/Spearman correlation between columns
- **pivot.b** — Pivot table generation
- **regression.b** — Linear regression using LinAlg (LU solve)
- **histogram.b** — Frequency distribution / binning
- **join.b** — Join two datasets on a key column

These would live in `/appl/lib/analytics/` and the agent would `include`
and compose them as needed, writing only the glue code specific to each
query.

#### Pros

- Zero new infrastructure — works today with existing tools
- Native to Inferno — no external dependencies, no containers, no network
- Fast iteration: write → compile (milliseconds) → run → read output
- Runs in Dis VM sandbox with namespace restrictions (secure by default)
- Agent can inspect compiler errors and self-correct
- Limbo is statically typed — catches errors at compile time before execution
- Libraries are surprisingly capable (BLAS-level matrix ops, FFT, etc.)
- Reusable template library amortizes the cost of Limbo's verbosity

#### Cons

- LLMs are less fluent in Limbo than Python (but can learn from examples)
- No equivalent to pandas DataFrames — agent must manage arrays manually
- No visualization (no matplotlib equivalent)
- Smaller ecosystem — no sklearn, no statsmodels
- Limbo is more verbose than Python for the same task
- 5s exec timeout may be tight for large datasets (configurable)
- Template library needs to be built up front

#### Effort: Medium

The template modules (~500-800 lines total for core set) plus testing.
The compilation/execution pipeline already exists.


### Approach C: Hybrid — Limbo Native + Python Container for Heavy Lifting

Use Limbo module composition for routine data tasks (filtering, basic
stats, grouping, sorting) where it's fast and self-contained. Mount the
Python container for heavy analytics (ML, complex visualization,
statistical modeling) when needed.

```
Agent decides based on task complexity:
  "what's the average?" → Limbo module (fast, no dependencies)
  "train a classifier"  → Python container (full sklearn)
  "plot a histogram"    → Python container (matplotlib)
  "filter rows where…"  → Limbo module (fast, native)
```

The Veltro tool system already supports capability-based tool selection,
so adding a `python` tool alongside the existing `exec` tool is natural.

#### Pros

- Best of both worlds — fast native path + full Python ecosystem
- Graceful degradation: works without Docker (Limbo path always available)
- Agent learns to use the right tool for the job

#### Cons

- Two systems to maintain
- Agent needs to know when to use which path

#### Effort: Large (both A and B combined)


## Recommendation

**Start with Approach B (Limbo module composition).**

Rationale:
1. It works today — no new infrastructure to build or deploy
2. It's native to Inferno's philosophy (everything is a module, everything
   runs in the VM, everything is sandboxed by namespace)
3. The numerical libraries are genuinely capable — Math.m alone has
   BLAS gemm, dot product, norms, full trig, error functions, Bessel
   functions. LinAlg.m does LU decomposition. FFTs.m does multivariate
   FFT. This isn't a toy.
4. Building the template module library is useful regardless — even if
   Python is added later, having fast native analytics for common tasks
   is valuable
5. The LLM can iterate on compile errors (Limbo's type system catches
   bugs early, and the agent can read compiler output and fix issues)

**Then add Approach A (Python container) when specific needs arise** —
machine learning, complex visualization, or access to specialized Python
libraries that have no Limbo equivalent.

## Template Module Library (Phase 1)

Priority order for building the reusable analytics modules:

1. **csv_table.b** — Load CSV into columnar storage, header parsing,
   type inference (string vs real vs int), row/column access
2. **stats.b** — mean, median, stddev, min, max, sum, count, variance
3. **groupby.b** — Group by string column + apply aggregate
4. **filter.b** — Row filtering by column conditions
5. **tabfmt.b** — Formatted table output (aligned columns, truncation)

These five cover the vast majority of "here's a dataset, tell me about it"
use cases.

## Open Questions

- Should the template modules be standalone commands or importable libraries?
  (Commands are simpler for exec; libraries compose better)
- How should the agent handle datasets larger than available memory?
  (Streaming/chunked processing vs. loading everything)
- Should we pre-build common analytics commands (like a `csvstat` command)
  so the agent doesn't always need to generate code?
- What's the right timeout for analytics exec? 5s default may be too
  short for large datasets.
