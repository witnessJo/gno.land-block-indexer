#!/bin/bash

docker compose -f ./containers-compose.yaml down --remove-orphans
docker compose -f ./containers-compose.yaml up -d
