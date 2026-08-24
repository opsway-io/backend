#!/bin/bash
cd /home/teis/opsway/backend
go run main.go api --config config.yaml &
PID=$!
sleep 5

echo "--- Registering user ---"
curl -s -X POST http://localhost:8005/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name": "Test User", "email": "test2@opsway.eu", "password": "password123"}'

echo -e "\n\n--- Logging in ---"
curl -s -v -X POST http://localhost:8005/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "test2@opsway.eu", "password": "password123"}'

kill $PID
