#!/bin/sh
# Tạo secrets cho dev (chạy một lần sau khi clone: make init). Không ghi đè file đã có:
# đổi khoá mã hoá làm dữ liệu cũ không đọc được.
#   deploy/secrets/pg_password.txt        mật khẩu Postgres của app
#   deploy/secrets/privacy_enc_key.txt    AES-256 (32 byte, base64) mã hoá email, tên, căn hộ
#   deploy/secrets/privacy_hmac_key.txt   HMAC (32 byte, base64) tra cứu email, mã ẩn danh
set -eu
cd "$(dirname "$0")/.."
mkdir -p secrets
chmod 700 secrets 2>/dev/null || true

gen() {
	file="secrets/$1"
	if [ -s "$file" ]; then
		echo "keep $file"
		return
	fi
	if command -v openssl >/dev/null 2>&1; then
		openssl rand -base64 "$2" | tr -d '\r\n' > "$file"
	else
		head -c "$2" /dev/urandom | base64 | tr -d '\r\n' > "$file"
	fi
	chmod 600 "$file" 2>/dev/null || true
	echo "created $file"
}

gen pg_password.txt 24
gen privacy_enc_key.txt 32
gen privacy_hmac_key.txt 32
