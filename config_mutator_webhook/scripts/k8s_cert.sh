#!/usr/bin/env bash
set -e

SVC_NAME=config-mutator
POD_NAME=config-mutator
NAMESPACE=default

cat <<EOF | cfssl genkey - | cfssljson -bare server
{
  "hosts": [
    "${SVC_NAME}.${NAMESPACE}.svc.cluster.local",
    "${POD_NAME}.${NAMESPACE}.pod.cluster.local",
    "${SVC_NAME}.${NAMESPACE}.svc"
  ],
  "CN": "${POD_NAME}.${NAMESPACE}.pod.cluster.local",
  "key": {
    "algo": "ecdsa",
    "size": 256
  }
}
EOF

cat <<EOF | kubectl create -f -
apiVersion: certificates.k8s.io/v1beta1
kind: CertificateSigningRequest
metadata:
  name: ${SVC_NAME}.${NAMESPACE}
spec:
  groups:
  - system:authenticated
  request: $(cat server.csr | base64 | tr -d '\n')
  usages:
  - digital signature
  - key encipherment
  - server auth
EOF

kubectl certificate approve ${SVC_NAME}.${NAMESPACE}

kubectl get csr ${SVC_NAME}.${NAMESPACE} -o jsonpath='{.status.certificate}' \
    | base64 --decode > server.crt

kubectl create configmap mutator-config --from-file=server.crt --from-file=server-key.pem

rm server.crt server.csr server-key.pem
