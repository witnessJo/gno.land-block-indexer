#!/bin/bash

# stop infra containers

# start infra containers
kubectl apply -f ./containers-infra.yaml

echo "Waiting for PostgreSQL pod to be ready..."
kubectl wait --for=condition=ready pod/postgres-pod --timeout=60s

echo "Waiting for Redis pod to be ready..."
kubectl wait --for=condition=ready pod/redis-pod --timeout=60s

pkill -f "kubectl port-forward"
kubectl port-forward postgres-pod 5432:5432 &
kubectl port-forward redis-pod 6379:6379 &


