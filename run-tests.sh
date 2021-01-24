#!/usr/bin/env bash

if [ $1 = "unit" ]; then
  echo "Creating docker container on port 27018"
  docker run --name test-mongo -d -p 27018:27017 mongo
  go test ./...
  retcode=$?
  echo "Tests exited with exit code: $?"

  echo "Deleting container test-mongo"
  docker -l error stop test-mongo
  docker -l error rm test-mongo

  exit $retcode

elif [ $1 = "integration" ]; then
  ./test-docker.sh
  echo "Sleeping 10 seconds to make sure all services are up"
  sleep 10
  curl -f http://localhost:8000
  curl -f http://localhost:8000/reports
  curl -f -d '{"url": "https://www.google.com"}' localhost:8000/reports
  echo "Integration tests passed"

else
    echo "Please run with './run-tests.sh unit' or './run-tests.sh integration'"
fi
