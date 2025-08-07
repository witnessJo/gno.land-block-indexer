#!/bin/bash

echo "Cleaning up existing port-forward processes..."
pkill -f "kubectl port-forward" || true
sleep 2

kubectl delete -f ./containers-kube.yaml  --ignore-not-found=true

echo "Waiting for pods to be deleted..."
sleep 5

kubectl apply -f ./containers-kube.yaml

echo "Waiting for pods to be created..."
until kubectl get pod postgres-pod &> /dev/null; do
  sleep 1
done
until kubectl get pod redis-pod &> /dev/null; do
  sleep 1
done
until kubectl get pod localstack-pod &> /dev/null; do
  sleep 1
done

echo "Waiting for PostgreSQL pod to be ready..."
kubectl wait --for=condition=ready pod/postgres-pod --timeout=60s

echo "Waiting for Redis pod to be ready..."
kubectl wait --for=condition=ready pod/redis-pod --timeout=60s

echo "Waiting for LocalStack pod to be ready..."
kubectl wait --for=condition=ready pod/localstack-pod --timeout=120s

echo "Starting port forwarding..."
nohup kubectl port-forward pod/postgres-pod 5432:5432 > /dev/null 2>&1 &
nohup kubectl port-forward pod/redis-pod 6379:6379 > /dev/null 2>&1 &
nohup kubectl port-forward pod/localstack-pod 4566:4566 > /dev/null 2>&1 &

echo "Port forwarding started!"
echo "PostgreSQL: localhost:5432"
echo "Redis: localhost:6379"
echo "LocalStack: localhost:4566"
echo ""
echo "LocalStack endpoints:"
echo "  S3: http://localhost:4566"
echo "  DynamoDB: http://localhost:4566"
echo "  SQS: http://localhost:4566"
echo "  SNS: http://localhost:4566"
echo ""
