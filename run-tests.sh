#!/usr/bin/env bash

set -xeE

testURL() {
  curl --silent --show-error --fail "$@"
}

if [ "$1" = "unit" ]; then
  echo "Creating docker container on port 27018"
  docker run --name test-mongo -d -p 27018:27017 mongo
  trap "docker stop test-mongo && docker rm test-mongo" EXIT SIGINT
  go test ./...
elif [ "$1" = "integration" ]; then
  ./test-docker.sh
  echo "Sleeping 20 seconds to make sure all services are up"
  sleep 20
  trap "docker-compose logs websu-api lighthouse-server" ERR
  testURL http://localhost:8000/
  testURL http://localhost:8000/reports
  testURL -d '{"url": "https://www.google.com"}' localhost:8000/reports
  echo "Integration tests passed"
  # Adding a basic and premium location
  testURL -d '{"name":"basic1","display_name":"Basic1","address":"lighthouse-server:50051","secure":false,"order":0,"premium":false}' http://localhost:8000/locations
  testURL -d '{"name":"premium","display_name":"Premium","address":"lighthouse-server:50051","secure":false,"order":0,"premium":true}' http://localhost:8000/locations
else
  echo "Please run with './run-tests.sh unit' or './run-tests.sh integration'"
fi
