package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

const defaultListenAddr = ":7788" // Default listening address for health checks

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
	m.processes = append(m.processes, &Process{
		Name: name,
		Cmd:  exec.Command(command), // Directly execute the command without a shell
	})
}

// Start initiates all managed processes
func (m *Manager) Start() error {
	for _, proc := range m.processes {
		m.wg.Add(1)

		proc.Cmd.Stdout = os.Stdout
		proc.Cmd.Stderr = os.Stderr

		if err := proc.Cmd.Start(); err != nil {
			return err
		}

		proc.Alive = true // Mark as started

		go func(p *Process) {
			defer m.wg.Done()
			if err := p.Cmd.Wait(); err != nil {
				log.Printf("Process %s exited with error: %v", p.Name, err)
			} else {
				log.Printf("Process %s exited successfully", p.Name)
			}
			p.Alive = false // Update alive status
		}(proc)
	}

	return nil
}

// Wait blocks until all processes have finished
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
	flag.Var(&commands, "c", "Command to start the service in the format 'alias:command' or 'command'") // Allow multiple -c flags
	flag.StringVar(&listenAddr, "l", defaultListenAddr, "Address to listen for health checks")          // Default to ":7788"
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

			mgr.AddProcess(alias, command) // Add the process
		}
	}

	// Start the manager
	if err := mgr.Start(); err != nil {
		log.Fatalf("Failed to start manager: %v", err)
	}

	// Run the HTTP server for health checks
	go RunHTTPServer(listenAddr, mgr)

	// Wait for termination signal (e.g., Ctrl+C)
	mgr.Wait()
}
