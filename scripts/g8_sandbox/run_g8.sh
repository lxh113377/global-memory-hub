#!/bin/sh
# G8 driver. Emits one NDJSON record per step. Exit code is three-state:
#   0 = every required step passed
#   1 = a required step was judged red
#   2 = a step could not be measured (UNVERIFIED) -- never folded into 0
set -u
BIN=/g8/bin/fenjue-agent
CFG=/g8/platforms.json
RC=0
FAILED=""
UNVER=""

# $5 defaults to empty: under `set -u` a short call would otherwise abort the
# whole run instead of emitting a record.
emit() { printf '{"step":"%s","platform":"%s","rc":%s,"verdict":"%s","evidence":"%s"}\n' "${1:-}" "${2:-}" "${3:-}" "${4:-}" "${5:-}"; }
flat() { echo "$1" | tr '\n' ' '; }

for P in mm ds; do
  if [ "$P" = "mm" ]; then HOME_DIR=/g8/home/.minimax; else HOME_DIR=/g8/home/.dsh; fi

  OUT=$("$BIN" enable "$P" --platforms "$CFG" 2>&1); ERC=$?
  if [ "$ERC" -ne 0 ]; then
    emit enable "$P" "$ERC" RED "$(flat "$OUT" | cut -c1-200)"
    RC=1; FAILED="$FAILED enable:$P"; continue
  fi
  emit enable "$P" 0 PASS "$(flat "$OUT" | cut -c1-200)"

  # verify: keep only this platform's block, otherwise the evidence for ds
  # would be mm's lines (verify prints every platform, we truncate per platform).
  VOUT=$("$BIN" verify --platforms "$CFG" 2>&1); VRC=$?
  VEV=$(echo "$VOUT" | grep -A3 "platform $P " | tr '\n' ' ')
  if [ "$VRC" -eq 0 ]; then V=PASS; else V=RED; RC=1; FAILED="$FAILED verify:$P"; fi
  emit verify "$P" "$VRC" "$V" "$(echo "$VEV" | cut -c1-240)"

  for K in memory skills; do
    T="$HOME_DIR/$K"
    if [ -L "$T" ]; then
      emit link "$P/$K" 0 PASS "symlink -> $(readlink "$T")"
    else
      emit link "$P/$K" 1 RED "not a symlink at $T"
      RC=1; FAILED="$FAILED link:$P/$K"
    fi
  done

  ZRC=0; "$BIN" disable "$P" --platforms "$CFG" >/dev/null 2>&1 || ZRC=$?
  R2=$("$BIN" enable "$P" --platforms "$CFG" 2>&1); RRC=$?
  if [ "$ZRC" -eq 0 ] && [ "$RRC" -eq 0 ] && echo "$R2" | grep -q "zero-overwrite"; then
    emit zero-overwrite "$P" 0 PASS "re-enable reported zero-overwrite"
  else
    emit zero-overwrite "$P" 1 RED "disable=$ZRC re-enable=$RRC $(flat "$R2" | cut -c1-160)"
    RC=1; FAILED="$FAILED zero-overwrite:$P"
  fi

  # The target program is a desktop app with no headless CLI, so consumption
  # cannot be measured here. UNVERIFIED, never folded into the pass count.
  emit consumption "$P" 2 UNVERIFIED "no headless client in sandbox; desktop app only"
  UNVER="$UNVER consumption:$P"
done

# counter-example leg: an undeclared platform must be refused
URC=0; "$BIN" enable nonexist --platforms "$CFG" >/dev/null 2>&1 || URC=$?
if [ "$URC" -ne 0 ]; then
  emit unknown-platform "$URC" 0 PASS "undeclared platform refused (rc=$URC)"
else
  emit unknown-platform 0 1 RED "undeclared platform was accepted"
  RC=1; FAILED="$FAILED unknown-platform"
fi

echo "{\"summary\":{\"rc\":$RC,\"failed\":\"${FAILED# }\",\"unverified\":\"${UNVER# }\"}}"
exit $RC