package core

import (
	"bytes"
	"io"
	"sync"
	"strings"

	"go.uber.org/zap"
)

// setupProcessOutput configures stdout/stderr handling for the process
func setupProcessOutput(cmd ProcessCmd, serviceName string, logger *zap.Logger) {
	// Create service logger
	serviceLogger := logger.With(zap.String("service", serviceName))
	
	// Create prefixed writers for stdout and stderr
	stdout := newPrefixedLogWriter(serviceLogger, serviceName, false)
	stderr := newPrefixedLogWriter(serviceLogger, serviceName, true)
	
	// Attach to command
	cmd.SetOutput(stdout, stderr)
}

// prefixedLogWriter writes process output to a logger
type prefixedLogWriter struct {
	logger     *zap.Logger
	service    string
	isError    bool
	buffer     bytes.Buffer
	bufferLock sync.Mutex
}

// newPrefixedLogWriter creates a new prefixed log writer
func newPrefixedLogWriter(logger *zap.Logger, service string, isError bool) *prefixedLogWriter {
	return &prefixedLogWriter{
		logger:  logger,
		service: service,
		isError: isError,
	}
}

// Write implements io.Writer
func (w *prefixedLogWriter) Write(p []byte) (n int, err error) {
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
		
		// Log the line
		if w.isError {
			w.logger.Error(line, 
				zap.String("source", "stderr"))
		} else {
			w.logger.Info(line, 
				zap.String("source", "stdout"))
		}
	}
	
	return n, nil
}

// LogBuffer is a buffer that logs its contents
type LogBuffer struct {
	name       string
	isError    bool
	service    string
	logger     *zap.Logger
	lines      []string
	lineLimit  int
	mu         sync.Mutex
}

// NewLogBuffer creates a new log buffer
func NewLogBuffer(name string, isError bool, service string, logger *zap.Logger) *LogBuffer {
	return &LogBuffer{
		name:      name,
		isError:   isError,
		service:   service,
		logger:    logger,
		lines:     make([]string, 0, 100),
		lineLimit: 1000, // Keep last 1000 lines
	}
}

// Write implements io.Writer
func (b *LogBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Split input into lines
	input := string(p)
	lines := strings.Split(input, "\n")
	
	// Process each line
	for i, line := range lines {
		// Skip empty last line from the split
		if i == len(lines)-1 && line == "" {
			continue
		}
		
		// Add to buffer
		b.lines = append(b.lines, line)
		
		// Log the line
		if b.isError {
			b.logger.Error(line,
				zap.String("service", b.service),
				zap.String("stream", b.name))
		} else {
			b.logger.Info(line,
				zap.String("service", b.service),
				zap.String("stream", b.name))
		}
	}
	
	// Trim buffer if needed
	if len(b.lines) > b.lineLimit {
		b.lines = b.lines[len(b.lines)-b.lineLimit:]
	}
	
	return len(p), nil
}

// GetLines returns all buffered lines
func (b *LogBuffer) GetLines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Return a copy to prevent race conditions
	result := make([]string, len(b.lines))
	copy(result, b.lines)
	
	return result
}

// Clear clears the buffer
func (b *LogBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.lines = make([]string, 0, 100)
}