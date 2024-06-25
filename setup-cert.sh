#!/usr/bin/env bash

set -e

go install \
    'github.com/cloudflare/cfssl/cmd/cfssl' \
    'github.com/cloudflare/cfssl/cmd/cfssljson'
