#!/bin/bash

set -xe

docker build -t samos123/websu:latest .
[ ! "$(docker ps -a | grep mongo)" ] && docker run --name mongo -d mongo:4
docker run --cap-add SYS_ADMIN -p 8000:8000 -e MONGO_URI=mongodb://mongo:27017 --link mongo:mongo samos123/websu:latest
