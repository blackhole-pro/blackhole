// Package main implements a stateful counter plugin demonstrating hot-swapping capabilities.
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

// Plugin state with versioning
type pluginStateV1 struct {
	Version  string            `json:"version"`
	Counters map[string]int64  `json:"counters"`
}

type pluginStateV2 struct {
	Version     string                 `json:"version"`
	Counters    map[string]int64       `json:"counters"`
	Labels      map[string]string      `json:"labels"`
	History     []historyEntry         `json:"history"`
	LastUpdated time.Time              `json:"last_updated"`
}

type historyEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Counter   string    `json:"counter"`
	Value     int64     `json:"value"`
	Operation string    `json:"operation"`
}

var (
	encoder      *json.Encoder
	decoder      *json.Decoder
	state        pluginStateV2
	running      bool
	pluginVersion = "2.0.0"
)

func main() {
	// Set up JSON RPC communication
	encoder = json.NewEncoder(os.Stdout)
	decoder = json.NewDecoder(os.Stdin)
	running = true

	// Initialize state
	state = pluginStateV2{
		Version:  pluginVersion,
		Counters: make(map[string]int64),
		Labels:   make(map[string]string),
		History:  make([]historyEntry, 0),
	}

	// Log to stderr
	log.SetOutput(os.Stderr)
	log.Printf("Stateful Counter plugin v%s starting...", pluginVersion)

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

	log.Println("Stateful Counter plugin exiting...")
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
	case "increment":
		counter := "default"
		if c, ok := req.Params["counter"].(string); ok {
			counter = c
		}
		
		state.Counters[counter]++
		state.LastUpdated = time.Now()
		
		// Add to history
		state.History = append(state.History, historyEntry{
			Timestamp: time.Now(),
			Counter:   counter,
			Value:     state.Counters[counter],
			Operation: "increment",
		})
		
		// Keep history limited
		if len(state.History) > 100 {
			state.History = state.History[len(state.History)-100:]
		}

		resp = pluginResponse{
			ID:      req.ID,
			Success: true,
			Result: map[string]interface{}{
				"counter": counter,
				"value":   state.Counters[counter],
			},
		}

	case "decrement":
		counter := "default"
		if c, ok := req.Params["counter"].(string); ok {
			counter = c
		}
		
		state.Counters[counter]--
		state.LastUpdated = time.Now()
		
		// Add to history
		state.History = append(state.History, historyEntry{
			Timestamp: time.Now(),
			Counter:   counter,
			Value:     state.Counters[counter],
			Operation: "decrement",
		})

		resp = pluginResponse{
			ID:      req.ID,
			Success: true,
			Result: map[string]interface{}{
				"counter": counter,
				"value":   state.Counters[counter],
			},
		}

	case "get":
		counter := "default"
		if c, ok := req.Params["counter"].(string); ok {
			counter = c
		}

		resp = pluginResponse{
			ID:      req.ID,
			Success: true,
			Result: map[string]interface{}{
				"counter": counter,
				"value":   state.Counters[counter],
				"label":   state.Labels[counter],
			},
		}

	case "set_label":
		counter := "default"
		if c, ok := req.Params["counter"].(string); ok {
			counter = c
		}
		
		label := ""
		if l, ok := req.Params["label"].(string); ok {
			label = l
		}
		
		state.Labels[counter] = label
		state.LastUpdated = time.Now()

		resp = pluginResponse{
			ID:      req.ID,
			Success: true,
			Result: map[string]interface{}{
				"counter": counter,
				"label":   label,
			},
		}

	case "get_all":
		resp = pluginResponse{
			ID:      req.ID,
			Success: true,
			Result: map[string]interface{}{
				"counters":     state.Counters,
				"labels":       state.Labels,
				"last_updated": state.LastUpdated,
			},
		}

	case "get_history":
		limit := 10
		if l, ok := req.Params["limit"].(float64); ok {
			limit = int(l)
		}
		
		history := state.History
		if len(history) > limit {
			history = history[len(history)-limit:]
		}

		resp = pluginResponse{
			ID:      req.ID,
			Success: true,
			Result: map[string]interface{}{
				"history": history,
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

	// Return state as string
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

	// Try to unmarshal as V2 first
	var newState pluginStateV2
	if err := json.Unmarshal([]byte(stateData), &newState); err == nil && newState.Version == "2.0.0" {
		state = newState
		log.Printf("State imported (V2): %d counters, %d labels, %d history entries", 
			len(state.Counters), len(state.Labels), len(state.History))
	} else {
		// Try V1 format for migration
		var v1State pluginStateV1
		if err := json.Unmarshal([]byte(stateData), &v1State); err == nil {
			// Migrate from V1 to V2
			state = pluginStateV2{
				Version:     pluginVersion,
				Counters:    v1State.Counters,
				Labels:      make(map[string]string),
				History:     make([]historyEntry, 0),
				LastUpdated: time.Now(),
			}
			
			// Add migration history entry
			for counter, value := range state.Counters {
				state.History = append(state.History, historyEntry{
					Timestamp: time.Now(),
					Counter:   counter,
					Value:     value,
					Operation: "migrated_from_v1",
				})
			}
			
			log.Printf("State migrated from V1 to V2: %d counters", len(state.Counters))
		} else {
			return rpcResponse{
				ID:    msg.ID,
				Error: fmt.Sprintf("failed to unmarshal state: %v", err),
			}
		}
	}

	return rpcResponse{
		ID:     msg.ID,
		Result: json.RawMessage(`{"status":"imported"}`),
	}
}