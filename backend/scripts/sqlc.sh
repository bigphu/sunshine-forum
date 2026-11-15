#!/bin/sh
# Chạy sqlc (generate | compile | vet) bằng Docker, chỉ với các module đã có file query
# (sqlc báo lỗi khi thư mục queries còn trống). Makefile gọi script này; trên Windows
# chạy trong Git Bash.
set -eu
cmd="${1:-generate}"
version="${SQLC_VERSION:-1.31.1}"
tmp=".sqlc.tmp.yml"

# Block đầu giữ anchor &go_defaults nên luôn được giữ; block khác chỉ giữ khi có query
awk '
  BEGIN { inSql = 0; n = 0 }
  /^sql:/ { print; inSql = 1; next }
  !inSql { print; next }
  /^  - engine:/ { n++ }
  { block[n] = block[n] $0 "\n" }
  END {
    for (i = 1; i <= n; i++) {
      if (match(block[i], /queries: "[^"]+"/)) {
        dir = substr(block[i], RSTART + 10, RLENGTH - 11)
        cmd = "ls " dir "/*.sql >/dev/null 2>&1"
        if (i == 1 || system(cmd) == 0) printf "%s", block[i]
      }
    }
  }
' sqlc.yml > "$tmp"

src="$(pwd -W 2>/dev/null || pwd)"
MSYS_NO_PATHCONV=1 docker run --rm -v "$src:/src" -w /src "sqlc/sqlc:$version" -f "$tmp" "$cmd"
status=$?
rm -f "$tmp"
exit $status
