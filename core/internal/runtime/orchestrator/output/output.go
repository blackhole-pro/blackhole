// Package output provides process output handling functionality for the Process Orchestrator.
// It handles the capture and logging of process stdout and stderr streams.
package output

import (
	"bytes"
	"io"
	"strings"
	"sync"

	"github.com/blackhole-pro/blackhole/core/internal/runtime/orchestrator/types"
	"go.uber.org/zap"
)

// Setup configures process output handling for a command
func Setup(cmd types.ProcessCmd, serviceName string, logger *zap.Logger) {
	// Create service logger with context
	serviceLogger := logger.With(zap.String("service", serviceName))
	
	// Create prefixed writers for stdout and stderr
	stdout := NewPrefixedLogWriter(serviceLogger, serviceName, false)
	stderr := NewPrefixedLogWriter(serviceLogger, serviceName, true)
	
	// Attach writers to command
	cmd.SetOutput(stdout, stderr)
}

// PrefixedLogWriter writes process output to a logger with proper line handling
type PrefixedLogWriter struct {
	logger     *zap.Logger
	service    string
	isError    bool
	buffer     bytes.Buffer
	bufferLock sync.Mutex
}

// NewPrefixedLogWriter creates a new prefixed log writer
func NewPrefixedLogWriter(logger *zap.Logger, service string, isError bool) *PrefixedLogWriter {
	return &PrefixedLogWriter{
		logger:  logger,
		service: service,
		isError: isError,
	}
}

// Write implements io.Writer interface to capture and log process output
func (w *PrefixedLogWriter) Write(p []byte) (n int, err error) {
	w.bufferLock.Lock()
	defer w.bufferLock.Unlock()
	
	// Write to buffer
	n, err = w.buffer.Write(p)
	if err != nil {
		return n, err
	}
	
	// Process complete lines
	for {
		line, err := w.buffer.ReadString('\n')
		if err == io.EOF {
			// Put back incomplete line
			w.buffer.WriteString(line)
			break
		}
		
		// Trim trailing newline
		line = strings.TrimSuffix(line, "\n")
		if line == "" {
			continue
		}
		
		// Log the line with appropriate level
		if w.isError {
			w.logger.Error(line, zap.String("source", "stderr"))
		} else {
			w.logger.Info(line, zap.String("source", "stdout"))
		}
	}
	
	return n, nil
}