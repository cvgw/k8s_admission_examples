#!/usr/bin/env bash
set -e

kubectl delete initializerconfiguration/annotator
kubectl delete deployments/annotator
kubectl delete rolebindings/initialize-deployments
kubectl delete role/deployment-initializer
kubectl delete serviceaccounts/demo-init
