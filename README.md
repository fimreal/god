# God

[简体中文](./README_zh.md)

## Overview 

God is a lightweight process management tool written in Go. It allows you to start and manage multiple processes simultaneously, supporting health checks and optional aliases for each service. This tool is particularly useful in containerized environments where monitoring and managing services are crucial.

## Features

- **Lightweight and Simple**: Designed to be minimalistic with no unnecessary overhead, making it easy to use and deploy.
- **Start and Manage Multiple Processes Simultaneously**: Launch and control several processes at once without additional complexity.
- **Optional Aliases for Easy Identification**: Assign custom aliases to your processes for better organization; automatically generate aliases if none are provided.
- **HTTP Health Checks**: Monitor the status of running services through simple HTTP requests, ensuring that your applications are functioning correctly.
- **Flexible Configuration**: Easily adjust settings such as the health check listening address using command-line options.
- **Made for Docker**: Designed to work seamlessly with Docker containers, making it easy to manage services in a containerized environment.

## Installation

### Prerequisites

Ensure that you have Go installed on your machine. You can download it from the [official Go website](https://golang.org/dl/).

### Build the Application

1. Clone the repository:

   ```bash
   git clone https://github.com/fimreal/god.git
   cd god
   ```

2. Build the application:

   ```bash
   go build -o god .
   ```

## Usage

### Command Format

You can run the `god` executable with the following command-line options:

```bash
god -l ":7788" -c "alias:command --arg1 value" -c "alias:command2 subcommand --arg1 value"
```

- **alias**: (Optional) The service alias for easier identification.

### Health Check

God provides a health check endpoint at `/health`. You can use `curl` or any HTTP client to check the health status of the services:

```bash
curl http://localhost:7788/health
```

This will return the health status of all managed services:

```
Health Check:
app1: Healthy
app2: Unhealthy
```

## License

This project is licensed under the MIT License. For more details, please refer to the LICENSE file.
