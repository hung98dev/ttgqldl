#!/usr/bin/env bash
# TTGQLDL — canonical plan validator.
# Usage: ./validate-plan.sh [-json] [-v]
# Exit codes: 0 = clean, 1 = errors found, 2 = usage/IO error.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/.." && pwd)"
cd "$here"
exec go run ./planvalidate -root "$root" "$@"
