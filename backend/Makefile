# Windows: chạy recipe bằng sh.exe của Git for Windows thay cho cmd.exe.
# Thêm Git/bin, Git/usr/bin vào PATH vì lệnh đơn giản được make gọi thẳng, không qua SHELL
ifeq ($(MAKE_HOST),Windows32)
empty :=
space := $(empty) $(empty)
GIT_ROOT := $(shell git --exec-path)/../../..
ifneq ($(wildcard $(subst $(space),\ ,$(GIT_ROOT)/bin/sh.exe)),)
SHELL := $(GIT_ROOT)/bin/sh.exe
export PATH := $(PATH);$(GIT_ROOT)/bin;$(GIT_ROOT)/usr/bin
export MSYS_NO_PATHCONV := 1
endif
endif

# Lệnh Go của backend. Stack Docker, database, init: Makefile ở thư mục gốc

# Env cho Go chạy trên host: export đúng các biến khai báo trong .env.local
-include .env.local
LOCAL_ENV_KEYS := $(shell sed -n 's/^\([A-Z_][A-Z0-9_]*\)=.*/\1/p' .env.local 2>/dev/null)
# Đường dẫn file trong .env.local tính từ backend/; go test chạy mỗi package trong thư
# mục của nó, nên đổi sang đường dẫn tuyệt đối trước khi export
PATH_KEYS := $(filter %_FILE,$(LOCAL_ENV_KEYS))
$(foreach k,$(PATH_KEYS),$(if $($(k)),$(eval $(k) := $(abspath $($(k))))))
export $(LOCAL_ENV_KEYS)

# Version của tool chạy bằng Docker, CI cũng dùng
export SQLC_VERSION := 1.31.1
GOLANGCI_VERSION    := v2.14.0

.DEFAULT_GOAL := help
.PHONY: help migrate migrate-new seed generate generate-check sqlc sqlc-vet sqlc-compile \
        run-api run-worker bin test test-db fmt fmt-check vet lint tidy check

help:
	@echo Migrations  : migrate migrate-new name=... seed
	@echo Codegen     : generate generate-check sqlc sqlc-vet sqlc-compile
	@echo Go          : run-api run-worker bin test test-db fmt fmt-check vet lint tidy check
	@echo Docker, DB  : make -C .. help

# --- Migrations ---
# Chạy trên host với .env.local. Trong Docker thì `make up` tự chạy migrate
migrate:
	go run ./cmd/migrate

# Tạo file migration mới, đặt tên <module>_<thay đổi>
migrate-new:
ifndef name
	$(error usage: make migrate-new name=content_reactions)
endif
	go tool goose -s -dir migrations create $(name) sql

# Dữ liệu demo (cần SEED_PASSWORD trong .env.local)
seed:
	go run ./cmd/seed

# --- Codegen ---
# Sinh code từ query .sql (sqlc) và openapi.yaml (oapi-codegen). Code sinh ra được commit
generate: sqlc
	go generate ./...

# Cho CI: sinh lại code, fail nếu có gì khác bản đã commit
generate-check: generate
	@test -z "$$(git status --porcelain -- .)" || (echo "code sinh ra chưa cập nhật, chạy make generate rồi commit:"; git status --porcelain -- .; exit 1)

# sqlc chạy bằng Docker; script bỏ qua module chưa có file query
sqlc:
	sh scripts/sqlc.sh generate

sqlc-vet:
	sh scripts/sqlc.sh vet

sqlc-compile:
	sh scripts/sqlc.sh compile

# --- Go (chạy trên host) ---
run-api:
	go run ./cmd/server

run-worker:
	go run ./cmd/worker

# Build server, worker, migrate, seed vào bin/
bin:
	go build -o bin/ ./cmd/...

# Test nhanh: test DB (*_db_test.go) tự bỏ qua khi không có Docker
test:
	go test ./...

# Mọi test, kể cả test DB; -p 1 vì nhiều container test cùng lúc dễ lỗi trên Windows
test-db:
	CI=true go test -p 1 ./...

fmt:
	go fmt ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "chưa gofmt:"; gofmt -l .; exit 1)

vet:
	go vet ./...

# golangci-lint bằng Docker, config ở .golangci.yml (depguard giữ ranh giới module)
lint:
	docker run --rm -v "$(CURDIR):/app" -w /app \
		-v sunshine_gomod:/go/pkg/mod -v sunshine_golangci:/root/.cache \
		golangci/golangci-lint:$(GOLANGCI_VERSION) golangci-lint run

tidy:
	go mod tidy

# Chạy trước khi commit
check: sqlc-compile fmt-check vet lint test
