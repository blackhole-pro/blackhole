// Package main implements a simple hello world plugin for the Blackhole Framework.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// RPC message types matching the framework protocol
type rpcMessage struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// Plugin request/response types
type pluginRequest struct {
	ID      string                 `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
	Data    []byte                 `json:"data,omitempty"`
}

type pluginResponse struct {
	ID       string                 `json:"id"`
	Success  bool                   `json:"success"`
	Result   map[string]interface{} `json:"result,omitempty"`
	Data     []byte                 `json:"data,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata responseMetadata       `json:"metadata"`
}

type responseMetadata struct {
	ProcessingTime time.Duration `json:"processing_time"`
}

// Plugin state
type pluginState struct {
	GreetingCount int    `json:"greeting_count"`
	LastGreeting  string `json:"last_greeting"`
}

var (
	encoder *json.Encoder
	decoder *json.Decoder
	state   pluginState
	running bool
)

func main() {
	// Set up JSON RPC communication
	encoder = json.NewEncoder(os.Stdout)
	decoder = json.NewDecoder(os.Stdin)
	running = true

	// Log to stderr
	log.SetOutput(os.Stderr)
	log.Println("Hello plugin starting...")

	// Main RPC loop
	scanner := bufio.NewScanner(os.Stdin)
	for running && scanner.Scan() {
		var msg rpcMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			log.Printf("Failed to decode message: %v", err)
			continue
		}

		response := handleMessage(msg)
		if err := encoder.Encode(response); err != nil {
			log.Printf("Failed to encode response: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
	}

	log.Println("Hello plugin exiting...")
}

func handleMessage(msg rpcMessage) rpcResponse {
	switch msg.Method {
	case "initialize":
		return handleInitialize(msg)
	case "handle":
		return handleRequest(msg)
	case "healthcheck":
		return handleHealthCheck(msg)
	case "prepare_shutdown":
		return handlePrepareShutdown(msg)
	case "shutdown":
		return handleShutdown(msg)
	case "export_state":
		return handleExportState(msg)
	case "import_state":
		return handleImportState(msg)
	default:
		return rpcResponse{
			ID:    msg.ID,
			Error: fmt.Sprintf("unknown method: %s", msg.Method),
		}
	}
}

func handleInitialize(msg rpcMessage) rpcResponse {
	log.Println("Plugin initialized")
	return rpcResponse{
		ID:     msg.ID,
		Result: json.RawMessage(`{"status":"initialized"}`),
	}
}

func handleRequest(msg rpcMessage) rpcResponse {
	var req pluginRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		return rpcResponse{
			ID:    msg.ID,
			Error: fmt.Sprintf("failed to unmarshal request: %v", err),
		}
	}

	startTime := time.Now()

	// Handle different methods
	var resp pluginResponse
	switch req.Method {
	case "greet":
		name := "World"
		if n, ok := req.Params["name"].(string); ok {
			name = n
		}

		greeting := fmt.Sprintf("Hello, %s!", name)
		state.GreetingCount++
		state.LastGreeting = greeting

		resp = pluginResponse{
			ID:      req.ID,
			Success: true,
			Result: map[string]interface{}{
				"greeting": greeting,
				"count":    state.GreetingCount,
			},
		}

	case "get_stats":
		resp = pluginResponse{
			ID:      req.ID,
			Success: true,
			Result: map[string]interface{}{
				"greeting_count": state.GreetingCount,
				"last_greeting":  state.LastGreeting,
			},
		}

	default:
		resp = pluginResponse{
			ID:      req.ID,
			Success: false,
			Error:   fmt.Sprintf("unknown method: %s", req.Method),
		}
	}

	resp.Metadata.ProcessingTime = time.Since(startTime)

	// Marshal response
	result, err := json.Marshal(resp)
	if err != nil {
		return rpcResponse{
			ID:    msg.ID,
			Error: fmt.Sprintf("failed to marshal response: %v", err),
		}
	}

	return rpcResponse{
		ID:     msg.ID,
		Result: result,
	}
}

func handleHealthCheck(msg rpcMessage) rpcResponse {
	return rpcResponse{
		ID:     msg.ID,
		Result: json.RawMessage(`{"status":"healthy"}`),
	}
}

func handlePrepareShutdown(msg rpcMessage) rpcResponse {
	log.Println("Preparing for shutdown...")
	return rpcResponse{
		ID:     msg.ID,
		Result: json.RawMessage(`{"status":"prepared"}`),
	}
}

func handleShutdown(msg rpcMessage) rpcResponse {
	log.Println("Shutting down...")
	running = false
	return rpcResponse{
		ID:     msg.ID,
		Result: json.RawMessage(`{"status":"shutdown"}`),
	}
}

func handleExportState(msg rpcMessage) rpcResponse {
	data, err := json.Marshal(state)
	if err != nil {
		return rpcResponse{
			ID:    msg.ID,
			Error: fmt.Sprintf("failed to marshal state: %v", err),
		}
	}

	// Return state as string (base64 encoding could be added)
	result, _ := json.Marshal(string(data))
	return rpcResponse{
		ID:     msg.ID,
		Result: result,
	}
}

func handleImportState(msg rpcMessage) rpcResponse {
	var stateData string
	if err := json.Unmarshal(msg.Params, &stateData); err != nil {
		return rpcResponse{
			ID:    msg.ID,
			Error: fmt.Sprintf("failed to unmarshal state data: %v", err),
		}
	}

	if err := json.Unmarshal([]byte(stateData), &state); err != nil {
		return rpcResponse{
			ID:    msg.ID,
			Error: fmt.Sprintf("failed to unmarshal state: %v", err),
		}
	}

	log.Printf("State imported: count=%d, last=%s", state.GreetingCount, state.LastGreeting)
	return rpcResponse{
		ID:     msg.ID,
		Result: json.RawMessage(`{"status":"imported"}`),
	}
}