#!/bin/bash
set -e

echo Creating K8s resources

kubectl create serviceaccount test
kubectl create clusterrolebinding test-cluster-admin --clusterrole cluster-admin --serviceaccount default:test
./substitute_config_map.sh | kubectl create -f -
kubectl create -f ./kubernetes/deployment.yml
kubectl create -f ./kubernetes/admission-config.yml
