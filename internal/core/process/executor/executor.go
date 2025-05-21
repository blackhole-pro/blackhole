// Package executor provides implementations for process execution
// interfaces defined in the types package. It encapsulates OS-level
// process management functionality.
package executor

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/handcraftdev/blackhole/internal/core/process/types"
)

// DefaultProcessExecutor uses os/exec to execute processes
type DefaultProcessExecutor struct{}

// Command creates a new command using os/exec
func (e *DefaultProcessExecutor) Command(path string, args ...string) types.ProcessCmd {
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
func (c *DefaultProcessCmd) Process() types.Process {
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

// NewDefaultExecutor creates a new DefaultProcessExecutor
func NewDefaultExecutor() types.ProcessExecutor {
	return &DefaultProcessExecutor{}
}