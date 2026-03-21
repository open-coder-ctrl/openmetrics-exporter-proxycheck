# Proxy Check Exporter

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[English](README_EN.md) | 简体中文

一个用于监控 HTTP、HTTPS 和 SOCKS5 代理服务器健康状态和性能的 Prometheus Exporter。

## 功能特性

- 并发监控多个代理服务器
- 支持 HTTP、HTTPS 和 SOCKS5 代理类型
- 可配置的检查间隔和超时时间
- 支持响应体正则表达式验证
- 支持基本认证（Basic Authentication）
- 提供 Prometheus 兼容的指标端点
- 提供 `/health` 健康检查端点

## 快速开始

### 环境要求

- Go 1.21 或更高版本

### 安装

使用 Makefile 编译（推荐）：

```bash
make build
# 输出: .build/openmetric-proxycheck-exporter-<os>-<arch>
```

或直接编译：

```bash
go build -o proxycheck-exporter .
```

### 最小配置

创建 `config.yaml` 文件：

```yaml
server:
  listen: ":9099"
  interval: 30s

defaults:
  target: "https://api.ipify.org"
  timeout: 10s

proxies:
  - name: "my-proxy"
    type: http
    address: "192.168.1.100:8080"
```

### 启动

```bash
./proxycheck-exporter -config config.yaml
```

### 验证安装

```bash
# 健康检查
curl http://localhost:9099/health
# 输出: OK

# 查看指标
curl http://localhost:9099/metrics | grep proxy_
```

## 配置说明

### 完整配置示例

```yaml
server:
  listen: ":9099"        # 监听地址
  interval: 30s          # 检查间隔

defaults:
  target: "https://api.ipify.org"  # 默认检测目标
  timeout: 10s                      # 默认超时时间

proxies:
  - name: "http-proxy-1"
    type: http
    address: "192.168.1.100:8080"

  - name: "https-proxy-auth"
    type: https
    address: "proxy.example.com:443"
    username: "user"
    password: "pass"
    target: "https://api.ipify.org?format=json"
    expect_regex: ".*ip.*"

  - name: "socks5-proxy-1"
    type: socks5
    address: "127.0.0.1:1080"
    timeout: 5s
```

### 服务器配置

| 字段 | 描述 | 默认值 |
|------|------|--------|
| `listen` | HTTP 服务监听地址 | 无（必须指定） |
| `interval` | 代理健康检查间隔 | 无（必须指定） |

### 默认配置

| 字段 | 描述 | 默认值 |
|------|------|--------|
| `target` | 默认检测目标 URL | 无（必须指定） |
| `timeout` | 默认请求超时时间 | 无（必须指定） |

### 代理配置

| 字段 | 描述 | 是否必填 | 示例 |
|------|------|----------|------|
| `name` | 代理名称（用于指标标签） | 是 | `"my-proxy"` |
| `type` | 代理类型：`http`、`https` 或 `socks5` | 是 | `http` |
| `address` | 代理地址，格式为 `host:port` | 是 | `"192.168.1.100:8080"` |
| `target` | 覆盖默认检测目标 URL | 否 | `"https://example.com"` |
| `timeout` | 覆盖默认超时时间 | 否 | `5s` |
| `username` | 认证用户名 | 否 | `"user"` |
| `password` | 认证密码 | 否 | `"pass"` |
| `expect_regex` | 响应体匹配的正则表达式 | 否 | `".*ok.*"` |

## 指标

### 指标列表

| 指标名称 | 类型 | 标签 | 描述 |
|----------|------|------|------|
| `proxy_up` | Gauge | `name`, `type`, `address` | 代理可用为 1，不可用为 0 |
| `proxy_response_seconds` | Gauge | `name`, `type` | 响应时间（秒） |
| `proxy_status_code` | Gauge | `name`, `type` | 目标返回的 HTTP 状态码 |
| `proxy_check_errors_total` | Counter | `name`, `type`, `error_type` | 按类型统计的检查错误总数 |

### 指标输出示例

```
# HELP proxy_up Whether the proxy is up (1) or down (0)
# TYPE proxy_up gauge
proxy_up{name="my-proxy",type="http",address="192.168.1.100:8080"} 1

# HELP proxy_response_seconds Response time of the proxy check in seconds
# TYPE proxy_response_seconds gauge
proxy_response_seconds{name="my-proxy",type="http"} 0.234

# HELP proxy_status_code HTTP status code returned by the target through the proxy
# TYPE proxy_status_code gauge
proxy_status_code{name="my-proxy",type="http"} 200

# HELP proxy_check_errors_total Total number of proxy check errors by type
# TYPE proxy_check_errors_total counter
proxy_check_errors_total{name="my-proxy",type="http",error_type="timeout"} 2
```

### 错误类型

| 类型 | 描述 |
|------|------|
| `timeout` | 请求超时 |
| `connection` | 连接失败 |
| `status` | HTTP 状态码非 2xx |
| `regex` | 响应体不匹配预期正则 |
| `auth` | 认证失败 |

## Prometheus 集成

### 抓取配置

在 `prometheus.yml` 中添加：

```yaml
scrape_configs:
  - job_name: 'proxycheck'
    static_configs:
      - targets: ['localhost:9099']
```

### PromQL 查询示例

```promql
# 所有代理的可用性状态
proxy_up

# 平均响应时间（按代理类型分组）
avg by (type) (proxy_response_seconds)

# 最近5分钟的错误率
rate(proxy_check_errors_total[5m])

# 响应时间超过1秒的代理
proxy_response_seconds > 1

# 统计可用代理数量
count(proxy_up == 1)
```

## Grafana 集成

### 推荐面板

| 面板名称 | PromQL 查询 | 可视化类型 |
|----------|-------------|------------|
| 代理可用性 | `proxy_up` | Stat / Gauge |
| 响应时间趋势 | `proxy_response_seconds` | Time series |
| 错误率 | `rate(proxy_check_errors_total[5m])` | Graph |
| 代理状态总览 | `count by (type) (proxy_up == 1)` | Pie chart |

### 导入仪表板

1. 在 Grafana 中创建新仪表板
2. 添加 Prometheus 数据源（指向你的 Prometheus 实例）
3. 使用上述 PromQL 查询创建面板

## 告警规则

在 Prometheus 的 `rules.yml` 中添加：

```yaml
groups:
  - name: proxy_alerts
    rules:
      - alert: ProxyDown
        expr: proxy_up == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "代理 {{ $labels.name }} 不可用"
          description: "代理 {{ $labels.name }} ({{ $labels.address }}) 已不可用超过2分钟"

      - alert: ProxyHighErrorRate
        expr: rate(proxy_check_errors_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "代理 {{ $labels.name }} 错误率过高"
          description: "代理 {{ $labels.name }} 最近5分钟错误率为 {{ $value | humanize }}/s"

      - alert: ProxySlowResponse
        expr: proxy_response_seconds > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "代理 {{ $labels.name }} 响应缓慢"
          description: "代理 {{ $labels.name }} 响应时间超过5秒"
```

## Makefile 使用

### 常用命令

| 命令 | 描述 |
|------|------|
| `make build` | 编译当前平台二进制文件 |
| `make build-linux-amd64` | 编译 Linux AMD64 版本 |
| `make build-linux-arm64` | 编译 Linux ARM64 版本 |
| `make build-darwin-amd64` | 编译 macOS AMD64 版本 |
| `make build-darwin-arm64` | 编译 macOS ARM64 版本 |
| `make build-all` | 编译所有平台版本 |
| `make test` | 运行单元测试 |
| `make test-race` | 运行竞态检测测试 |
| `make test-cover` | 生成测试覆盖率报告 |
| `make fmt` | 格式化代码 |
| `make lint` | 运行代码检查 |
| `make check` | 运行 fmt + lint + test |
| `make clean` | 清理构建产物 |
| `make run` | 编译并运行 |
| `make help` | 显示帮助信息 |

### 交叉编译示例

```bash
# 编译 Linux AMD64 版本
make build-linux-amd64

# 编译所有平台
make build-all
```

## 故障排除

### 常见问题

| 问题 | 可能原因 | 解决方案 |
|------|----------|----------|
| `proxy_up == 0` | 代理地址错误或网络不通 | 检查地址格式 `host:port`，确认网络可达 |
| 认证失败 (`auth` 错误) | 用户名或密码错误 | 验证 `username` 和 `password` 配置 |
| 请求超时 (`timeout` 错误) | 网络延迟高或目标不可达 | 增加 `timeout` 值，检查 `target` URL |
| 状态码错误 (`status` 错误) | 目标返回非 2xx 状态 | 检查 `target` URL 是否正确 |
| 正则不匹配 (`regex` 错误) | 响应体不匹配 `expect_regex` | 使用 `curl` 测试实际响应，调整正则 |

### 调试命令

```bash
# 检查服务是否运行
curl http://localhost:9099/health

# 查看原始指标
curl http://localhost:9099/metrics

# 测试代理连接（手动）
curl -x http://192.168.1.100:8080 https://api.ipify.org
```

### 日志分析

启动时会输出日志：

```
2026/03/21 10:00:00 Configuration loaded: 3 proxies configured
2026/03/21 10:00:00 Starting proxycheck-exporter on :9099
2026/03/21 10:00:00 [http-proxy-1] OK - 234ms
2026/03/21 10:00:00 [https-proxy-auth] FAILED - auth: authentication failed
```

## 端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/metrics` | GET | Prometheus 指标端点 |
| `/health` | GET | 健康检查（服务器运行时返回 "OK"） |

## 安全注意事项

- **凭据存储**：生产环境应避免在配置文件中存储明文凭据。建议使用环境变量或密钥管理服务。
- **文件权限**：限制配置文件访问权限：`chmod 600 config.yaml`
- **网络安全**：`/metrics` 端点会暴露代理地址。如有需要，请使用网络策略或认证机制限制访问。
- **健康检查说明**：`/health` 端点仅表示 HTTP 服务运行正常，不代表代理可用性。

## 许可证

MIT License
