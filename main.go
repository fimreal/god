package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

const defaultListenAddr = "127.0.0.1:7788" // 默认开启健康检查，设为空字符串可关闭

type TaskType int

const (
	TaskTypeInit    TaskType = iota // One-time initialization task
	TaskTypeService                 // Long-running service
)

type Process struct {
	Name     string     // Alias for the process
	Cmd      *exec.Cmd  // Command to execute
	Alive    bool       // Used to track whether the process is alive
	Command  string     // Original command string
	Type     TaskType   // Task type (init or service)
	ExitCode int        // Exit code for init tasks
	Success  bool       // Whether init task completed successfully
	mu       sync.Mutex // Protect status operations
}

type Manager struct {
	processes []*Process
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	initDone  chan struct{} // Signal when all init tasks are done
}

func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		ctx:      ctx,
		cancel:   cancel,
		initDone: make(chan struct{}),
	}
}

// AddProcess adds a process to be managed, supports alias and command
func (m *Manager) AddProcess(name string, command string, taskType TaskType) {
	var cmd *exec.Cmd

	// Check if `sh` is available
	if _, err := exec.LookPath("sh"); err == nil {
		// Try using sh -c to execute the command
		cmd = exec.Command("sh", "-c", command)
	} else {
		log.Printf("[%s] Using regular command execution", name)

		// Split the command into executable and arguments
		parts := strings.Fields(command)
		if len(parts) == 0 {
			log.Fatalf("[%s] No command provided for process", name)
		}
		cmd = exec.Command(parts[0], parts[1:]...)
	}

	m.processes = append(m.processes, &Process{
		Name:    name,
		Cmd:     cmd,
		Command: command,
		Type:    taskType,
	})
}

// createPrefixedWriter creates a writer that prefixes each line with the process name
func createPrefixedWriter(name string, writer io.Writer) io.Writer {
	return &prefixedWriter{
		name:   name,
		writer: writer,
	}
}

type prefixedWriter struct {
	name   string
	writer io.Writer
}

func (pw *prefixedWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// Split by newlines and prefix each line
	lines := strings.Split(string(p), "\n")

	for i, line := range lines {
		if i > 0 && i < len(lines)-1 {
			// Add newline for all lines except the first and last
			line = "\n" + line
		}

		if line != "" {
			prefixedLine := fmt.Sprintf("[%s] %s", pw.name, line)
			if _, err := pw.writer.Write([]byte(prefixedLine)); err != nil {
				return n, err
			}
		}
		n += len(line)
	}

	return n, nil
}

func (m *Manager) Start() error {
	// First, start all init tasks
	initWg := sync.WaitGroup{}
	initTasks := []*Process{}
	serviceTasks := []*Process{}

	// Separate init tasks and service tasks
	for _, proc := range m.processes {
		if proc.Type == TaskTypeInit {
			initTasks = append(initTasks, proc)
		} else {
			serviceTasks = append(serviceTasks, proc)
		}
	}

	// Start init tasks first
	if len(initTasks) > 0 {
		log.Println("Starting initialization tasks...")
		for _, proc := range initTasks {
			initWg.Add(1)
			go m.runInitTask(proc, &initWg)
		}
		initWg.Wait()

		// Check if all init tasks succeeded
		allInitSuccess := true
		for _, proc := range initTasks {
			proc.mu.Lock()
			if !proc.Success {
				allInitSuccess = false
				log.Printf("[%s] Init task failed with exit code %d", proc.Name, proc.ExitCode)
			}
			proc.mu.Unlock()
		}

		if !allInitSuccess {
			log.Println("Some initialization tasks failed, not starting services")
			close(m.initDone)
			return fmt.Errorf("initialization tasks failed")
		}
		log.Println("All initialization tasks completed successfully")
	}

	// Signal that init is done
	close(m.initDone)

	// Start service tasks
	if len(serviceTasks) > 0 {
		log.Println("Starting service tasks...")
		for _, proc := range serviceTasks {
			m.wg.Add(1)
			go m.runServiceTask(proc)
		}
	}

	return nil
}

// runInitTask runs a one-time initialization task
func (m *Manager) runInitTask(proc *Process, wg *sync.WaitGroup) {
	defer wg.Done()

	stdout := createPrefixedWriter(proc.Name, os.Stdout)
	stderr := createPrefixedWriter(proc.Name, os.Stderr)
	proc.Cmd.Stdout = stdout
	proc.Cmd.Stderr = stderr

	log.Printf("[%s] Starting init task", proc.Name)
	if err := proc.Cmd.Start(); err != nil {
		log.Printf("[%s] Failed to start init task: %v", proc.Name, err)
		proc.mu.Lock()
		proc.Success = false
		proc.ExitCode = -1
		proc.mu.Unlock()
		return
	}

	proc.Alive = true
	log.Printf("[%s] Init task started successfully", proc.Name)

	// Wait for init task to complete
	if err := proc.Cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			proc.mu.Lock()
			proc.ExitCode = exitErr.ExitCode()
			proc.Success = (proc.ExitCode == 0)
			proc.mu.Unlock()
			log.Printf("[%s] Init task completed with exit code %d", proc.Name, proc.ExitCode)
		} else {
			proc.mu.Lock()
			proc.Success = false
			proc.ExitCode = -1
			proc.mu.Unlock()
			log.Printf("[%s] Init task failed: %v", proc.Name, err)
		}
	} else {
		proc.mu.Lock()
		proc.ExitCode = 0
		proc.Success = true
		proc.mu.Unlock()
		log.Printf("[%s] Init task completed successfully", proc.Name)
	}

	proc.Alive = false
}

// runServiceTask runs a long-running service
func (m *Manager) runServiceTask(proc *Process) {
	defer m.wg.Done()

	stdout := createPrefixedWriter(proc.Name, os.Stdout)
	stderr := createPrefixedWriter(proc.Name, os.Stderr)
	proc.Cmd.Stdout = stdout
	proc.Cmd.Stderr = stderr

	log.Printf("[%s] Starting service", proc.Name)
	if err := proc.Cmd.Start(); err != nil {
		log.Printf("[%s] Failed to start service: %v", proc.Name, err)
		proc.Alive = false
		return
	}

	proc.Alive = true
	log.Printf("[%s] Service started successfully", proc.Name)

	// Wait for service to exit
	if err := proc.Cmd.Wait(); err != nil {
		log.Printf("[%s] Service exited with error: %v", proc.Name, err)
	} else {
		log.Printf("[%s] Service exited successfully", proc.Name)
	}

	proc.Alive = false
}

func (m *Manager) Wait() {
	log.Println("Waiting for processes to finish...")
	m.wg.Wait()
	log.Println("All processes have finished.")
}

// Shutdown gracefully shuts down all processes
func (m *Manager) Shutdown() {
	log.Println("Shutting down all processes...")
	m.cancel()
	m.wg.Wait()
}

// HealthCheckHandler returns health check status including each service's status
func (m *Manager) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	response := "Health Check:\n"
	allHealthy := true

	// If no processes, return healthy
	if len(m.processes) == 0 {
		response += "No processes configured\n"
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
		return
	}

	// Check if init tasks are still running
	select {
	case <-m.initDone:
		// Init tasks completed, check service status
		for _, proc := range m.processes {
			if proc.Type == TaskTypeService {
				if !proc.Alive {
					response += proc.Name + ": Unhealthy\n"
					allHealthy = false
				} else {
					response += proc.Name + ": Healthy\n"
				}
			} else {
				// For init tasks, check if they succeeded
				proc.mu.Lock()
				if proc.Success {
					response += proc.Name + ": Completed\n"
				} else {
					response += proc.Name + ": Failed\n"
					allHealthy = false
				}
				proc.mu.Unlock()
			}
		}
	default:
		// Init tasks still running
		response += "Initialization in progress...\n"
		allHealthy = false
	}

	if allHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}

	// Write the response
	_, _ = w.Write([]byte(response))
}

// RunHTTPServer starts the HTTP server for health checks
func RunHTTPServer(addr string, mgr *Manager) {
	http.HandleFunc("/health", mgr.HealthCheckHandler)
	log.Printf("Starting HTTP server for health check on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

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

func main() {
	var commands StringSlice
	var initCommands StringSlice
	var listenAddr string

	// Define command line flags
	flag.Var(&commands, "c", "Command to start the service in the format '[alias:]command', allowing multiple -c flags")
	flag.Var(&initCommands, "i", "One-time initialization command in the format '[alias:]command', allowing multiple -i flags")
	flag.StringVar(&listenAddr, "l", defaultListenAddr, "Address to listen for health checks (empty to disable)")
	flag.Parse()

	if len(commands) == 0 && len(initCommands) == 0 {
		log.Fatal("No commands provided.")
	}

	mgr := NewManager()

	// Add init tasks first
	for i, cmdStr := range initCommands {
		if cmdStr != "" {
			parts := strings.SplitN(cmdStr, ":", 2)
			var alias string
			var command string

			if len(parts) == 2 {
				alias = strings.TrimSpace(parts[0])
				command = strings.TrimSpace(parts[1])
			} else {
				alias = "init" + strconv.Itoa(i+1)
				command = strings.TrimSpace(parts[0])
			}

			log.Printf("Adding init task: %s -> %s", alias, command)
			mgr.AddProcess(alias, command, TaskTypeInit)
		}
	}

	// Add service tasks
	for i, cmdStr := range commands {
		if cmdStr != "" {
			parts := strings.SplitN(cmdStr, ":", 2)
			var alias string
			var command string

			if len(parts) == 2 {
				alias = strings.TrimSpace(parts[0])
				command = strings.TrimSpace(parts[1])
			} else {
				alias = "app" + strconv.Itoa(i+1)
				command = strings.TrimSpace(parts[0])
			}

			log.Printf("Adding service: %s -> %s", alias, command)
			mgr.AddProcess(alias, command, TaskTypeService)
		}
	}

	// Start the manager
	if err := mgr.Start(); err != nil {
		log.Printf("Manager start failed: %v", err)
		// Don't exit immediately, let health check show the status
	}

	// Run the HTTP server for health checks only if listenAddr is provided
	if listenAddr != "" {
		go RunHTTPServer(listenAddr, mgr)
		log.Printf("Health check server enabled on %s", listenAddr)
	} else {
		log.Println("Health check server disabled")
	}

	// Set up signal handling for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		log.Println("Received interrupt signal, shutting down gracefully...")
		mgr.Shutdown() // Use new shutdown method
		os.Exit(0)     // Exit normally
	}()

	// Wait for termination signal (e.g., Ctrl+C)
	mgr.Wait()
}
