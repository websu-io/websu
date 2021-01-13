#!/usr/bin/env bash

docker run --name test-mongo -d -p 27018:27017 mongo
go test ./...
retcode=$?
docker stop test-mongo
docker rm test-mongo

exit $retcode
