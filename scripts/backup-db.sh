#!/usr/bin/env bash
set -euo pipefail
umask 077

db_file="${1:-/home/azureuser/whoknows/backend/whoknows.db}"
backup_dir="${2:-/home/azureuser/backups/whoknows}"

db_file=$(realpath -e -- "$db_file")
mkdir -p -- "$backup_dir"
cd -- "$backup_dir"

backup_file=$(mktemp "whoknows-$(date -u +%Y%m%dT%H%M%SZ)-XXXXXX.db")
trap 'rm -f -- "$backup_file"' EXIT

sqlite3 -readonly -cmd ".timeout 5000" "$db_file" ".backup '$backup_file'"
result=$(sqlite3 -readonly "$backup_file" "PRAGMA integrity_check;")

if [[ "$result" != "ok" ]]; then
    printf 'Backup integrity check failed: %s\n' "$result" >&2
    exit 1
fi

trap - EXIT
printf 'Backup verified: %s/%s\n' "$PWD" "$backup_file"
