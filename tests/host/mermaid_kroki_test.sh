#!/bin/sh
#
# mermaid_kroki_test.sh — Test Mermaid diagram types against Kroki.io
#
# Mirrors the exact POST format used by rendermermaid() in lucifer.b:
#   POST https://kroki.io/mermaid/png
#   Content-Type: text/plain
#   Body: raw Mermaid syntax
#
# For each diagram type, reports: HTTP status, response size, PASS/FAIL.
# On failure, prints the first 200 bytes of the response (usually an error msg).
#
# Usage: sh tests/host/mermaid_kroki_test.sh
#

URL="https://kroki.io/mermaid/png"
TMPDIR=/tmp/mermaid_kroki_test
mkdir -p "$TMPDIR"

passed=0
failed=0

test_diagram() {
    name="$1"
    syntax="$2"

    outfile="$TMPDIR/${name}.png"
    httpcode=$(printf '%s' "$syntax" | curl -s \
        -o "$outfile" \
        -w "%{http_code}" \
        -X POST "$URL" \
        -H "Content-Type: text/plain" \
        --data-binary @-)

    size=0
    if [ -f "$outfile" ]; then
        size=$(wc -c < "$outfile" | tr -d ' ')
    fi

    if [ "$httpcode" = "200" ] && [ "$size" -gt 100 ]; then
        echo "PASS  $name  (HTTP $httpcode, ${size} bytes)"
        passed=$((passed + 1))
    else
        echo "FAIL  $name  (HTTP $httpcode, ${size} bytes)"
        if [ -f "$outfile" ] && [ "$size" -gt 0 ]; then
            snippet=$(head -c 200 "$outfile" | tr -dc '[:print:]\n')
            echo "      response: $snippet"
        fi
        failed=$((failed + 1))
    fi
}

echo "Testing Mermaid via Kroki.io POST (Content-Type: text/plain)"
echo "URL: $URL"
echo "---"

# 1. Flowchart (graph TD)
test_diagram "flowchart_TD" \
"graph TD
    A[Start] --> B{Decision}
    B -->|Yes| C[End]
    B -->|No| D[Retry]"

# 2. Flowchart (graph LR)
test_diagram "flowchart_LR" \
"graph LR
    A --> B --> C"

# 3. Sequence diagram
test_diagram "sequenceDiagram" \
"sequenceDiagram
    participant A as Alice
    participant B as Bob
    A->>B: Hello
    B-->>A: Hi"

# 4. Class diagram
test_diagram "classDiagram" \
"classDiagram
    class Animal {
        +String name
        +makeSound()
    }
    class Dog {
        +fetch()
    }
    Animal <|-- Dog"

# 5. State diagram (v2)
test_diagram "stateDiagram" \
"stateDiagram-v2
    [*] --> Idle
    Idle --> Working : start
    Working --> Idle : done
    Working --> [*] : quit"

# 6. Gantt chart — minimal
test_diagram "gantt_minimal" \
"gantt
    title Minimal Gantt
    dateFormat YYYY-MM-DD
    section Work
        Task A :2024-01-01, 7d"

# 7. Gantt chart — with dependencies and sections
test_diagram "gantt_deps" \
"gantt
    title Project Timeline
    dateFormat YYYY-MM-DD
    section Phase 1
        Research :a1, 2024-01-01, 14d
        Design   :a2, after a1, 10d
    section Phase 2
        Build    :a3, after a2, 30d
        Test     :a4, after a3, 14d"

# 8. Gantt chart — with crit and active
test_diagram "gantt_crit" \
"gantt
    title Schedule
    dateFormat YYYY-MM-DD
    section Critical
        Deploy  :crit, 2024-03-01, 5d
    section Normal
        Review  :active, 2024-03-06, 3d"

# 9. Pie chart
test_diagram "pie" \
"pie title Resource Usage
    \"CPU\" : 45
    \"Memory\" : 30
    \"Disk\" : 25"

# 10. Pie chart — showData variant
test_diagram "pie_showData" \
"pie showData
    title Votes
    \"Option A\" : 60
    \"Option B\" : 40"

# 11. XY chart (bar)
test_diagram "xychart_bar" \
"xychart-beta
    title \"Monthly Sales\"
    x-axis [Jan, Feb, Mar, Apr]
    y-axis \"Revenue\" 0 --> 10000
    bar [4000, 6000, 8000, 5000]"

# 12. XY chart (line)
test_diagram "xychart_line" \
"xychart-beta
    title \"Temperature\"
    x-axis [Mon, Tue, Wed, Thu, Fri]
    y-axis \"Celsius\" 0 --> 40
    line [22, 25, 30, 28, 24]"

# 13. Mindmap
test_diagram "mindmap" \
"mindmap
  root((Project))
    Research
      Papers
      Experiments
    Development
      Frontend
      Backend"

# 14. ER diagram
test_diagram "erDiagram" \
"erDiagram
    CUSTOMER ||--o{ ORDER : places
    ORDER ||--|{ LINE_ITEM : contains
    CUSTOMER {
        string name
        string email
    }"

# 15. Timeline
test_diagram "timeline" \
"timeline
    title History
    2020 : Event A
    2021 : Event B
         : Event C
    2022 : Event D"

# 16. Git graph
test_diagram "gitGraph" \
"gitGraph
    commit id: \"init\"
    branch feature
    checkout feature
    commit id: \"add\"
    checkout main
    merge feature"

# 17. Quadrant chart
test_diagram "quadrantChart" \
"quadrantChart
    title Effort vs Impact
    x-axis Low Effort --> High Effort
    y-axis Low Impact --> High Impact
    Task A: [0.3, 0.8]
    Task B: [0.7, 0.6]"

# 18. Requirement diagram
test_diagram "requirementDiagram" \
"requirementDiagram
    requirement req1 {
        id: 1
        text: The system shall run
        risk: Low
        verifymethod: Test
    }"

# 19. Journey diagram
test_diagram "journey" \
"journey
    title User Journey
    section Login
        Open app: 5: User
        Enter credentials: 3: User
        Submit: 4: User, System"

# 20. Block diagram
test_diagram "block_beta" \
"block-beta
    columns 3
    A B C
    D E F"

echo "---"
echo "Results: $passed passed, $failed failed"
echo "PNG files saved to $TMPDIR/"
