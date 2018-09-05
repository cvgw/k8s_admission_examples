# Vault Initializer
## Description
Demo initializer to show modifying an initializing k8s resource with data pulled from vault

## Requirements
* deployed and unsealed vault instance
* kubenetes api-server with admission plugin `Initializers` added
* kubenetes api-server with api-resource type `admissionregistration.k8s.io/v1alpha1`

## Setup
Create a new policy in vault named `mypolicy` using the included file `vault_policy.hcl`

`vault policy write mypolicy vault_policy.hcl`

Create a token for `mypolicy`

`vault token create --policy=mypolicy`

Copy the token value and export it in your shell as `VAULT_TOKEN`

`export VAULT_TOKEN=xxxxx`

Write the test value to vault

`vault write /secret/foo --value=TEST_VALUE`

Build the vault initializer docker image

`docker build -t vault-initializer .`

Create the kubernetes resources

`./init_resources.sh`

## Usage
Create the test deployment

`kubectl create -f ./kubernetes/sleep.yml`

Verify that the deployment was annotated with the value from vault

````
kubectl get deployments/sleep -o yaml

annotations:
deployment.kubernetes.io/revision: "1"
initializer.cvgw.me/vault: "true"
vault-initializer: --value-TEST_VALUE
````

## Cleanup
Delete the test deployment

`kubectl delete deployments/sleep`

Delete the vault-initializer resources

`./delete_resources.sh`
