package main

import (
	"os/exec"
	"sync"
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
