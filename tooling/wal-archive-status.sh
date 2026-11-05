#!/usr/bin/env bash
set -euo pipefail
archive="${1:-}"
if [[ -z "$archive" || "$archive" == "--help" || "$archive" == "-h" ]]; then
  echo "Usage: tooling/wal-archive-status.sh <archive-directory>" >&2
  exit 2
fi
if [[ ! -d "$archive" ]]; then
  echo "archive directory not found" >&2
  exit 2
fi
count=$(find "$archive" -maxdepth 1 -type f -name '0*' | wc -l | tr -d ' ')
latest=$(find "$archive" -maxdepth 1 -type f -name '0*' -printf '%f\n' | sort | tail -1 || true)
latest_mtime=$(find "$archive" -maxdepth 1 -type f -name '0*' -printf '%T@\n' | sort -n | tail -1 || true)
printf '{"archive":"%s","wal_segments":%s,"latest_segment":"%s","latest_mtime_epoch":"%s"}\n' "$archive" "$count" "$latest" "$latest_mtime"
