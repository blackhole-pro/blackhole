// Package types defines the core type definitions and interfaces for the
// process component. It provides the contract that implementations must follow
// and ensures consistency across the process subsystem.
package types

import (
	"io"
	"os"
	"time"
)

// ProcessState represents the state of a service process
type ProcessState string

const (
	ProcessStateStopped    ProcessState = "stopped"
	ProcessStateStarting   ProcessState = "starting"
	ProcessStateRunning    ProcessState = "running"
	ProcessStateFailed     ProcessState = "failed"
	ProcessStateRestarting ProcessState = "restarting"
)

// ProcessManager defines the interface for process lifecycle operations
type ProcessManager interface {
	Start(name string) error
	Stop(name string) error
	Restart(name string) error
	Status(name string) (ProcessState, error)
	IsRunning(name string) bool
}

// ProcessExecutor abstracts the execution mechanism for better testability
type ProcessExecutor interface {
	Command(path string, args ...string) ProcessCmd
}

// ProcessCmd abstracts os/exec.Cmd for better testability
type ProcessCmd interface {
	Start() error
	Wait() error
	SetEnv(env []string)
	SetDir(dir string)
	SetOutput(stdout, stderr io.Writer)
	Signal(sig os.Signal) error
	Process() Process
}

// Process abstracts os.Process
type Process interface {
	Pid() int
	Kill() error
}

// ServiceInfo contains diagnostic information about a service
type ServiceInfo struct {
	Name         string        `json:"name"`
	Configured   bool          `json:"configured"`
	Enabled      bool          `json:"enabled"`
	State        string        `json:"state"`
	PID          int           `json:"pid,omitempty"`
	Uptime       time.Duration `json:"uptime,omitempty"`
	Restarts     int           `json:"restarts,omitempty"`
	LastExitCode int           `json:"last_exit_code,omitempty"`
	LastError    string        `json:"last_error,omitempty"`
}