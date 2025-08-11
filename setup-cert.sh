#!/usr/bin/env bash

set -e

go install \
    'github.com/cloudflare/cfssl/cmd/cfssl@v1.6.5' \
    'github.com/cloudflare/cfssl/cmd/cfssljson@v1.6.5'
