#!/usr/bin/env bash

CERT_DIR=".cert"
CA_CERT="$CERT_DIR/ca.pem"
CA_KEY="$CERT_DIR/ca-key.pem"
SERVER_CERT="$CERT_DIR/server.pem"
SERVER_KEY="$CERT_DIR/server-key.pem"

# 함수 정의: 인증서와 키 파일 확인 및 매칭 확인
check_cert_key() {
  local cn="$1"
  local cert_file="$CERT_DIR/$cn-client.pem"
  local key_file="$CERT_DIR/$cn-client-key.pem"

  echo "=== $cn 클라이언트 인증서 확인 ==="
  openssl x509 -in "$cert_file" -text -noout

  echo "=== $cn 클라이언트 키 파일 확인 ==="
  openssl rsa -in "$key_file" -check || openssl ec -in "$key_file" -check

  echo "=== $cn 클라이언트 인증서와 키 파일 매칭 확인 ==="
  CERT_MODULUS=$(openssl x509 -noout -modulus -in "$cert_file" | openssl md5)
  KEY_MODULUS=$(openssl rsa -noout -modulus -in "$key_file" | openssl md5)

  if [ "$CERT_MODULUS" == "$KEY_MODULUS" ]; then
    echo "$cn 클라이언트 인증서와 키 파일이 매칭됩니다."
  else
    echo "$cn 클라이언트 인증서와 키 파일이 일치하지 않습니다!"
  fi

  echo "=== $cn 클라이언트 인증서 검증 ==="
  openssl verify -CAfile "$CA_CERT" "$cert_file"
}

# CA 인증서 확인
echo "=== CA 인증서 확인 ==="
openssl x509 -in "$CA_CERT" -text -noout

# CA 키 파일 확인
echo "=== CA 키 파일 확인 ==="
openssl rsa -in "$CA_KEY" -check || openssl ec -in "$CA_KEY" -check

# 서버 인증서와 키 파일 확인 및 매칭 확인
echo "=== 서버 인증서 확인 ==="
openssl x509 -in "$SERVER_CERT" -text -noout

echo "=== 서버 키 파일 확인 ==="
openssl rsa -in "$SERVER_KEY" -check || openssl ec -in "$SERVER_KEY" -check

echo "=== 서버 인증서와 키 파일 매칭 확인 ==="
CERT_MODULUS=$(openssl x509 -noout -modulus -in "$SERVER_CERT" | openssl md5)
KEY_MODULUS=$(openssl rsa -noout -modulus -in "$SERVER_KEY" | openssl md5)

if [ "$CERT_MODULUS" == "$KEY_MODULUS" ]; then
  echo "서버 인증서와 키 파일이 매칭됩니다."
else
  echo "서버 인증서와 키 파일이 일치하지 않습니다!"
fi

echo "=== 서버 인증서 검증 ==="
openssl verify -CAfile "$CA_CERT" "$SERVER_CERT"

# 각 클라이언트 인증서들 확인 및 매칭 확인
CN=("root" "nobody")  # 클라이언트 인증서들의 CN 리스트
for cn in "${CN[@]}"; do
  check_cert_key "$cn"
done
