#!/usr/bin/env bash

echo "Creating docker container on port 27018"
docker run --name test-mongo -d -p 27018:27017 mongo
go test ./...
retcode=$?
echo "Tests exited with exit code: $?"

echo "Deleting container test-mongo"
docker -l error stop test-mongo
docker -l error rm test-mongo

exit $retcode
