package main

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// recoveryInterceptor handles panics in gRPC handlers
func recoveryInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic with stack trace
				logger.Error("gRPC handler panic recovered",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())))
				
				// Return appropriate gRPC error
				err := status.Errorf(codes.Internal, "internal server error")
				panic(err) // Re-panic with gRPC error for proper error handling
			}
		}()
		
		return handler(ctx, req)
	}
}

// loggingInterceptor logs all gRPC requests and responses
func loggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		
		// Extract request metadata
		md, _ := metadata.FromIncomingContext(ctx)
		requestID := getRequestID(md)
		userAgent := getUserAgent(md)
		
		// Log request start
		logger.Info("gRPC request started",
			zap.String("method", info.FullMethod),
			zap.String("request_id", requestID),
			zap.String("user_agent", userAgent),
			zap.Time("start_time", start))
		
		// Call the handler
		resp, err := handler(ctx, req)
		
		// Calculate duration
		duration := time.Since(start)
		
		// Log request completion
		if err != nil {
			logger.Error("gRPC request completed with error",
				zap.String("method", info.FullMethod),
				zap.String("request_id", requestID),
				zap.Duration("duration", duration),
				zap.Error(err),
				zap.String("grpc_code", status.Code(err).String()))
		} else {
			logger.Info("gRPC request completed successfully",
				zap.String("method", info.FullMethod),
				zap.String("request_id", requestID),
				zap.Duration("duration", duration))
		}
		
		return resp, err
	}
}

// nodeLoggingInterceptor provides node-specific logging
func nodeLoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		
		// Log node-specific operations
		switch info.FullMethod {
		case "/blackhole.node.v1.NodeService/ConnectToPeer":
			logger.Info("Peer connection request",
				zap.String("method", info.FullMethod),
				zap.Time("timestamp", start))
			
		case "/blackhole.node.v1.NodeService/DisconnectFromPeer":
			logger.Info("Peer disconnection request",
				zap.String("method", info.FullMethod),
				zap.Time("timestamp", start))
			
		case "/blackhole.node.v1.NodeService/DiscoverPeers":
			logger.Info("Peer discovery request",
				zap.String("method", info.FullMethod),
				zap.Time("timestamp", start))
			
		case "/blackhole.node.v1.NodeService/GetNetworkStatus":
			logger.Debug("Network status request",
				zap.String("method", info.FullMethod))
		}
		
		// Call the handler
		resp, err := handler(ctx, req)
		
		// Log node-specific completion
		duration := time.Since(start)
		
		if err != nil {
			logger.Warn("Node operation failed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.Error(err))
		} else {
			// Log success with operation-specific details
			switch info.FullMethod {
			case "/blackhole.node.v1.NodeService/ConnectToPeer",
				 "/blackhole.node.v1.NodeService/DisconnectFromPeer":
				logger.Info("Peer operation completed",
					zap.String("method", info.FullMethod),
					zap.Duration("duration", duration))
				
			case "/blackhole.node.v1.NodeService/DiscoverPeers":
				logger.Info("Peer discovery completed",
					zap.String("method", info.FullMethod),
					zap.Duration("duration", duration))
			}
		}
		
		return resp, err
	}
}

// metricsInterceptor collects metrics for gRPC requests
func metricsInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		
		// Call the handler
		resp, err := handler(ctx, req)
		
		// Collect metrics
		duration := time.Since(start)
		success := err == nil
		
		// Log metrics (in production, you'd send to metrics system)
		logger.Debug("gRPC metrics",
			zap.String("method", info.FullMethod),
			zap.Duration("duration_ms", duration),
			zap.Bool("success", success),
			zap.String("status_code", status.Code(err).String()))
		
		// Update internal metrics counters here
		updateRequestMetrics(info.FullMethod, duration, success)
		
		return resp, err
	}
}

// streamingLoggingInterceptor handles streaming operations logging
func streamingLoggingInterceptor(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		
		logger.Info("gRPC streaming request started",
			zap.String("method", info.FullMethod),
			zap.Bool("client_stream", info.IsClientStream),
			zap.Bool("server_stream", info.IsServerStream),
			zap.Time("start_time", start))
		
		// Call the handler
		err := handler(srv, stream)
		
		// Log completion
		duration := time.Since(start)
		
		if err != nil {
			logger.Error("gRPC streaming request completed with error",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.Error(err))
		} else {
			logger.Info("gRPC streaming request completed successfully",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration))
		}
		
		return err
	}
}

// rateLimitingInterceptor provides basic rate limiting
func rateLimitingInterceptor(logger *zap.Logger, requestsPerSecond int) grpc.UnaryServerInterceptor {
	// Simple token bucket implementation
	tokenBucket := make(chan struct{}, requestsPerSecond)
	
	// Fill the bucket
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(requestsPerSecond))
		defer ticker.Stop()
		
		for range ticker.C {
			select {
			case tokenBucket <- struct{}{}:
			default:
				// Bucket is full, skip
			}
		}
	}()
	
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Try to get a token
		select {
		case <-tokenBucket:
			// Token acquired, proceed
			return handler(ctx, req)
		case <-time.After(100 * time.Millisecond):
			// No token available, rate limited
			logger.Warn("Request rate limited",
				zap.String("method", info.FullMethod))
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
		}
	}
}

// Helper functions

func getRequestID(md metadata.MD) string {
	if values := md.Get("request-id"); len(values) > 0 {
		return values[0]
	}
	// Generate a simple request ID if not provided
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

func getUserAgent(md metadata.MD) string {
	if values := md.Get("user-agent"); len(values) > 0 {
		return values[0]
	}
	return "unknown"
}

// updateRequestMetrics updates internal metrics (placeholder implementation)
func updateRequestMetrics(method string, duration time.Duration, success bool) {
	// In a real implementation, you would update metrics here
	// For example, increment counters, update histograms, etc.
	// This could integrate with Prometheus, StatsD, or other metrics systems
}

// AuthenticationInterceptor provides authentication for node operations
func authenticationInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract authentication token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "metadata not found")
		}
		
		// Check for authorization header
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			// Some methods may not require authentication
			if isPublicMethod(info.FullMethod) {
				return handler(ctx, req)
			}
			
			logger.Warn("Missing authorization header",
				zap.String("method", info.FullMethod))
			return nil, status.Errorf(codes.Unauthenticated, "authorization header required")
		}
		
		// Validate the token (placeholder implementation)
		token := authHeaders[0]
		if !isValidToken(token) {
			logger.Warn("Invalid authorization token",
				zap.String("method", info.FullMethod))
			return nil, status.Errorf(codes.Unauthenticated, "invalid authorization token")
		}
		
		// Add authenticated user info to context
		userID := extractUserID(token)
		newCtx := context.WithValue(ctx, "user_id", userID)
		
		return handler(newCtx, req)
	}
}

// Helper functions for authentication
func isPublicMethod(method string) bool {
	publicMethods := []string{
		"/blackhole.node.v1.NodeService/GetNodeInfo",
		"/blackhole.node.v1.NodeService/GetNetworkStatus",
	}
	
	for _, publicMethod := range publicMethods {
		if method == publicMethod {
			return true
		}
	}
	return false
}

func isValidToken(token string) bool {
	// Placeholder implementation - in production, validate against your auth system
	return token != "" && len(token) > 10
}

func extractUserID(token string) string {
	// Placeholder implementation - extract user ID from token
	return "user-from-token"
}