#!/usr/bin/env bash

set -e

TARGET_DIR=.cert

cfssl gencert \
  -initca tools/certs/ca-csr.json | cfssljson -bare $TARGET_DIR/ca

cfssl gencert \
  -ca=$TARGET_DIR/ca.pem \
  -ca-key=$TARGET_DIR/ca-key.pem \
  -config=tools/certs/ca-config.json \
  -profile=server \
  tools/certs/server-csr.json | cfssljson -bare $TARGET_DIR/server
