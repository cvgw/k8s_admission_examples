#!/usr/bin/env bash
set -e

./scripts/k8s_cert.sh
kubectl create -f ./kubernetes/deployment.yaml
kubectl create -f ./kubernetes/svc.yaml
kubectl create -f ./kubernetes/config-map.yaml
./scripts/create_webhook_config.sh
