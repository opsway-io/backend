#!/bin/bash
cd /home/teis/opsway/backend
export OPSWAY_REST_PORT=8015
go run main.go api --config config.yaml &
PID=$!
sleep 5

echo "--- Registering user ---"
curl -s -v -X POST http://localhost:8015/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name": "New Test User", "email": "testnew2@opsway.eu", "password": "password123"}'

kill $PID
