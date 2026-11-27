#!/usr/bin/env bash
# Checks that the code in this checkout can take over the data of a previous release. It runs the
# previous release's server on that release's demo household, so a real database and the exported
# CSV files exist, then starts this checkout's server on the same database and files. The upgrade
# is fine when:
#   - the new server starts (its migrations and import accept what the old one left behind);
#   - every answer of scripts/snapshot-api.sh is 200, and is the same as before the upgrade;
#   - the CSV files are the same as before the upgrade.
# An answer that differs on purpose (a field this release adds) shows up in the diff; judge it.
#
#   scripts/upgrade-check.sh OLD_BINARY OLD_DEMO_DIR NEW_BINARY
#
# OLD_DEMO_DIR is the demo/ folder of the previous release (its data, in the format it wrote).
# Needs bash, curl and jq. Set UPGRADE_PORT to change the port (18081). The work is done in a
# temporary folder that is kept when the check fails and removed when it passes.
set -euo pipefail

old_bin="${1:?usage: upgrade-check.sh OLD_BINARY OLD_DEMO_DIR NEW_BINARY}"
old_demo="${2:?usage: upgrade-check.sh OLD_BINARY OLD_DEMO_DIR NEW_BINARY}"
new_bin="${3:?usage: upgrade-check.sh OLD_BINARY OLD_DEMO_DIR NEW_BINARY}"
here="$(cd "$(dirname "$0")" && pwd)"
port="${UPGRADE_PORT:-18081}"
base="http://127.0.0.1:$port"

work="$(mktemp -d)"
mkdir -p "$work/seed"
cp -r "$old_demo"/. "$work/seed/"
cat >"$work/config.yaml" <<YAML
server:
  host: 127.0.0.1
  port: $port
database:
  path: $work/household.db
seed:
  dir: $work/seed
YAML

server_pid=""
stop_server() {
	if [ -n "$server_pid" ]; then
		kill "$server_pid" 2>/dev/null || true
		wait "$server_pid" 2>/dev/null || true
		server_pid=""
	fi
}
failed=1
cleanup() {
	stop_server
	if [ "$failed" -eq 0 ]; then
		rm -rf "$work"
	else
		echo "kept $work for a look" >&2
	fi
}
trap cleanup EXIT

# run NAME BINARY: starts the server on the shared database and waits until it answers.
run() {
	echo "== starting the $1 server"
	"$2" -config "$work/config.yaml" >"$work/$1.log" 2>&1 &
	server_pid=$!
	for _ in $(seq 1 60); do
		if curl -fsS "$base/api/v1/info" >/dev/null 2>&1; then
			return 0
		fi
		if ! kill -0 "$server_pid" 2>/dev/null; then
			break
		fi
		sleep 0.5
	done
	echo "the $1 server did not start; its log:" >&2
	cat "$work/$1.log" >&2
	exit 1
}

# snapshot NAME: saves the API answers to WORK/NAME, from the running server.
snapshot() {
	HOUSEHOLD_USER=alex HOUSEHOLD_PASSWORD=demo bash "$here/snapshot-api.sh" "$base" "$work/$1"
}

run old "$old_bin"
snapshot before
stop_server
cp -r "$work/seed" "$work/seed-before"

echo "== what the new server would change (dry run)"
"$new_bin" -config "$work/config.yaml" -dry-run

run new "$new_bin"
snapshot after
stop_server

status=0
if ! diff -r "$work/seed-before" "$work/seed"; then
	echo "The CSV files differ after the upgrade (above)." >&2
	status=1
fi
if ! diff -r "$work/before" "$work/after"; then
	echo "The API answers differ after the upgrade (above). A field this release adds is expected; anything else is not." >&2
	status=1
fi
if [ "$status" -eq 0 ]; then
	echo "Upgrade check passed: same data, same answers."
fi
failed=$status
exit "$status"
