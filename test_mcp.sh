#!/usr/bin/env bash
# Default matches control-plane server.bind (:8123). Override: BASE_URL=http://127.0.0.1:8080 ./test_mcp.sh

BASE_URL="${BASE_URL:-http://127.0.0.1:8123}"
API_KEY="${PLURIBUS_API_KEY:-}"   # optional; same env name as the server

HDR=(-H "Content-Type: application/json")
if [ -n "$API_KEY" ]; then
  HDR+=(-H "X-API-Key: $API_KEY")
fi

echo "== STEP 1: healthz =="
curl -s "$BASE_URL/healthz"
echo -e "\n"

echo "== STEP 2: MCP endpoint probe =="
curl -s "$BASE_URL/v1/mcp"
echo -e "\n"

echo "== STEP 3: tools/list =="
curl -s "$BASE_URL/v1/mcp" "${HDR[@]}" -d '{
  "jsonrpc":"2.0",
  "id":1,
  "method":"tools/list",
  "params":{}
}'
echo -e "\n"

echo "== STEP 4: wakeup_context =="
curl -s "$BASE_URL/v1/mcp" "${HDR[@]}" -d '{
  "jsonrpc":"2.0",
  "id":2,
  "method":"tools/call",
  "params":{
    "name":"wakeup_context",
    "arguments":{}
  }
}'
echo -e "\n"

echo "== STEP 5: wakeup_context (limit test) =="
curl -s "$BASE_URL/v1/mcp" "${HDR[@]}" -d '{
  "jsonrpc":"2.0",
  "id":3,
  "method":"tools/call",
  "params":{
    "name":"wakeup_context",
    "arguments":{
      "max_governing_total":1
    }
  }
}'
echo -e "\n"

echo "== DONE =="
