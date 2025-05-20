package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// Application represents the main blackhole application
type Application struct {
	config    *Config
	lifecycle *LifecycleManager
	services  map[string]Service
	logger    *logrus.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// Service interface that all services must implement
type Service interface {
	Start(ctx context.Context) error
	Stop() error
	Name() string
	Health() bool
}

// NewApplication creates a new application instance
func NewApplication() *Application {
	ctx, cancel := context.WithCancel(context.Background())
	
	app := &Application{
		config:    NewConfig(),
		lifecycle: NewLifecycleManager(),
		services:  make(map[string]Service),
		logger:    logrus.New(),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Configure logger
	app.logger.SetLevel(logrus.InfoLevel)
	app.logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	return app
}

// RegisterService registers a service with the application
func (a *Application) RegisterService(service Service) error {
	if _, exists := a.services[service.Name()]; exists {
		return fmt.Errorf("service %s already registered", service.Name())
	}
	a.services[service.Name()] = service
	a.logger.WithField("service", service.Name()).Info("Service registered")
	return nil
}

// Start starts the application and all registered services
func (a *Application) Start() error {
	a.logger.Info("Starting blackhole application")

	// Load configuration
	if err := a.config.Load(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Start lifecycle manager
	if err := a.lifecycle.Start(); err != nil {
		return fmt.Errorf("failed to start lifecycle manager: %w", err)
	}

	// Start all services
	for name, service := range a.services {
		a.wg.Add(1)
		go func(name string, svc Service) {
			defer a.wg.Done()
			
			a.logger.WithField("service", name).Info("Starting service")
			if err := svc.Start(a.ctx); err != nil {
				a.logger.WithField("service", name).WithError(err).Error("Service failed to start")
			}
		}(name, service)
	}

	// Wait for shutdown signal
	<-a.ctx.Done()
	
	// Stop all services
	a.Stop()
	
	return nil
}

// Stop stops the application and all services
func (a *Application) Stop() {
	a.logger.Info("Stopping blackhole application")
	
	// Cancel context to signal shutdown
	a.cancel()
	
	// Stop all services
	for name, service := range a.services {
		a.logger.WithField("service", name).Info("Stopping service")
		if err := service.Stop(); err != nil {
			a.logger.WithField("service", name).WithError(err).Error("Error stopping service")
		}
	}
	
	// Wait for all goroutines to finish
	a.wg.Wait()
	
	// Stop lifecycle manager
	if err := a.lifecycle.Stop(); err != nil {
		a.logger.WithError(err).Error("Error stopping lifecycle manager")
	}
	
	a.logger.Info("Blackhole application stopped")
}

// GetService returns a service by name
func (a *Application) GetService(name string) (Service, bool) {
	service, exists := a.services[name]
	return service, exists
}

// Config returns the application configuration
func (a *Application) Config() *Config {
	return a.config
}

// Logger returns the application logger
func (a *Application) Logger() *logrus.Logger {
	return a.logger
}