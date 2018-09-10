#!/usr/bin/env bash
set -e

./scripts/k8s_cert.sh
kubectl create -f ./kubernetes/deployment.yml
kubectl create -f ./kubernetes/svc.yml
./scripts/create_webhook_config.sh
