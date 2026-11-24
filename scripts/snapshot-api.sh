#!/usr/bin/env bash
# Saves what the API answers, for every user-visible list and detail, as normalised JSON files, so
# two runs can be compared with `diff -r`. Used to check that a change of code or of data location
# leaves the answers unchanged, and as a smoke test: it fails on any answer that is not 200.
#
#   HOUSEHOLD_USER=... HOUSEHOLD_PASSWORD=... scripts/snapshot-api.sh http://127.0.0.1:8080 out-dir
#
# Needs curl and jq. What is compared is the content, not the bookkeeping of one database:
#   - files are named after what they hold (an account's name, a month), not its numeric id;
#   - the ids (`id` and every `...Id`) and the `createdAt` times are left out, because two
#     databases built from the same files number and date their rows differently;
#   - the items of every list are put in a fixed order, so only the content is compared.
# The overview depends on today's date, so compare runs made on the same day.
set -euo pipefail

base="${1:?usage: snapshot-api.sh BASE_URL OUT_DIR}"
out="${2:?usage: snapshot-api.sh BASE_URL OUT_DIR}"
: "${HOUSEHOLD_USER:?set HOUSEHOLD_USER}" "${HOUSEHOLD_PASSWORD:?set HOUSEHOLD_PASSWORD}"
api="${base%/}/api/v1"
jar="$(mktemp)"
trap 'rm -f "$jar"' EXIT
mkdir -p "$out"

# Logs in; the session cookie is kept in the jar.
login="$(jq -n --arg u "$HOUSEHOLD_USER" --arg p "$HOUSEHOLD_PASSWORD" '{username:$u,password:$p}')"
curl -fsS -c "$jar" -H 'Content-Type: application/json' -d "$login" "$api/auth/login" >/dev/null

# The same answer from two databases: no ids or creation times, lists in a fixed order.
normalise='
  def clean: walk(
    if type == "object" then with_entries(select(.key != "id" and .key != "createdAt" and (.key | test("Id$") | not)))
    elif type == "array" then sort_by(tojson)
    else . end);
  clean'

# fetch PATH: prints the answer.
fetch() { curl -fsS -b "$jar" "$api$1"; }
# save PATH NAME: saves the normalised answer to OUT/NAME.json.
save() {
  local file="$out/$2.json"
  mkdir -p "$(dirname "$file")"
  # jq.exe on Windows ends its lines with CRLF; the carriage returns are removed everywhere.
  fetch "$1" | jq -S "$normalise" | tr -d '\r' >"$file"
}
# each LIST FILTER: for every item of a list prints "id<TAB>name" (the name is made safe for a file name).
each() {
  fetch "$1" | jq -r "$2" | tr -d '\r'
}
slug() { printf '%s' "$1" | tr -c 'A-Za-z0-9\n' '-'; }
ITEMS='.[] | "\(.id)\t\(.name)"'

save /auth/me auth-me
save /overview overview
save /overview/net-worth net-worth

while IFS=$'\t' read -r id name; do
  dir="goals/$(slug "$name")"
  for sub in "" /goals /movements /projection; do save "/goals/accounts/$id$sub" "$dir/account${sub//\//-}"; done
  while IFS=$'\t' read -r goal gname; do save "/goals/$goal" "$dir/goal-$(slug "$gname")"; done < <(each "/goals/accounts/$id/goals" "$ITEMS")
done < <(each /goals/accounts "$ITEMS")

save /credits/due credits/due
while IFS=$'\t' read -r id name; do
  for sub in "" /payments; do save "/credits/$id$sub" "credits/$(slug "$name")${sub//\//-}"; done
done < <(each /credits "$ITEMS")

while IFS=$'\t' read -r id name; do
  for sub in "" /movements /snapshots; do save "/savings/accounts/$id$sub" "savings/$(slug "$name")${sub//\//-}"; done
done < <(each /savings/accounts "$ITEMS")

while IFS=$'\t' read -r id name; do
  edir="investments/$(slug "$name")"
  for sub in "" /instruments; do save "/investments/entities/$id$sub" "$edir/entity${sub//\//-}"; done
  while IFS=$'\t' read -r inst iname; do
    for sub in "" /contributions /snapshots; do
      save "/investments/instruments/$inst$sub" "$edir/$(slug "$iname")${sub//\//-}"
    done
  done < <(each "/investments/entities/$id/instruments" "$ITEMS")
done < <(each /investments/entities "$ITEMS")

save /cashflow/categories cashflow/categories
for month in $(each /cashflow/months '.[].month'); do
  for path in "/cashflow/months/$month" "/cashflow/months/$month/movements" "/cashflow/balance/$month" "/cashflow/budget/$month"; do
    save "$path" "cashflow/${path#/cashflow/}"
  done
done

echo "saved $(find "$out" -name '*.json' | wc -l) answers in $out"
