package core

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// LifecycleManager manages the application lifecycle
type LifecycleManager struct {
	hooks     map[LifecyclePhase][]LifecycleHook
	mu        sync.RWMutex
	logger    *logrus.Logger
	shutdown  chan os.Signal
	isRunning bool
}

// LifecyclePhase represents different phases of the application lifecycle
type LifecyclePhase string

const (
	PhasePreStart  LifecyclePhase = "pre-start"
	PhaseStart     LifecyclePhase = "start"
	PhasePostStart LifecyclePhase = "post-start"
	PhasePreStop   LifecyclePhase = "pre-stop"
	PhaseStop      LifecyclePhase = "stop"
	PhasePostStop  LifecyclePhase = "post-stop"
)

// LifecycleHook is a function that runs during a lifecycle phase
type LifecycleHook func() error

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{
		hooks:    make(map[LifecyclePhase][]LifecycleHook),
		logger:   logrus.New(),
		shutdown: make(chan os.Signal, 1),
	}
}

// RegisterHook registers a hook for a specific lifecycle phase
func (lm *LifecycleManager) RegisterHook(phase LifecyclePhase, hook LifecycleHook) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	lm.hooks[phase] = append(lm.hooks[phase], hook)
	lm.logger.WithField("phase", phase).Debug("Registered lifecycle hook")
}

// Start starts the lifecycle manager
func (lm *LifecycleManager) Start() error {
	lm.mu.Lock()
	if lm.isRunning {
		lm.mu.Unlock()
		return fmt.Errorf("lifecycle manager is already running")
	}
	lm.isRunning = true
	lm.mu.Unlock()
	
	// Setup signal handling
	signal.Notify(lm.shutdown, syscall.SIGINT, syscall.SIGTERM)
	
	// Execute pre-start hooks
	if err := lm.executeHooks(PhasePreStart); err != nil {
		return fmt.Errorf("pre-start hooks failed: %w", err)
	}
	
	// Execute start hooks
	if err := lm.executeHooks(PhaseStart); err != nil {
		return fmt.Errorf("start hooks failed: %w", err)
	}
	
	// Execute post-start hooks in background
	go func() {
		time.Sleep(1 * time.Second) // Small delay before post-start
		if err := lm.executeHooks(PhasePostStart); err != nil {
			lm.logger.WithError(err).Error("Post-start hooks failed")
		}
	}()
	
	// Start shutdown listener
	go lm.shutdownListener()
	
	return nil
}

// Stop stops the lifecycle manager
func (lm *LifecycleManager) Stop() error {
	lm.mu.Lock()
	if !lm.isRunning {
		lm.mu.Unlock()
		return fmt.Errorf("lifecycle manager is not running")
	}
	lm.isRunning = false
	lm.mu.Unlock()
	
	// Execute pre-stop hooks
	if err := lm.executeHooks(PhasePreStop); err != nil {
		lm.logger.WithError(err).Error("Pre-stop hooks failed")
	}
	
	// Execute stop hooks
	if err := lm.executeHooks(PhaseStop); err != nil {
		lm.logger.WithError(err).Error("Stop hooks failed")
	}
	
	// Execute post-stop hooks
	if err := lm.executeHooks(PhasePostStop); err != nil {
		lm.logger.WithError(err).Error("Post-stop hooks failed")
	}
	
	return nil
}

// executeHooks executes all hooks for a given phase
func (lm *LifecycleManager) executeHooks(phase LifecyclePhase) error {
	lm.mu.RLock()
	hooks := lm.hooks[phase]
	lm.mu.RUnlock()
	
	lm.logger.WithField("phase", phase).WithField("count", len(hooks)).Info("Executing lifecycle hooks")
	
	for i, hook := range hooks {
		if err := hook(); err != nil {
			return fmt.Errorf("hook %d in phase %s failed: %w", i, phase, err)
		}
	}
	
	return nil
}

// shutdownListener listens for shutdown signals
func (lm *LifecycleManager) shutdownListener() {
	<-lm.shutdown
	lm.logger.Info("Received shutdown signal")
	
	// Trigger graceful shutdown
	go func() {
		if err := lm.Stop(); err != nil {
			lm.logger.WithError(err).Error("Error during shutdown")
		}
		os.Exit(0)
	}()
	
	// Force exit after timeout
	time.Sleep(30 * time.Second)
	lm.logger.Error("Shutdown timeout exceeded, forcing exit")
	os.Exit(1)
}

// IsRunning returns whether the lifecycle manager is running
func (lm *LifecycleManager) IsRunning() bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.isRunning
}