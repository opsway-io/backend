#!/usr/bin/env bash
set -e

echo "Starting regular opsway environment..."
docker-compose -f ../../docker-compose.yaml up -d

echo "Waiting for services to be ready..."
sleep 15

echo "Starting Pumba Chaos Testing container..."
docker-compose -f ../../docker-compose.chaos.yaml up -d

echo "Running health checks during chaos for 1 minute..."
for i in {1..12}; do
    echo "Check $i / 12"
    # Basic check against the API (it might fail if DB is down, but should recover)
    curl -s -f http://localhost:8001/v1/healthz || echo "API is temporarily unreachable (Expected during chaos)"
    sleep 5
done

echo "Stopping Pumba Chaos..."
docker-compose -f ../../docker-compose.chaos.yaml down

echo "Waiting for services to recover..."
sleep 20

echo "Verifying API is fully recovered..."
if curl -s -f http://localhost:8001/v1/healthz; then
    echo "Chaos test completed successfully: API recovered."
else
    echo "Chaos test failed: API did not recover."
    exit 1
fi
