package process

import (
	"fmt"
	"io"
	"os"
	"os/exec"
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

// ProcessError provides contextual information about process errors
type ProcessError struct {
	Service  string
	Err      error
	ExitCode int
}

// Error implements the error interface
func (e *ProcessError) Error() string {
	return fmt.Sprintf("service %s: %v (exit code: %d)", e.Service, e.Err, e.ExitCode)
}

// Unwrap returns the underlying error
func (e *ProcessError) Unwrap() error {
	return e.Err
}

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

// DefaultProcessExecutor uses os/exec to execute processes
type DefaultProcessExecutor struct{}

// Command creates a new command using os/exec
func (e *DefaultProcessExecutor) Command(path string, args ...string) ProcessCmd {
	return &DefaultProcessCmd{cmd: exec.Command(path, args...)}
}

// DefaultProcessCmd wraps os/exec.Cmd
type DefaultProcessCmd struct {
	cmd *exec.Cmd
}

// Start starts the command
func (c *DefaultProcessCmd) Start() error {
	return c.cmd.Start()
}

// Wait waits for the command to complete
func (c *DefaultProcessCmd) Wait() error {
	return c.cmd.Wait()
}

// SetEnv sets the environment variables for the command
func (c *DefaultProcessCmd) SetEnv(env []string) {
	c.cmd.Env = env
}

// SetDir sets the working directory for the command
func (c *DefaultProcessCmd) SetDir(dir string) {
	c.cmd.Dir = dir
}

// SetOutput sets the stdout and stderr writers for the command
func (c *DefaultProcessCmd) SetOutput(stdout, stderr io.Writer) {
	c.cmd.Stdout = stdout
	c.cmd.Stderr = stderr
}

// Signal sends a signal to the running process
func (c *DefaultProcessCmd) Signal(sig os.Signal) error {
	if c.cmd.Process == nil {
		return fmt.Errorf("process not started")
	}
	return c.cmd.Process.Signal(sig)
}

// Process returns the underlying Process interface
func (c *DefaultProcessCmd) Process() Process {
	if c.cmd.Process == nil {
		return nil
	}
	return &DefaultProcess{process: c.cmd.Process}
}

// DefaultProcess wraps os.Process
type DefaultProcess struct {
	process *os.Process
}

// Pid returns the process ID
func (p *DefaultProcess) Pid() int {
	return p.process.Pid
}

// Kill terminates the process
func (p *DefaultProcess) Kill() error {
	return p.process.Kill()
}

// ServiceInfo contains diagnostic information about a service
type ServiceInfo struct {
	Name         string        `json:"name"`
	Configured   bool          `json:"configured"`
	Enabled      bool          `json:"enabled"`
	State        string        `json:"state"`
	PID          int           `json:"pid,omitempty"`
	Uptime       int64         `json:"uptime_seconds,omitempty"`
	Restarts     int           `json:"restarts,omitempty"`
	LastExitCode int           `json:"last_exit_code,omitempty"`
	LastError    string        `json:"last_error,omitempty"`
}