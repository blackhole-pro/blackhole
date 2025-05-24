// Package plugins provides the plugin execution framework
package plugins

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// RPCRequest represents a JSON-RPC request from the host
type RPCRequest struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// RPCResponse represents a JSON-RPC response to the host
type RPCResponse struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *RPCError       `json:"error,omitempty"`
}

// RPCError represents an error in JSON-RPC format
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Run starts the plugin RPC server using stdin/stdout
func Run(plugin Plugin) error {
	log.SetPrefix(fmt.Sprintf("[%s] ", plugin.Info().Name))
	log.Printf("Plugin starting...")
	
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	
	for scanner.Scan() {
		line := scanner.Bytes()
		
		var request RPCRequest
		if err := json.Unmarshal(line, &request); err != nil {
			log.Printf("Failed to unmarshal request: %v", err)
			continue
		}
		
		response := handleRequest(plugin, request)
		
		if err := encoder.Encode(response); err != nil {
			log.Printf("Failed to encode response: %v", err)
			return err
		}
	}
	
	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("scanner error: %w", err)
	}
	
	return nil
}

func handleRequest(plugin Plugin, request RPCRequest) RPCResponse {
	switch request.Method {
	case "initialize":
		return handleInitialize(plugin, request)
	case "start":
		return handleStart(plugin, request)
	case "stop":
		return handleStop(plugin, request)
	case "handle":
		return handlePluginRequest(plugin, request)
	case "healthcheck":
		return handleHealthCheck(plugin, request)
	case "getinfo":
		return handleGetInfo(plugin, request)
	case "getstatus":
		return handleGetStatus(plugin, request)
	case "shutdown":
		return handleShutdown(plugin, request)
	case "export_state":
		return handleExportState(plugin, request)
	case "import_state":
		return handleImportState(plugin, request)
	default:
		return RPCResponse{
			ID: request.ID,
			Error: &RPCError{
				Code:    -32601,
				Message: "Method not found",
			},
		}
	}
}

func handleInitialize(plugin Plugin, request RPCRequest) RPCResponse {
	// Plugin is already initialized when created
	result, _ := json.Marshal(map[string]interface{}{
		"success": true,
		"message": "Plugin initialized",
	})
	
	return RPCResponse{
		ID:     request.ID,
		Result: result,
	}
}

func handleStart(plugin Plugin, request RPCRequest) RPCResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := plugin.Start(ctx); err != nil {
		return RPCResponse{
			ID: request.ID,
			Error: &RPCError{
				Code:    -32000,
				Message: err.Error(),
			},
		}
	}
	
	result, _ := json.Marshal(map[string]interface{}{
		"success": true,
		"message": "Plugin started",
	})
	
	return RPCResponse{
		ID:     request.ID,
		Result: result,
	}
}

func handleStop(plugin Plugin, request RPCRequest) RPCResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := plugin.Stop(ctx); err != nil {
		return RPCResponse{
			ID: request.ID,
			Error: &RPCError{
				Code:    -32000,
				Message: err.Error(),
			},
		}
	}
	
	result, _ := json.Marshal(map[string]interface{}{
		"success": true,
		"message": "Plugin stopped",
	})
	
	return RPCResponse{
		ID:     request.ID,
		Result: result,
	}
}

func handlePluginRequest(plugin Plugin, request RPCRequest) RPCResponse {
	var pluginReq PluginRequest
	if err := json.Unmarshal(request.Params, &pluginReq); err != nil {
		return RPCResponse{
			ID: request.ID,
			Error: &RPCError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	response, err := plugin.Handle(ctx, pluginReq)
	if err != nil {
		return RPCResponse{
			ID: request.ID,
			Error: &RPCError{
				Code:    -32000,
				Message: err.Error(),
			},
		}
	}
	
	result, _ := json.Marshal(response)
	
	return RPCResponse{
		ID:     request.ID,
		Result: result,
	}
}

func handleHealthCheck(plugin Plugin, request RPCRequest) RPCResponse {
	if err := plugin.HealthCheck(); err != nil {
		return RPCResponse{
			ID: request.ID,
			Error: &RPCError{
				Code:    -32000,
				Message: err.Error(),
			},
		}
	}
	
	result, _ := json.Marshal(map[string]interface{}{
		"healthy": true,
		"message": "Plugin is healthy",
	})
	
	return RPCResponse{
		ID:     request.ID,
		Result: result,
	}
}

func handleGetInfo(plugin Plugin, request RPCRequest) RPCResponse {
	info := plugin.Info()
	result, _ := json.Marshal(info)
	
	return RPCResponse{
		ID:     request.ID,
		Result: result,
	}
}

func handleGetStatus(plugin Plugin, request RPCRequest) RPCResponse {
	status := plugin.GetStatus()
	result, _ := json.Marshal(map[string]interface{}{
		"status": string(status),
	})
	
	return RPCResponse{
		ID:     request.ID,
		Result: result,
	}
}

func handleShutdown(plugin Plugin, request RPCRequest) RPCResponse {
	if err := plugin.PrepareShutdown(); err != nil {
		log.Printf("Error preparing shutdown: %v", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := plugin.Stop(ctx); err != nil {
		log.Printf("Error stopping plugin: %v", err)
	}
	
	result, _ := json.Marshal(map[string]interface{}{
		"success": true,
		"message": "Plugin shutting down",
	})
	
	response := RPCResponse{
		ID:     request.ID,
		Result: result,
	}
	
	// Send response before exiting
	encoder := json.NewEncoder(os.Stdout)
	encoder.Encode(response)
	
	// Exit after a small delay to ensure response is sent
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}()
	
	return response
}

func handleExportState(plugin Plugin, request RPCRequest) RPCResponse {
	state, err := plugin.ExportState()
	if err != nil {
		return RPCResponse{
			ID: request.ID,
			Error: &RPCError{
				Code:    -32000,
				Message: err.Error(),
			},
		}
	}
	
	result, _ := json.Marshal(map[string]interface{}{
		"state": state,
	})
	
	return RPCResponse{
		ID:     request.ID,
		Result: result,
	}
}

func handleImportState(plugin Plugin, request RPCRequest) RPCResponse {
	var params struct {
		State []byte `json:"state"`
	}
	
	if err := json.Unmarshal(request.Params, &params); err != nil {
		return RPCResponse{
			ID: request.ID,
			Error: &RPCError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
	}
	
	if err := plugin.ImportState(params.State); err != nil {
		return RPCResponse{
			ID: request.ID,
			Error: &RPCError{
				Code:    -32000,
				Message: err.Error(),
			},
		}
	}
	
	result, _ := json.Marshal(map[string]interface{}{
		"success": true,
		"message": "State imported",
	})
	
	return RPCResponse{
		ID:     request.ID,
		Result: result,
	}
}