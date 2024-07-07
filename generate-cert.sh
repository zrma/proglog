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

cfssl gencert \
  -ca=$TARGET_DIR/ca.pem \
  -ca-key=$TARGET_DIR/ca-key.pem \
  -config=tools/certs/ca-config.json \
  -profile=client \
  -cn="root" \
  tools/certs/client-csr.json | cfssljson -bare $TARGET_DIR/root-client

cfssl gencert \
  -ca=$TARGET_DIR/ca.pem \
  -ca-key=$TARGET_DIR/ca-key.pem \
  -config=tools/certs/ca-config.json \
  -profile=client \
  -cn="nobody" \
  tools/certs/client-csr.json | cfssljson -bare $TARGET_DIR/nobody-client

  cp tools/acl/model.conf $TARGET_DIR/acl-model.conf
  cp tools/acl/policy.csv $TARGET_DIR/acl-policy.csv
