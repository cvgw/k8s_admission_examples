#!/usr/bin/env bash
set -e

kubectl delete csr/config-mutator.default
kubectl delete deployments/config-mutator-deployment
kubectl delete configmaps/mutator-config
kubectl delete mutatingwebhookconfiguration/config-mutator-cfg
kubectl delete svc/config-mutator
