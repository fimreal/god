// manager.go
// 进程管理核心逻辑，包括Manager结构体及其方法。
// Process management core logic: Manager struct and methods.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type TaskType int

const (
	TaskTypeInit    TaskType = iota // One-time initialization task
	TaskTypeService                 // Long-running service
)

type Manager struct {
	processes []*Process
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	initDone  chan struct{} // Signal when all init tasks are done
	debug     bool          // Enable debug logging
}

func NewManager(debug bool) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		ctx:      ctx,
		cancel:   cancel,
		initDone: make(chan struct{}),
		debug:    debug,
	}
}

func (m *Manager) AddProcess(name string, command string, taskType TaskType) {
	var cmd *exec.Cmd
	if _, err := exec.LookPath("sh"); err == nil {
		cmd = exec.Command("sh", "-c", command)
	} else {
		if m.debug {
			log.Printf("Using regular command execution for %s", name)
		}
		parts := strings.Fields(command)
		if len(parts) == 0 {
			log.Fatalf("No command provided for process %s", name)
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

func (m *Manager) Start() error {
	initTasks := []*Process{}
	serviceTasks := []*Process{}
	for _, proc := range m.processes {
		if proc.Type == TaskTypeInit {
			initTasks = append(initTasks, proc)
		} else {
			serviceTasks = append(serviceTasks, proc)
		}
	}
	if len(initTasks) > 0 {
		if m.debug {
			log.Println("Starting initialization tasks...")
		}
		for i, proc := range initTasks {
			if m.debug {
				log.Printf("Running init task %d/%d: %s", i+1, len(initTasks), proc.Name)
			}
			m.runInitTask(proc)
			proc.mu.Lock()
			if !proc.Success {
				if m.debug {
					log.Printf("Init task %s failed with exit code %d", proc.Name, proc.ExitCode)
				}
				close(m.initDone)
				return fmt.Errorf("initialization task %s failed", proc.Name)
			}
			proc.mu.Unlock()
		}
		if m.debug {
			log.Println("All initialization tasks completed successfully")
		}
	}
	close(m.initDone)
	if len(serviceTasks) > 0 {
		if m.debug {
			log.Println("Starting service tasks...")
		}
		for _, proc := range serviceTasks {
			m.wg.Add(1)
			go m.runServiceTask(proc)
		}
	}
	return nil
}

func (m *Manager) runInitTask(proc *Process) {
	stdout := createPrefixedWriter(proc.Name, os.Stdout)
	stderr := createPrefixedWriter(proc.Name, os.Stderr)
	proc.Cmd.Stdout = stdout
	proc.Cmd.Stderr = stderr
	if m.debug {
		log.Printf("Starting init task: %s", proc.Name)
	}
	if err := proc.Cmd.Start(); err != nil {
		if m.debug {
			log.Printf("Failed to start init task %s: %v", proc.Name, err)
		}
		proc.mu.Lock()
		proc.Success = false
		proc.ExitCode = -1
		proc.mu.Unlock()
		return
	}
	proc.Alive = true
	if m.debug {
		log.Printf("Init task %s started successfully", proc.Name)
	}
	if err := proc.Cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			proc.mu.Lock()
			proc.ExitCode = exitErr.ExitCode()
			proc.Success = (proc.ExitCode == 0)
			proc.mu.Unlock()
			if m.debug {
				log.Printf("Init task %s completed with exit code %d", proc.Name, proc.ExitCode)
			}
		} else {
			proc.mu.Lock()
			proc.Success = false
			proc.ExitCode = -1
			proc.mu.Unlock()
			if m.debug {
				log.Printf("Init task %s failed: %v", proc.Name, err)
			}
		}
	} else {
		proc.mu.Lock()
		proc.ExitCode = 0
		proc.Success = true
		proc.mu.Unlock()
		if m.debug {
			log.Printf("Init task %s completed successfully", proc.Name)
		}
	}
	proc.Alive = false
}

func (m *Manager) runServiceTask(proc *Process) {
	defer m.wg.Done()
	stdout := createPrefixedWriter(proc.Name, os.Stdout)
	stderr := createPrefixedWriter(proc.Name, os.Stderr)
	proc.Cmd.Stdout = stdout
	proc.Cmd.Stderr = stderr
	if m.debug {
		log.Printf("Starting service: %s", proc.Name)
	}
	if err := proc.Cmd.Start(); err != nil {
		if m.debug {
			log.Printf("Failed to start service %s: %v", proc.Name, err)
		}
		proc.Alive = false
		return
	}
	proc.Alive = true
	if m.debug {
		log.Printf("Service %s started successfully", proc.Name)
	}
	if err := proc.Cmd.Wait(); err != nil {
		if m.debug {
			log.Printf("Service %s exited with error: %v", proc.Name, err)
		}
	} else {
		if m.debug {
			log.Printf("Service %s exited successfully", proc.Name)
		}
	}
	proc.Alive = false
}

func (m *Manager) Wait() {
	if m.debug {
		log.Println("Waiting for processes to finish...")
	}
	m.wg.Wait()
	if m.debug {
		log.Println("All processes have finished.")
	}
}

func (m *Manager) Shutdown() {
	if m.debug {
		log.Println("Shutting down all processes...")
	}
	m.cancel()
	m.wg.Wait()
}

func (m *Manager) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	response := "Health Check:\n"
	allHealthy := true
	if len(m.processes) == 0 {
		response += "No processes configured\n"
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
		return
	}
	select {
	case <-m.initDone:
		for _, proc := range m.processes {
			if proc.Type == TaskTypeService {
				status := "Healthy"
				if !proc.Alive {
					status = "Unhealthy"
					allHealthy = false
				}
				response += fmt.Sprintf("%s: %s (ExitCode=%d)\n", proc.Name, status, proc.ExitCode)
			} else {
				proc.mu.Lock()
				status := "Completed"
				if !proc.Success {
					status = "Failed"
					allHealthy = false
				}
				response += fmt.Sprintf("%s: %s (ExitCode=%d)\n", proc.Name, status, proc.ExitCode)
				proc.mu.Unlock()
			}
		}
	default:
		response += "Initialization in progress...\n"
		allHealthy = false
	}
	if allHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	_, _ = w.Write([]byte(response))
}
