#!/usr/bin/env bash
set -e

echo Deleting K8s resources

kubectl delete initializerconfiguration.admissionregistration.k8s.io/vault
kubectl delete deployments vault-initializer
kubectl delete configmaps vault-initializer-config
kubectl delete clusterrolebindings test-cluster-admin
kubectl delete serviceaccounts test
