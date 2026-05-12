APP_NAME := server
CMD_SERVER := ./cmd/server
BIN_DIR := bin
WEB_DIR := web
WEB_DIST_DIR := $(WEB_DIR)/dist
WEB_EMBED_DIR := internal/webui/dist
GO ?= go
PNPM ?= pnpm
THIS_FILE := $(lastword $(MAKEFILE_LIST))

ifeq ($(OS),Windows_NT)
HOST_EXE_EXT := .exe
else
HOST_EXE_EXT :=
endif

HOST_OUTPUT := $(BIN_DIR)/$(APP_NAME)$(HOST_EXE_EXT)

.DEFAULT_GOAL := help

.PHONY: help check-go check-pnpm prepare-bin prepare-web-embed-dir web-install web-build web-sync build-web test build build-windows-amd64 build-linux-amd64 build-linux-arm64 build-linux-armv7 build-all clean

help: ## Show available targets. Web targets skip until web/package.json or web/dist exists.
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "$$lines = Get-Content '$(THIS_FILE)'; foreach ($$line in $$lines) { if ($$line -match '^([A-Za-z0-9_.-]+):.*## (.+)$$') { '{0,-24} {1}' -f $$matches[1], $$matches[2] } }"
else
	@awk 'match($$0, /^([A-Za-z0-9_.-]+):.*## (.*)$$/, m) { printf "%-24s %s\n", m[1], m[2] }' "$(THIS_FILE)"
endif

check-go: ## Verify the Go toolchain is available.
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "Get-Command '$(GO)' | Out-Null"
else
	@command -v "$(GO)" >/dev/null 2>&1 || { echo "$(GO) not found in PATH"; exit 1; }
endif

check-pnpm: ## Verify pnpm is available when the web project is initialized.
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "if (-not (Test-Path '$(WEB_DIR)/package.json')) { Write-Host 'web project not initialized, skip pnpm check'; exit 0 }; Get-Command '$(PNPM)' | Out-Null"
else
	@if [ ! -f "$(WEB_DIR)/package.json" ]; then echo "web project not initialized, skip pnpm check"; exit 0; fi
	@command -v "$(PNPM)" >/dev/null 2>&1 || { echo "$(PNPM) not found in PATH"; exit 1; }
endif

prepare-bin:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path '$(BIN_DIR)' | Out-Null"
else
	@mkdir -p "$(BIN_DIR)"
endif

prepare-web-embed-dir:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path '$(WEB_EMBED_DIR)' | Out-Null"
else
	@mkdir -p "$(WEB_EMBED_DIR)"
endif

web-install: check-pnpm ## Install web dependencies with pnpm when web/package.json exists.
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "if (-not (Test-Path '$(WEB_DIR)/package.json')) { Write-Host 'web project not initialized, skip'; exit 0 }; Set-Location '$(WEB_DIR)'; & '$(PNPM)' install --frozen-lockfile"
else
	@if [ ! -f "$(WEB_DIR)/package.json" ]; then echo "web project not initialized, skip"; exit 0; fi
	@cd "$(WEB_DIR)" && "$(PNPM)" install --frozen-lockfile
endif

web-build: check-pnpm ## Build the web app when web/package.json exists.
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "if (-not (Test-Path '$(WEB_DIR)/package.json')) { Write-Host 'web project not initialized, skip'; exit 0 }; Set-Location '$(WEB_DIR)'; & '$(PNPM)' build"
else
	@if [ ! -f "$(WEB_DIR)/package.json" ]; then echo "web project not initialized, skip"; exit 0; fi
	@cd "$(WEB_DIR)" && "$(PNPM)" build
endif

web-sync: ## Copy web/dist into internal/webui/dist when web/dist/index.html exists.
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "if (-not (Test-Path '$(WEB_DIST_DIR)/index.html')) { Write-Host 'web dist not found, skip embed sync'; exit 0 }; New-Item -ItemType Directory -Force -Path '$(WEB_EMBED_DIR)' | Out-Null; Get-ChildItem -Force '$(WEB_EMBED_DIR)' -ErrorAction SilentlyContinue | Where-Object { $$_.Name -ne '.gitkeep' } | Remove-Item -Recurse -Force; Copy-Item -Path '$(WEB_DIST_DIR)/*' -Destination '$(WEB_EMBED_DIR)' -Recurse -Force"
else
	@if [ ! -f "$(WEB_DIST_DIR)/index.html" ]; then echo "web dist not found, skip embed sync"; exit 0; fi
	@mkdir -p "$(WEB_EMBED_DIR)"
	@find "$(WEB_EMBED_DIR)" -mindepth 1 ! -name '.gitkeep' -exec rm -rf {} +
	@cp -R "$(WEB_DIST_DIR)"/. "$(WEB_EMBED_DIR)/"
endif

# Shared frontend build step used by all build targets
build-web:
	@$(MAKE) --no-print-directory web-build
	@$(MAKE) --no-print-directory web-sync

test: ## Run the Go test suite.
	@$(MAKE) --no-print-directory check-go
	@"$(GO)" test ./...

build: ## Build the server for the current host platform after web build and tests.
	@$(MAKE) --no-print-directory check-go
	@$(MAKE) --no-print-directory build-web
	@$(MAKE) --no-print-directory test
	@$(MAKE) --no-print-directory prepare-bin
	@"$(GO)" build -o "$(HOST_OUTPUT)" "$(CMD_SERVER)"

build-windows-amd64: ## Cross-compile the server for Windows amd64 (includes frontend).
	@$(MAKE) --no-print-directory check-go
	@$(MAKE) --no-print-directory build-web
	@$(MAKE) --no-print-directory prepare-bin
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "$$env:CGO_ENABLED='0'; $$env:GOOS='windows'; $$env:GOARCH='amd64'; & '$(GO)' build -o '$(BIN_DIR)/$(APP_NAME)-windows-amd64.exe' '$(CMD_SERVER)'"
else
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 "$(GO)" build -o "$(BIN_DIR)/$(APP_NAME)-windows-amd64.exe" "$(CMD_SERVER)"
endif

build-linux-amd64: ## Cross-compile the server for Linux amd64 (includes frontend).
	@$(MAKE) --no-print-directory check-go
	@$(MAKE) --no-print-directory build-web
	@$(MAKE) --no-print-directory prepare-bin
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "$$env:CGO_ENABLED='0'; $$env:GOOS='linux'; $$env:GOARCH='amd64'; & '$(GO)' build -o '$(BIN_DIR)/$(APP_NAME)-linux-amd64' '$(CMD_SERVER)'"
else
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 "$(GO)" build -o "$(BIN_DIR)/$(APP_NAME)-linux-amd64" "$(CMD_SERVER)"
endif

build-linux-arm64: ## Cross-compile the server for Linux arm64 (includes frontend).
	@$(MAKE) --no-print-directory check-go
	@$(MAKE) --no-print-directory build-web
	@$(MAKE) --no-print-directory prepare-bin
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "$$env:CGO_ENABLED='0'; $$env:GOOS='linux'; $$env:GOARCH='arm64'; & '$(GO)' build -o '$(BIN_DIR)/$(APP_NAME)-linux-arm64' '$(CMD_SERVER)'"
else
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 "$(GO)" build -o "$(BIN_DIR)/$(APP_NAME)-linux-arm64" "$(CMD_SERVER)"
endif

build-linux-armv7: ## Cross-compile the server for Linux armv7 (includes frontend).
	@$(MAKE) --no-print-directory check-go
	@$(MAKE) --no-print-directory build-web
	@$(MAKE) --no-print-directory prepare-bin
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "$$env:CGO_ENABLED='0'; $$env:GOOS='linux'; $$env:GOARCH='arm'; $$env:GOARM='7'; & '$(GO)' build -o '$(BIN_DIR)/$(APP_NAME)-linux-armv7' '$(CMD_SERVER)'"
else
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 "$(GO)" build -o "$(BIN_DIR)/$(APP_NAME)-linux-armv7" "$(CMD_SERVER)"
endif

build-all: ## Build the server for Windows amd64, Linux amd64, Linux arm64, and Linux armv7.
	@$(MAKE) --no-print-directory check-go
	@$(MAKE) --no-print-directory build-web
	@$(MAKE) --no-print-directory test
	@$(MAKE) --no-print-directory build-windows-amd64
	@$(MAKE) --no-print-directory build-linux-amd64
	@$(MAKE) --no-print-directory build-linux-arm64
	@$(MAKE) --no-print-directory build-linux-armv7

clean: ## Remove build outputs and generated embedded web assets while preserving .gitkeep.
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "if (Test-Path '$(BIN_DIR)') { Remove-Item -Recurse -Force '$(BIN_DIR)' }; if (Test-Path '$(WEB_EMBED_DIR)') { Get-ChildItem -Force '$(WEB_EMBED_DIR)' -ErrorAction SilentlyContinue | Where-Object { $$_.Name -ne '.gitkeep' } | Remove-Item -Recurse -Force }"
else
	@rm -rf "$(BIN_DIR)"
	@mkdir -p "$(WEB_EMBED_DIR)"
	@find "$(WEB_EMBED_DIR)" -mindepth 1 ! -name '.gitkeep' -exec rm -rf {} +
endif
