# God

[English](./README.md)

[![god](https://github.com/fimreal/god/actions/workflows/release_build.yaml/badge.svg)](https://github.com/fimreal/god/actions/workflows/release_build.yaml)

## 概述

God 是一个轻量级的进程管理工具，使用 Go 语言编写。它允许您同时启动和管理多个进程，并支持健康检查以及为每个服务指定可选别名，该工具在容器化环境中尤其有用。

## 特性

- **轻量且简单**：设计为极简，无不必要的开销，使其易于使用和部署。
- **同时启动和管理多个进程**：一次性启动和控制多个进程，而无需额外复杂性。
- **可选别名以便于识别**：为您的进程分配自定义别名，以便更好地组织；如果没有提供别名，则自动生成别名。
- **HTTP 健康检查**：通过简单的 HTTP 请求监控正在运行的服务状态，确保您的应用程序正常运行。
- **灵活配置**：使用命令行选项轻松调整设置，例如健康检查监听地址。
- **为 Docker 而生**：旨在与 Docker 容器无缝协作，使在容器化环境中管理服务变得简单。

## 安装

### 前提条件

确保您的计算机上已安装 Go。您可以从 [Go 官方网站](https://golang.org/dl/) 下载。

### 构建应用程序

1. 克隆仓库：

   ```bash
   git clone https://github.com/fimreal/god.git
   cd god
   ```

2. 构建应用程序：

   ```bash
   go build -o god .
   ```

## 使用方法

### 命令格式

您可以使用以下命令行选项运行 `god` 可执行文件：

```bash
god -l ":7788" -c "alias:command --arg1 value" -c "alias:command2 subcommand --arg1 value"
```

- **alias**: （可选）服务别名，以便于识别。

### 健康检查

God 提供了一个健康检查端点 `/health`。您可以使用 `curl` 或任何 HTTP 客户端来检查服务的健康状况：

```bash
curl http://localhost:7788/health
```

这将返回所有管理服务的健康状态：

```
Health Check:
app1: Healthy
app2: Unhealthy
```

## 许可证

本项目采用 MIT 许可证。有关详细信息，请参阅 LICENSE 文件。