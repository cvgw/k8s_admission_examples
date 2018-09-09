#!/usr/bin/env bash
set -e

kubectl create serviceaccount demo-init
kubectl create -f ./kubernetes/rbac.yaml
kubectl create -f ./kubernetes/role-binding.yaml
kubectl create -f ./kubernetes/deployment.yaml
kubectl create -f ./kubernetes/admission-config.yaml
