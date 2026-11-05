#!/usr/bin/env bash
set -euo pipefail
manifest="${1:-}"
if [[ -z "$manifest" || "$manifest" == "--help" || "$manifest" == "-h" ]]; then
  echo "Usage: tooling/backup-status.sh <manifest.json>" >&2
  exit 2
fi
if [[ ! -f "$manifest" ]]; then
  echo "manifest not found" >&2
  exit 2
fi
python3 - "$manifest" <<'PY'
import json, sys
m = json.load(open(sys.argv[1], encoding='utf-8'))
ok = (m.get('format') == 'ttg-pitr-backup-v1' and m.get('verified') is True
      and m.get('restore_eligible') is True and not m.get('incomplete'))
print(json.dumps({
    'backup_id': m.get('backup_id'),
    'postgres_version': m.get('postgres_version'),
    'system_identifier': m.get('system_identifier'),
    'start_wal': m.get('start_wal'),
    'end_wal': m.get('end_wal'),
    'timeline': m.get('timeline'),
    'encryption': m.get('encryption'),
    'restore_eligible': bool(ok),
    'blocker': None if ok else 'manifest is incomplete, unverified or ineligible',
}, sort_keys=True))
sys.exit(0 if ok else 1)
PY
