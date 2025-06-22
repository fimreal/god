# God

[简体中文](./README_zh.md)

[![god](https://github.com/fimreal/god/actions/workflows/release_build.yaml/badge.svg)](https://github.com/fimreal/god/actions/workflows/release_build.yaml)

## Overview 

God is a lightweight process management tool written in Go. It allows you to start and manage multiple processes simultaneously, supporting health checks, startup order, and optional aliases for each service. This tool is particularly useful in containerized environments where monitoring and managing services are crucial.

## Features

- **Lightweight and Simple**: Minimalistic, easy to use and deploy.
- **Start and Manage Multiple Processes**: Launch and control several processes at once.
- **Startup Order & Init Jobs**: Support for one-time initialization jobs (`-i`), service processes (`-c`) start only after all initialization tasks succeed.
- **No Auto-Restart**: No automatic process restart, recommend using Docker/K8s for restart control.
- **HTTP Health Checks**: `/health` endpoint reflects status of all processes and initialization tasks, suitable for container probes.
- **Flexible Configuration**: Health check port enabled by default (127.0.0.1:7788), can be disabled with `-l ""`.
- **Graceful Shutdown**: Supports graceful shutdown.
- **Made for Containers**: Naturally adapts to container environments.

## Installation

### Prerequisites

Ensure that you have Go installed on your machine. You can download it from the [official Go website](https://golang.org/dl/).

### Build the Application

```bash
git clone https://github.com/fimreal/god.git
cd god
go build -o god .
```

## Usage

### Start Services and Initialization Tasks

```bash
god -i "initdb:./init_db.sh" -c "nginx:/usr/sbin/nginx -g 'daemon off;'" -c "php:php-fpm"
```
- `-i`: One-time initialization tasks, service processes (`-c`) start only after all init tasks succeed.
- `-c`: Long-running service processes.

### Health Check

- Default listening on `127.0.0.1:7788`, can be disabled with `-l ""`.
- Health check endpoint: `/health`

```bash
curl http://localhost:7788/health
```

Response example:
```
Health Check:
initdb: Completed
nginx: Healthy
php: Healthy
```

## Changelog

- 2024-06-22
  - Support for one-time initialization tasks (`-i`) and startup order control
  - Enhanced health check interface, supports healthy status when no processes configured
  - Health check enabled by default, can be disabled with `-l ""`
  - No automatic process restart, recommend using Docker/K8s for restart control

## License

This project is licensed under the MIT License. For more details, please refer to the LICENSE file.
