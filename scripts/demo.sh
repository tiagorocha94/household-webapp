#!/usr/bin/env bash
# Starts the server on a copy of the invented demo household (demo/), with its own database, on
# port 8081. The server rewrites the data files it exports, so it never runs on demo/ itself.
# Log in as "alex" or "sam" with the password "demo".
set -euo pipefail

cd "$(dirname "$0")/.."
bin="${1:?usage: demo.sh PATH_TO_SERVER_BINARY}"

rm -rf .demo
mkdir -p .demo
cp -r demo .demo/seed
cat >.demo/config.yaml <<'YAML'
server:
  host: 127.0.0.1
  port: 8081
  secure_cookies: false
database:
  path: .demo/household.db
seed:
  dir: .demo/seed
log:
  level: info
YAML

echo "Demo household: http://127.0.0.1:8081  (user alex or sam, password demo)"
exec "$bin" -config .demo/config.yaml
