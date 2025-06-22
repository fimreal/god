package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

// main.go
// 程序入口，只负责参数解析、信号处理和主流程调度。
// Main entry: only handles argument parsing, signal handling, and main flow control.
// 业务逻辑见 manager.go, process.go, writer.go

const defaultListenAddr = "127.0.0.1:7788" // 默认开启健康检查，设为空字符串可关闭

// StringSlice is a custom type to hold a slice of strings for flag parsing
type StringSlice []string

// String returns the string representation of the slice
func (ss *StringSlice) String() string {
	return strings.Join(*ss, ", ")
}

// Set appends a new value to the slice
func (ss *StringSlice) Set(value string) error {
	*ss = append(*ss, value)
	return nil
}

// RunHTTPServer starts the HTTP server for health checks
func RunHTTPServer(addr string, mgr *Manager) {
	http.HandleFunc("/health", mgr.HealthCheckHandler)
	if mgr.debug {
		log.Printf("Starting HTTP server for health check on %s", addr)
	}
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func main() {
	var commands StringSlice
	var initCommands StringSlice
	var listenAddr string
	var debug bool

	flag.Var(&commands, "c", "Command to start the service in the format '[alias:]command', allowing multiple -c flags")
	flag.Var(&initCommands, "i", "One-time initialization command in the format '[alias:]command', allowing multiple -i flags")
	flag.StringVar(&listenAddr, "l", defaultListenAddr, "Address to listen for health checks (empty to disable)")
	flag.BoolVar(&debug, "d", false, "Enable debug logging")
	flag.Parse()

	if len(commands) == 0 && len(initCommands) == 0 {
		if debug {
			log.Println("No commands provided.")
		}
		os.Exit(1)
	}

	mgr := NewManager(debug)

	// Add init tasks first
	for i, cmdStr := range initCommands {
		if cmdStr != "" {
			idx := strings.Index(cmdStr, ":")
			var alias string
			var command string
			if idx > 0 && !strings.Contains(cmdStr[:idx], " ") {
				alias = strings.TrimSpace(cmdStr[:idx])
				command = strings.TrimSpace(cmdStr[idx+1:])
			} else {
				alias = "init" + strconv.Itoa(i+1)
				command = strings.TrimSpace(cmdStr)
			}
			if debug {
				log.Printf("Adding init task: %s -> %s", alias, command)
			}
			mgr.AddProcess(alias, command, TaskTypeInit)
		}
	}

	// Add service tasks
	for i, cmdStr := range commands {
		if cmdStr != "" {
			idx := strings.Index(cmdStr, ":")
			var alias string
			var command string
			if idx > 0 && !strings.Contains(cmdStr[:idx], " ") {
				alias = strings.TrimSpace(cmdStr[:idx])
				command = strings.TrimSpace(cmdStr[idx+1:])
			} else {
				alias = "app" + strconv.Itoa(i+1)
				command = strings.TrimSpace(cmdStr)
			}
			if debug {
				log.Printf("Adding service: %s -> %s", alias, command)
			}
			mgr.AddProcess(alias, command, TaskTypeService)
		}
	}

	// Start the manager
	if err := mgr.Start(); err != nil {
		if debug {
			log.Printf("Manager start failed: %v", err)
		}
		// Don't exit immediately, let health check show the status
	}

	// Run the HTTP server for health checks only if listenAddr is provided
	if listenAddr != "" {
		go RunHTTPServer(listenAddr, mgr)
		if debug {
			log.Printf("Health check server enabled on %s", listenAddr)
		}
	} else {
		if debug {
			log.Println("Health check server disabled")
		}
	}

	// Set up signal handling for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		if debug {
			log.Println("Received interrupt signal, shutting down gracefully...")
		}
		mgr.Shutdown()
		os.Exit(0)
	}()

	mgr.Wait()
}
