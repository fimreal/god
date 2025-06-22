# God

[English](./README.md)

[![god](https://github.com/fimreal/god/actions/workflows/release_build.yaml/badge.svg)](https://github.com/fimreal/god/actions/workflows/release_build.yaml)

## 概述

God 是一个轻量级的进程管理工具，使用 Go 语言编写。它允许您同时启动和管理多个进程，支持健康检查、启动顺序控制、一次性初始化任务（init job）和为每个服务指定可选别名，特别适合容器化环境。

## 特性

- **轻量且简单**：极简设计，易于使用和部署。
- **同时启动和管理多个进程**：一次性启动和控制多个进程。
- **启动顺序与初始化任务**：支持一次性初始化任务（`-i`），所有初始化任务成功后才启动服务（`-c`）。
- **不自动重启**：不自动重启进程，推荐用 Docker/K8s 控制重启。
- **HTTP 健康检查**：`/health` 端点反映所有进程和初始化任务状态，适合容器探针。
- **灵活配置**：健康检查端口默认开启（127.0.0.1:7788），可用 `-l ""` 关闭。
- **优雅关闭**：支持优雅关闭。
- **为容器而生**：天然适配容器环境。

## 安装

### 前提条件

确保您的计算机上已安装 Go。您可以从 [Go 官方网站](https://golang.org/dl/) 下载。

### 构建应用程序

```bash
git clone https://github.com/fimreal/god.git
cd god
go build -o god .
```

## 使用方法

### 启动服务与初始化任务

```bash
god -i "initdb:./init_db.sh" -c "nginx:/usr/sbin/nginx -g 'daemon off;'" -c "php:php-fpm"
```
- `-i`：一次性初始化任务，全部成功后才会启动 `-c` 服务。
- `-c`：常驻服务进程。

### 健康检查

- 默认监听 `127.0.0.1:7788`，可用 `-l ""` 关闭健康检查。
- 健康检查接口：`/health`

```bash
curl http://localhost:7788/health
```

返回示例：
```
Health Check:
initdb: Completed (ExitCode=0)
nginx: Healthy (ExitCode=0)
php: Healthy (ExitCode=0)
```

- 每行显示进程名、状态和最后一次退出码（ExitCode）。
- 对于失败或已退出的进程，ExitCode 会反映实际退出码。

#### 命令行参数说明
- `-i`  一次性初始化任务，如 `-i "initdb:./init_db.sh"`
- `-c`  服务进程，如 `-c "nginx:/usr/sbin/nginx -g 'daemon off;'"`
- `-l`  健康检查监听地址，默认 `127.0.0.1:7788`，设为空关闭
- `-d`  启用 debug 日志（可选）

> **注意：** 只有当冒号（`:`）出现在第一个空格前时，才会被识别为别名分隔符，否则整个字符串都作为命令。

## 更新日志

- 2024-06-22
  - 支持一次性初始化任务（`-i`）和启动顺序控制
  - 健康检查接口完善，支持无进程时返回健康
  - 默认开启健康检查，`-l ""` 可关闭
  - 不再自动重启进程，推荐用 Docker/K8s 控制

## 许可证

本项目采用 MIT 许可证。有关详细信息，请参阅 LICENSE 文件。