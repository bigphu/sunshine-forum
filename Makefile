# Windows: chạy recipe bằng sh.exe của Git for Windows thay cho cmd.exe.
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

# Makefile của monorepo: stack Docker (compose.yml), database, và gọi sang
# backend/Makefile (Go) và frontend (npm). Lệnh riêng của Go: make -C backend <target>

-include .env

APP_DB_NAME ?= sunshine
APP_DB_USER ?= sunshine
WEB_PORT    ?= 5180
APP_PORT    ?= 8091

COMPOSE := docker compose
PSQL    := $(COMPOSE) exec postgres psql -U $(APP_DB_USER) -d $(APP_DB_NAME)

TAG      ?= dev
REGISTRY ?= sunshine-forum

.DEFAULT_GOAL := help
.PHONY: help init dev up down restart build logs logs-app logs-worker logs-web ps \
        psql db-reset seed generate check images

help:
	@echo Prerequisite: init
	@echo Dev         : dev = up + logs   "(web http://localhost:$(WEB_PORT), API http://localhost:$(APP_PORT)/api/docs, mail http://localhost:8026)"
	@echo Docker      : up down restart build logs logs-app logs-worker logs-web ps
	@echo DB          : psql seed db-reset CONFIRM=yes
	@echo Code        : generate check
	@echo Release     : images TAG=1.0.0 REGISTRY=registry.example.com
	@echo Go only     : make -C backend help

# --- Prerequisite ---
# Chạy một lần sau khi clone: tạo secrets và file env
init:
	sh deploy/scripts/init.sh
	@test -f .env               || cp .env.example .env
	@test -f backend/.env.local || cp backend/.env.local.example backend/.env.local

# --- Dev ---
dev: up logs

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

restart:
	$(COMPOSE) restart app worker web

build:
	$(COMPOSE) build app web

logs:
	$(COMPOSE) logs -f

logs-app:
	$(COMPOSE) logs -f app

logs-worker:
	$(COMPOSE) logs -f worker

logs-web:
	$(COMPOSE) logs -f web

ps:
	$(COMPOSE) ps

# --- DB ---
psql:
	$(PSQL)

# Dữ liệu demo (SEED_PASSWORD trong .env)
seed:
	$(COMPOSE) run --rm seed

# Xoá sạch DB rồi chạy lại migration
db-reset:
ifneq ($(CONFIRM),yes)
	$(error make db-reset CONFIRM=yes: this deletes every row)
endif
	$(COMPOSE) down -v
	$(COMPOSE) up -d

# --- Code ---
generate:
	$(MAKE) -C backend generate
	npm --prefix frontend run gen:api

check:
	$(MAKE) -C backend check
	npm --prefix frontend run check

# --- Release ---
images:
	docker build --target prod -t $(REGISTRY)/backend:$(TAG) backend
	docker build --target prod -t $(REGISTRY)/web:$(TAG) frontend
