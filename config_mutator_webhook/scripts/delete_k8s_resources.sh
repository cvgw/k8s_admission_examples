#!/usr/bin/env bash
set -e

kubectl delete csr/config-mutator.default deployments/config-mutator-deployment configmaps/mutator-config mutatingwebhookconfiguration/config-mutator-cfg svc/config-mutator configmaps/mutating-demo-configmap
