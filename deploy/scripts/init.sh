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
	echo "created $file"
}

gen pg_password.txt 24
gen privacy_enc_key.txt 32
gen privacy_hmac_key.txt 32

# Khoá Resend (thư ở production) do người cài điền; để trống thì worker không gửi được thư
if [ ! -e secrets/resend_api_key.txt ]; then
	: > secrets/resend_api_key.txt
	echo "created secrets/resend_api_key.txt (empty: paste the Resend API key for production)"
fi

# Quyền: thư mục 700 nên người dùng khác trên máy không vào được; file 644 để container đọc
# được. Image backend chạy bằng user không phải root (distroless nonroot, uid 65532) và compose
# gắn file secret giữ nguyên quyền của host: file 600 thuộc user deploy làm migrate, app và
# worker không đọc được secret và dừng ngay khi khởi động.
chmod 700 secrets 2>/dev/null || true
chmod 644 secrets/*.txt 2>/dev/null || true
