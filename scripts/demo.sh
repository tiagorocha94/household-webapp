#!/usr/bin/env bash
# Starts the server on a copy of the invented demo household (demo/), with its own database, on
# port 8081. The server rewrites the data files it exports, so it never runs on demo/ itself.
# Log in as "alex" or "sam" with the password "demo".
#
# Optional environment: DEMO_PORT (8081), DEMO_DIR (.demo, where the copy and the database go) and
# DEMO_MODULES (a comma-separated list such as "cashflow,goals"; every module when empty).
set -euo pipefail

cd "$(dirname "$0")/.."
bin="${1:?usage: demo.sh PATH_TO_SERVER_BINARY}"

port="${DEMO_PORT:-8081}"
dir="${DEMO_DIR:-.demo}"
modules="${DEMO_MODULES:-}"

rm -rf "$dir"
mkdir -p "$dir"
cp -r demo "$dir/seed"
cat >"$dir/config.yaml" <<YAML
server:
  host: 127.0.0.1
  port: $port
  secure_cookies: false
database:
  path: $dir/household.db
seed:
  dir: $dir/seed
log:
  level: info
YAML
if [ -n "$modules" ]; then
	echo "modules: [$modules]" >>"$dir/config.yaml"
fi

echo "Demo household: http://127.0.0.1:$port  (user alex or sam, password demo)"
exec "$bin" -config "$dir/config.yaml"
