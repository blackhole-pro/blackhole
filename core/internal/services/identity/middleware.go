package main

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// loggingInterceptor creates a gRPC unary server interceptor for logging
func loggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		
		// Extract client information
		var clientAddr string
		if p, ok := peer.FromContext(ctx); ok {
			clientAddr = p.Addr.String()
		}

		// Log request start
		logger.Info("gRPC request started",
			zap.String("method", info.FullMethod),
			zap.String("client_addr", clientAddr),
			zap.Time("start_time", start),
		)

		// Call the handler
		resp, err := handler(ctx, req)
		
		// Calculate duration
		duration := time.Since(start)
		
		// Extract status code
		statusCode := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			}
		}

		// Log request completion
		if err != nil {
			logger.Error("gRPC request completed with error",
				zap.String("method", info.FullMethod),
				zap.String("client_addr", clientAddr),
				zap.Duration("duration", duration),
				zap.String("status_code", statusCode.String()),
				zap.Error(err),
			)
		} else {
			logger.Info("gRPC request completed successfully",
				zap.String("method", info.FullMethod),
				zap.String("client_addr", clientAddr),  
				zap.Duration("duration", duration),
				zap.String("status_code", statusCode.String()),
			)
		}

		return resp, err
	}
}

// recoveryInterceptor creates a gRPC unary server interceptor for panic recovery
func recoveryInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("gRPC handler panic recovered",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.Stack("stack_trace"),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// metricsInterceptor creates a gRPC unary server interceptor for basic metrics
func metricsInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Call the handler
		resp, err := handler(ctx, req)

		// Log metrics
		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			}
		}

		logger.Debug("gRPC request metrics",
			zap.String("method", info.FullMethod),
			zap.Duration("duration_ms", duration),
			zap.String("status", statusCode.String()),
			zap.Bool("success", err == nil),
		)

		return resp, err
	}
}

// authLoggingInterceptor creates a specialized interceptor for authentication requests
func authLoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract auth-specific information
		var authFields []zap.Field
		
		switch info.FullMethod {
		case "/blackhole.identity.auth.v1.AuthService/GenerateChallenge":
			// Log challenge generation without sensitive data
			authFields = append(authFields, 
				zap.String("auth_action", "challenge_generation"),
			)
			
		case "/blackhole.identity.auth.v1.AuthService/VerifyResponse":
			// Log verification attempt without sensitive data
			authFields = append(authFields,
				zap.String("auth_action", "signature_verification"),
			)
		}

		logger.Info("Authentication request",
			append(authFields, 
				zap.String("method", info.FullMethod),
			)...,
		)

		// Call the handler
		resp, err := handler(ctx, req)

		// Log authentication result
		if err != nil {
			logger.Warn("Authentication failed",
				append(authFields,
					zap.String("method", info.FullMethod),
					zap.Error(err),
				)...,
			)
		} else {
			logger.Info("Authentication successful",
				append(authFields,
					zap.String("method", info.FullMethod),
				)...,
			)
		}

		return resp, err
	}
}