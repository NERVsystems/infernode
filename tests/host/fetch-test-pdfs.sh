#!/bin/sh
#
# Fetch curated PDF test suites for conformance testing.
#
# Clones open-source PDF test repositories into usr/inferno/test-pdfs/.
# Idempotent: skips repos that are already cloned.
# Uses shallow clones (--depth 1) for speed.
#

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DEST="$ROOT/usr/inferno/test-pdfs"

# Use system git to avoid Inferno's git on PATH
GIT=/usr/bin/git

echo "=== Fetching PDF Test Suites ==="
echo "Destination: $DEST"
echo ""

mkdir -p "$DEST"

clone_repo() {
	name="$1"
	url="$2"
	dir="$DEST/$name"

	if [ -d "$dir/.git" ]; then
		echo "  $name: already cloned, skipping"
		return 0
	fi

	echo "  $name: cloning $url ..."
	if $GIT clone --depth 1 "$url" "$dir" 2>&1; then
		echo "  $name: done"
	else
		echo "  $name: FAILED (skipping)"
		rm -rf "$dir"
	fi
}

# 1. pdf-differences — PDF interoperability edge cases (blend modes, fonts, clipping, etc.)
clone_repo "pdf-differences" "https://github.com/pdf-association/pdf-differences.git"

# 2. Poppler test — rendering correctness tests with reference PNGs
clone_repo "poppler-test" "https://gitlab.freedesktop.org/poppler/test.git"

# 3. BFO PDF/A suite — PDF/A-2 conformance (pass/fail labeled by ISO section)
clone_repo "bfo-pdfa" "https://github.com/bfocom/pdfa-testsuite.git"

# 4. PDFTest — reader capabilities (fonts, encryption, content commands)
clone_repo "pdftest" "https://github.com/sambitdash/PDFTest.git"

# 5. PDF Cabinet of Horrors — edge cases from format-corpus (sparse checkout)
CABINET_DIR="$DEST/cabinet-of-horrors"
if [ -d "$CABINET_DIR/.git" ]; then
	echo "  cabinet-of-horrors: already cloned, skipping"
else
	echo "  cabinet-of-horrors: sparse checkout from format-corpus ..."
	if $GIT clone --depth 1 --filter=blob:none --sparse \
		"https://github.com/openpreserve/format-corpus.git" "$CABINET_DIR" 2>&1; then
		cd "$CABINET_DIR"
		$GIT sparse-checkout set pdfCabinetOfHorrors 2>&1
		cd "$ROOT"
		echo "  cabinet-of-horrors: done"
	else
		echo "  cabinet-of-horrors: FAILED (skipping)"
		rm -rf "$CABINET_DIR"
	fi
fi

echo ""

# Count PDFs in each suite
total=0
for dir in "$DEST"/*/; do
	name="$(basename "$dir")"
	count=$(find "$dir" -iname '*.pdf' 2>/dev/null | wc -l | tr -d ' ')
	total=$((total + count))
	echo "  $name: $count PDFs"
done

echo ""
echo "Total: $total PDFs"
echo "=== Done ==="
