#!/bin/bash
set -e
# login
TOKEN=$(curl -s -X POST http://localhost:8001/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"admin@opsway.eu","password":"pass"}' -c cookies.txt | jq -r '.user.id')

# create monitor
curl -s -X POST http://localhost:8001/v1/teams/1/monitors -H 'Content-Type: application/json' -b cookies.txt -d '{
  "name": "Test Monitor",
  "settings": {
    "method": "GET",
    "url": "https://example.com",
    "frequencySeconds": 60,
    "headers": [],
    "body": {"type": "NONE"},
    "tls": {"enabled": false}
  },
  "assertions": []
}'

# create active maintenance
START=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
END=$(date -u -d "+1 hour" +"%Y-%m-%dT%H:%M:%SZ")
curl -s -X POST http://localhost:8001/v1/teams/1/maintenance -H 'Content-Type: application/json' -b cookies.txt -d '{
  "title": "Test Maintenance",
  "settings": {
    "startAt": "'$START'",
    "endAt": "'$END'"
  }
}'

# get monitor ID
MONITOR_ID=$(curl -s -X GET http://localhost:8001/v1/teams/1/monitors -b cookies.txt | jq -r '.monitors[0].id')
echo "Monitor ID: $MONITOR_ID"

# set monitor state to ACTIVE
curl -s -X PUT http://localhost:8001/v1/teams/1/monitors/$MONITOR_ID/state -H 'Content-Type: application/json' -b cookies.txt -d '{"state": "ACTIVE"}' -w "\nHTTP Status: %{http_code}\n"
