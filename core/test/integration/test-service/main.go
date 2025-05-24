package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Log startup details
	log.Printf("Test service started with PID: %d", os.Getpid())
	log.Printf("Arguments: %v", os.Args)
	
	// Create a data file to indicate service has started
	dataDir := os.Getenv("BLACKHOLE_SERVICE_DATA_DIR")
	if dataDir != "" {
		err := os.MkdirAll(dataDir, 0755)
		if err != nil {
			log.Printf("Failed to create data directory: %v", err)
		} else {
			statusFile := fmt.Sprintf("%s/status.txt", dataDir)
			content := fmt.Sprintf("Service started at: %s\nPID: %d\n", time.Now().Format(time.RFC3339), os.Getpid())
			err = os.WriteFile(statusFile, []byte(content), 0644)
			if err != nil {
				log.Printf("Failed to write status file: %v", err)
			}
		}
	} else {
		log.Printf("BLACKHOLE_SERVICE_DATA_DIR not set")
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run until signal received
	log.Printf("Service running and waiting for signals...")
	sig := <-sigChan
	log.Printf("Received signal: %v, shutting down gracefully", sig)
	
	// Write shutdown status
	if dataDir != "" {
		shutdownFile := fmt.Sprintf("%s/shutdown.txt", dataDir)
		content := fmt.Sprintf("Service shutdown at: %s\nSignal: %s\n", time.Now().Format(time.RFC3339), sig)
		err := os.WriteFile(shutdownFile, []byte(content), 0644)
		if err != nil {
			log.Printf("Failed to write shutdown file: %v", err)
		}
	}
}