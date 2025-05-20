// Package isolation provides process isolation functionality for the Process Orchestrator.
// It handles resource limits, environment setup, and process isolation.
package isolation

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/handcraftdev/blackhole/internal/core/config/types"
	processtypes "github.com/handcraftdev/blackhole/internal/core/process/types"
)

// Setup configures process isolation settings for a command
func Setup(cmd processtypes.ProcessCmd, serviceCfg *types.ServiceConfig) {
	// Set working directory if specified
	if serviceCfg.DataDir != "" {
		cmd.SetDir(serviceCfg.DataDir)
	}
	
	// Create a clean environment
	cleanEnv := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"TEMP=" + os.TempDir(),
		"TMP=" + os.TempDir(),
	}
	
	// Add service-specific environment variables
	if len(serviceCfg.Environment) > 0 {
		for key, value := range serviceCfg.Environment {
			cleanEnv = append(cleanEnv, fmt.Sprintf("%s=%s", key, value))
		}
	}
	
	// Add Go memory limit (for Go services)
	if serviceCfg.MemoryLimit > 0 {
		cleanEnv = append(cleanEnv, fmt.Sprintf("GOMEMLIMIT=%dMiB", serviceCfg.MemoryLimit))
	}
	
	// Set the environment variables
	cmd.SetEnv(cleanEnv)
}

// FindServiceBinary locates the binary for a service
func FindServiceBinary(servicesDir, name string, customPath string) (string, error) {
	// Use custom path if provided
	if customPath != "" {
		if fileExists(customPath) {
			return customPath, nil
		}
		return "", fmt.Errorf("service binary not found at specified path: %s", customPath)
	}
	
	// Default to binary in service directory: servicesDir/name/name
	binaryPath := filepath.Join(servicesDir, name, name)
	
	// Ensure binary exists
	if !fileExists(binaryPath) {
		return "", fmt.Errorf("service binary not found at default path: %s", binaryPath)
	}
	
	return binaryPath, nil
}

// fileExists checks if a file exists and is not a directory
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}