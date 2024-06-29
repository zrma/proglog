#!/usr/bin/env bash

CERT_DIR=".cert"
CA_CERT=$CERT_DIR"/ca.pem"
CA_KEY=$CERT_DIR"/ca-key.pem"
SERVER_CERT=$CERT_DIR"/server.pem"
SERVER_KEY=$CERT_DIR"/server-key.pem"
CLIENT_CERT=$CERT_DIR"/client.pem"
CLIENT_KEY=$CERT_DIR"/client-key.pem"

echo "=== CA 인증서 확인 ==="
openssl x509 -in $CA_CERT -text -noout

echo "=== CA 키 파일 확인 ==="
openssl rsa -in $CA_KEY -check || openssl ec -in $CA_KEY -check

echo "=== 서버 인증서 확인 ==="
openssl x509 -in $SERVER_CERT -text -noout

echo "=== 서버 키 파일 확인 ==="
openssl rsa -in $SERVER_KEY -check || openssl ec -in $SERVER_KEY -check

echo "=== 서버 인증서와 키 파일 매칭 확인 ==="
CERT_MODULUS=$(openssl x509 -noout -modulus -in $SERVER_CERT | openssl md5)
KEY_MODULUS=$(openssl rsa -noout -modulus -in $SERVER_KEY | openssl md5)

if [ "$CERT_MODULUS" == "$KEY_MODULUS" ]; then
  echo "서버 인증서와 키 파일이 매칭됩니다."
else
  echo "서버 인증서와 키 파일이 일치하지 않습니다!"
fi

echo "=== 서버 인증서 검증 ==="
openssl verify -CAfile $CA_CERT $SERVER_CERT

echo "=== 클라이언트 인증서 확인 ==="
openssl x509 -in $CLIENT_CERT -text -noout

echo "=== 클라이언트 키 파일 확인 ==="
openssl rsa -in $CLIENT_KEY -check || openssl ec -in $CLIENT_KEY -check

echo "=== 클라이언트 인증서와 키 파일 매칭 확인 ==="
CERT_MODULUS=$(openssl x509 -noout -modulus -in $CLIENT_CERT | openssl md5)
KEY_MODULUS=$(openssl rsa -noout -modulus -in $CLIENT_KEY | openssl md5)

if [ "$CERT_MODULUS" == "$KEY_MODULUS" ]; then
  echo "클라이언트 인증서와 키 파일이 매칭됩니다."
else
  echo "클라이언트 인증서와 키 파일이 일치하지 않습니다!"
fi

echo "=== 클라이언트 인증서 검증 ==="
openssl verify -CAfile $CA_CERT $CLIENT_CERT
