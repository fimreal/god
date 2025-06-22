package main

import (
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

type Process struct {
	Name  string    // Alias for the process
	Cmd   *exec.Cmd // Command to execute
	Alive bool      // Used to track whether the process is alive
}

type Manager struct {
	processes []*Process
	wg        sync.WaitGroup
}

func NewManager() *Manager {
	return &Manager{}
}

// AddProcess adds a process to be managed, supports alias and command
func (m *Manager) AddProcess(name string, command string) {
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
		Name: name,
		Cmd:  cmd,
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
	for _, proc := range m.processes {
		m.wg.Add(1)

		// Create prefixed writers for stdout and stderr
		proc.Cmd.Stdout = createPrefixedWriter(proc.Name, os.Stdout)
		proc.Cmd.Stderr = createPrefixedWriter(proc.Name, os.Stderr)

		log.Printf("[%s] Starting process", proc.Name)
		if err := proc.Cmd.Start(); err != nil {
			log.Printf("[%s] Failed to start process: %v", proc.Name, err)
			return fmt.Errorf("failed to start process %s: %w", proc.Name, err)
		}

		proc.Alive = true // Mark as started
		log.Printf("[%s] Process started successfully", proc.Name)

		go func(p *Process) {
			defer m.wg.Done()
			if err := p.Cmd.Wait(); err != nil {
				log.Printf("[%s] Process exited with error: %v", p.Name, err)
			} else {
				log.Printf("[%s] Process exited successfully", p.Name)
			}
			p.Alive = false // Update alive status
		}(proc)
	}

	return nil
}

func (m *Manager) Wait() {
	log.Println("Waiting for processes to finish...")
	m.wg.Wait()
	log.Println("All processes have finished.")
}

// HealthCheckHandler returns health check status including each service's status
func (m *Manager) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	response := "Health Check:\n"
	allHealthy := true

	for _, proc := range m.processes {
		if !proc.Alive {
			response += proc.Name + ": Unhealthy\n"
			allHealthy = false
		} else {
			response += proc.Name + ": Healthy\n"
		}
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
	var listenAddr string

	// Define command line flags
	flag.Var(&commands, "c", "Command to start the service in the format '[alias:]command', allowing multiple -c flags")
	flag.StringVar(&listenAddr, "l", "127.0.0.1:7788", "Address to listen for health checks (empty to disable)")
	flag.Parse()

	if len(commands) == 0 {
		log.Fatal("No commands provided.")
	}

	mgr := NewManager()

	for i, cmdStr := range commands {
		if cmdStr != "" {
			// Split the input into alias and command
			parts := strings.SplitN(cmdStr, ":", 2)
			var alias string
			var command string

			if len(parts) == 2 {
				// User provided an alias
				alias = strings.TrimSpace(parts[0])
				command = strings.TrimSpace(parts[1])
			} else {
				// No alias provided, auto-generate one
				alias = "app" + strconv.Itoa(i+1)     // Auto-generate alias like app1, app2
				command = strings.TrimSpace(parts[0]) // Use the whole string as command
			}

			log.Printf("Adding process: %s -> %s", alias, command)
			mgr.AddProcess(alias, command) // Add the process
		}
	}

	// Start the manager
	if err := mgr.Start(); err != nil {
		log.Fatalf("Failed to start manager: %v", err)
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
		mgr.Wait() // Wait for all processes to finish
		os.Exit(0) // Exit normally
	}()

	// Wait for termination signal (e.g., Ctrl+C)
	mgr.Wait()
}
