package loader

import (
	"fmt"
	
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins"
	"github.com/blackhole-pro/blackhole/core/internal/framework/plugins/validator"
)

// complianceValidator validates plugin compliance with development guidelines
type complianceValidator struct {
	validator *validator.ComplianceValidator
}

// newComplianceValidator creates a new compliance validator
func newComplianceValidator(strictMode bool) *complianceValidator {
	return &complianceValidator{
		validator: validator.NewComplianceValidator(strictMode),
	}
}

// Validate checks if the plugin complies with development guidelines
func (v *complianceValidator) Validate(spec plugins.PluginSpec, binaryPath string) error {
	// For .plugin packages, validate the package
	if spec.Source.Type == plugins.SourceTypeRemote || spec.Source.Type == plugins.SourceTypeMarketplace {
		result, err := v.validator.ValidatePluginPackage(binaryPath)
		if err != nil {
			return fmt.Errorf("compliance validation error: %w", err)
		}
		
		if !result.Valid {
			return fmt.Errorf("plugin compliance violations: %v", result.Errors)
		}
		
		// Log warnings (in real implementation, use proper logger)
		if len(result.Warnings) > 0 {
			// fmt.Printf("Plugin compliance warnings: %v\n", result.Warnings)
		}
	}
	
	// For local plugins, validate the directory structure
	if spec.Source.Type == plugins.SourceTypeLocal {
		// Convert PluginSpec to manifest for validation
		manifest := &validator.PluginManifest{
			Name:        spec.Name,
			Version:     spec.Version,
			Description: spec.Description,
		}
		
		result, err := v.validator.ValidateLoadedPlugin(manifest, binaryPath)
		if err != nil {
			return fmt.Errorf("compliance validation error: %w", err)
		}
		
		if !result.Valid {
			return fmt.Errorf("plugin compliance violations: %v", result.Errors)
		}
	}
	
	return nil
}