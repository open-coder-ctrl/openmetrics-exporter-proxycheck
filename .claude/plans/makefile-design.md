# Makefile 设计规格

## 项目信息
- **项目名称**: openmetric-proxycheck-exporter
- **二进制名称**: `openmetric-proxycheck-exporter`
- **Go 版本**: 1.26.1

## 需求确认

### 构建目标平台
- linux/amd64
- linux/arm64
- darwin/amd64
- darwin/arm64

### 代码质量工具
- golangci-lint

### 版本管理
- Git tag 自动注入（通过 -ldflags）

### Docker 支持
- 暂不需要

---

## Makefile 目标设计

### 1. 变量配置区

| 变量 | 值 | 说明 |
|------|-----|------|
| `BINARY_NAME` | `openmetric-proxycheck-exporter` | 输出二进制文件名 |
| `VERSION` | Git tag 或 "dev" | 版本号 |
| `BUILD_TIME` | UTC 时间戳 | 构建时间 |
| `GIT_COMMIT` | Git short SHA | 提交哈希 |
| `GO` | `go` | Go 命令 |
| `BUILD_DIR` | `.build` | 构建输出目录 |
| `LDFLAGS` | 版本注入参数 | 链接器标志 |

### 2. 核心构建目标

| 目标 | 命令 | 说明 |
|------|------|------|
| `build` | 默认构建 | 构建当前平台二进制 |
| `build-linux-amd64` | GOOS=linux GOARCH=amd64 | Linux x86_64 |
| `build-linux-arm64` | GOOS=linux GOARCH=arm64 | Linux ARM64 |
| `build-darwin-amd64` | GOOS=darwin GOARCH=amd64 | macOS Intel |
| `build-darwin-arm64` | GOOS=darwin GOARCH=arm64 | macOS Apple Silicon |
| `build-all` | 所有平台 | 交叉编译所有目标 |

**输出路径**: `.build/$(BINARY_NAME)-$(GOOS)-$(GOARCH)`

### 3. 测试目标

| 目标 | 说明 |
|------|------|
| `test` | 运行 `go test -v ./...` |
| `test-race` | 运行竞态检测 `go test -race ./...` |
| `test-cover` | 生成覆盖率报告至 `.build/coverage.html` |

### 4. 代码质量目标

| 目标 | 说明 |
|------|------|
| `fmt` | 格式化代码 `gofmt -s -w .` |
| `lint` | 运行 golangci-lint |
| `check` | 完整检查流程 (fmt + lint + test) |

### 5. 工具目标

| 目标 | 说明 |
|------|------|
| `clean` | 清理 `.build/` 目录 |
| `install` | 安装至 `$GOPATH/bin` |
| `run` | 本地运行 `./$(BINARY_NAME) -config config.yaml` |
| `help` | 显示所有目标帮助信息 |

### 6. 默认目标

`.DEFAULT_GOAL := help`

---

## 帮助系统格式

```
Usage:
  make <target>

Targets:
  build              Build binary for current platform
  build-all          Build binaries for all platforms
  test               Run unit tests
  test-race          Run tests with race detection
  test-cover         Run tests with coverage report
  lint               Run golangci-lint
  fmt                Format code with gofmt
  check              Run fmt, lint and test
  clean              Clean build artifacts
  install            Install binary to GOPATH/bin
  run                Run the binary locally
  help               Show this help message

Cross-compile targets:
  build-linux-amd64  Build for linux/amd64
  build-linux-arm64  Build for linux/arm64
  build-darwin-amd64 Build for darwin/amd64
  build-darwin-arm64 Build for darwin/arm64
```

---

## 实现计划

### 步骤 1: 创建 Makefile 文件

**文件**: `Makefile`

创建包含以下内容的 Makefile：
1. 变量配置区
   - BINARY_NAME=openmetric-proxycheck-exporter
   - VERSION（从 Git tag 获取）
   - BUILD_TIME（UTC 时间戳）
   - GIT_COMMIT（短哈希）
   - BUILD_DIR=.build
   - LDFLAGS（版本注入参数）

2. 构建目标
   - `build`: 当前平台构建
   - `build-all`: 所有平台交叉编译
   - `build-linux-amd64`, `build-linux-arm64`, `build-darwin-amd64`, `build-darwin-arm64`

3. 测试目标
   - `test`, `test-race`, `test-cover`

4. 代码质量目标
   - `fmt`, `lint`, `check`

5. 工具目标
   - `clean`, `install`, `run`, `help`

6. 默认目标: `help`

**验证标准**:
- `make help` 显示帮助信息
- `make build` 生成 `.build/openmetric-proxycheck-exporter-$(GOOS)-$(GOARCH)`
- `make test` 通过所有测试
- `make lint` 无错误

### 步骤 2: 更新 .gitignore

添加 `.build/` 目录到 `.gitignore`，忽略构建产物。

### 步骤 3: 验证 Makefile

运行以下命令验证：
```bash
make help
make build
make test
make lint
```

---

## 实现文件

将生成/修改以下文件：
1. `Makefile` (新建)
2. `.gitignore` (修改)
