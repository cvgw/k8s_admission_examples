#!/usr/bin/env bash
set -e

curl -L https://pkg.cfssl.org/R1.2/cfssl_linux-amd64 -o cfssl
chmod +x cfssl
sudo ln -s ${pwd}/cfssl /usr/bin/cfssl

curl -L https://pkg.cfssl.org/R1.2/cfssljson_linux-amd64 -o cfssljson
chmod +x cfssljson
sudo ln -s ${pwd}/cfssljson /usr/bin/cfssljson
