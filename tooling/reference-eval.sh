#!/usr/bin/env bash
# TTGQLDL — independent gameplay reference evaluator.
# Usage: ./reference-eval.sh [-check] [-sweep] [-write]
# Default (no flag) verifies the golden vectors and the hand-derived assertions.
# Exit codes: 0 = assertions hold, 1 = assertion or golden drift, 2 = IO error.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$here"
if [ "$#" -eq 0 ]; then
    exec go run ./refeval -check
fi
exec go run ./refeval "$@"
