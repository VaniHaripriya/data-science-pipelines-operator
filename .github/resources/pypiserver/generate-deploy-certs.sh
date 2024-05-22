#!/usr/bin/env bash

set -ex

# Define variables
path=certs
pypiserver_namespace="test-pypiserver"

# Create directory if it doesn't exist
mkdir -p "${path}"

# Create Key and CSR
openssl req -newkey rsa:4096 -nodes -keyout "${path}/domain.key" -out "${path}/domain.csr" -subj "/C=XX/CN=nginx-service.test-pypiserver.svc.cluster.local" 2>/dev/null

# Creating a CA-Signed Certificate With Our Own CA

# Create a Self-Signed Root CA
openssl req \
  -x509 -sha256 \
  -days 3650 \
  -newkey rsa:4096 \
  -keyout "${path}/rootCA.key" \
  -nodes \
  -out "${path}/rootCA.crt" \
  -subj "/C=XX/CN=nginx-service.test-pypiserver.svc.cluster.local" 2>/dev/null

# Sign Our CSR With Root CA
# As a result, the CA-signed certificate will be in the domain.crt file.
openssl x509 -req -days 3650 -CA "${path}/rootCA.crt" -CAkey "${path}/rootCA.key" -in "${path}/domain.csr" -out "${path}/domain.crt" -CAcreateserial -extfile certs/manual-certs/domain.ext 2>/dev/null
