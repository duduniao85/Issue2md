# issue2md Makefile
# 详见 spec.md §11。零第三方依赖，纯 Go 标准库构建（宪法 1.2）。

GO      ?= go
BIN_DIR := bin
BIN     := $(BIN_DIR)/issue2md

.PHONY: all build run test vet fmt web clean help

all: build

## build: 编译 CLI 到 bin/issue2md
build:
	$(GO) build -o $(BIN) ./cmd/issue2md

## run: 便捷运行（ARGS="..." 传参）
run:
	$(GO) run ./cmd/issue2md $(ARGS)

## test: 运行全部测试
test:
	$(GO) test ./...

## vet: 静态检查
vet:
	$(GO) vet ./...

## fmt: 格式化源码
fmt:
	$(GO) fmt ./...

## web: Web 服务（占位，本期未实现，见 spec.md §7.2）
web:
	@echo "issue2md: Web 服务尚在规划中（见 spec.md §7.2 / API-sketch.md §B）"

## clean: 清理构建产物
clean:
	rm -rf $(BIN_DIR)

## help: 列出可用目标
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'
