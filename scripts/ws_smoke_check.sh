#!/usr/bin/env bash
set -euo pipefail

# Manual smoke-check helper for realtime chat WS.
# Requirements: jq, curl, wscat (npm i -g wscat)

BASE_URL="${BASE_URL:-http://localhost:8080}"
TOKEN_A="${TOKEN_A:-}"
TOKEN_B="${TOKEN_B:-}"
ROOM_ID="${ROOM_ID:-}"

if [[ -z "$TOKEN_A" || -z "$TOKEN_B" || -z "$ROOM_ID" ]]; then
  echo "Usage: BASE_URL=http://localhost:8080 TOKEN_A=... TOKEN_B=... ROOM_ID=... $0"
  exit 1
fi

echo "1) Open two terminals and run:"
echo "wscat -c \"${BASE_URL/http/ws}/ws?token=${TOKEN_A}\""
echo "wscat -c \"${BASE_URL/http/ws}/ws?token=${TOKEN_B}\""

echo
echo "2) In both wscat sessions subscribe to room:"
echo '{"type":"join","room_id":"'"$ROOM_ID"'"}'

echo
echo "3) Send text message from A via HTTP:"
curl -sS -X POST "${BASE_URL}/api/v1/chat/rooms/${ROOM_ID}/messages" \
  -H "Authorization: Bearer ${TOKEN_A}" \
  -H "Content-Type: application/json" \
  -d '{"content":"hello from smoke check","message_type":"text"}' | jq .

echo
echo "4) Send attachment message (replace upload id with committed chat_file upload):"
echo "curl -X POST \"${BASE_URL}/api/v1/chat/rooms/${ROOM_ID}/messages\" \\\n  -H \"Authorization: Bearer ${TOKEN_A}\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"attachment_upload_ids\":[\"<UPLOAD_ID>\"]}'"

echo
echo "5) Mark as read from B:"
curl -sS -X POST "${BASE_URL}/api/v1/chat/rooms/${ROOM_ID}/read" \
  -H "Authorization: Bearer ${TOKEN_B}" | jq .

echo
echo "6) Check websocket metrics:"
curl -sS "${BASE_URL}/debug/vars" | jq '{websocket_connections, websocket_events_sent_total, websocket_events_dropped_total}'
