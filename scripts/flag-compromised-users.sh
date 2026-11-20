#!/usr/bin/env bash
# Flags users so they must choose a new password at their next visit.
#
# Usage: sudo scripts/flag-compromised-users.sh <database> <usernames-file | --all>
#   <database>         path to the SQLite database, e.g. /srv/whoknows/data/whoknows.db
#   <usernames-file>   flag only the users listed in this file, one username per line
#                      (a CSV with a single column also works)
#   --all              flag every user, e.g. when the whole database has leaked
#
# Safe to run repeatedly: flagging a user who is already flagged changes nothing.
# Take a backup of the database first.
set -euo pipefail

if [ "$#" -ne 2 ]; then
	echo "Usage: $0 <database> <usernames-file | --all>" >&2
	exit 1
fi

db_file="$1"
target="$2"

if [ ! -f "$db_file" ]; then
	echo "Database not found: $db_file" >&2
	exit 1
fi

if [ "$target" = "--all" ]; then
	sqlite3 -bail "$db_file" <<SQL
.timeout 5000
UPDATE users SET must_change_password = 1;
SELECT changes() || ' user(s) flagged';
SQL
	exit 0
fi

if [ ! -f "$target" ]; then
	echo "Usernames file not found: $target" >&2
	exit 1
fi

sqlite3 -bail "$db_file" <<SQL
.timeout 5000
CREATE TEMP TABLE leaked (username TEXT);
.import --csv "$target" leaked
UPDATE users SET must_change_password = 1 WHERE username IN (SELECT username FROM leaked);
SELECT changes() || ' user(s) flagged';
SQL