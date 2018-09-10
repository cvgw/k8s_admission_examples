#!/usr/bin/env bash
set -e

CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n')

cat <<EOF | kubectl create -f -
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: config-mutator-cfg
  labels:
    app: config-mutator
webhooks:
  - name: config-mutator.cvgw.me
    clientConfig:
      service:
        name: config-mutator
        namespace: default
        path: "/mutate"
      caBundle: |
        ${CA_BUNDLE}
    failurePolicy: Fail
    rules:
      - apiGroups:
          - ""
        apiVersions:
          - v1
        operations:
          - "CREATE"
        resources:
          - "pods"
EOF
