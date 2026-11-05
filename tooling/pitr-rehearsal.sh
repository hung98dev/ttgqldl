#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  echo "Usage: tooling/pitr-rehearsal.sh"
  echo "Runs the isolated encrypted physical backup + WAL archive + PITR rehearsal."
  echo "Requires docker and a disposable local environment; synthetic data only."
  exit 0
fi
cd "$(dirname "$0")/../backend"
TTG_PITR_TEST=1 go test -count=1 -timeout 10m -v ./test/pitr/...
