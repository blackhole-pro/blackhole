// Package validator provides plugin compliance validation
package validator

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComplianceValidator validates plugin compliance at runtime
type ComplianceValidator struct {
	strictMode bool
}

// NewComplianceValidator creates a new compliance validator
func NewComplianceValidator(strictMode bool) *ComplianceValidator {
	return &ComplianceValidator{
		strictMode: strictMode,
	}
}

// ValidationResult contains the results of validation
type ValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

// PluginManifest represents the minimal required fields
type PluginManifest struct {
	Name         string `yaml:"name"`
	Version      string `yaml:"version"`
	Description  string `yaml:"description"`
	Type         string `yaml:"type"`
	Architecture []string `yaml:"architecture"`
	
	Resources struct {
		MinMemory string `yaml:"min_memory"`
		MaxMemory string `yaml:"max_memory"`
		MinCPU    float64 `yaml:"min_cpu"`
		MaxCPU    float64 `yaml:"max_cpu"`
	} `yaml:"resources"`
	
	Capabilities []string `yaml:"capabilities"`
	
	Mesh struct {
		Enabled     bool     `yaml:"enabled"`
		ServiceName string   `yaml:"service_name"`
		Subscribe   []string `yaml:"subscribe_patterns"`
		Publish     []string `yaml:"publish_patterns"`
	} `yaml:"mesh"`
	
	GRPC struct {
		Service string `yaml:"service"`
	} `yaml:"grpc"`
	
	Dependencies struct {
		Core    string `yaml:"core"`
		Plugins []struct {
			Name     string `yaml:"name"`
			Version  string `yaml:"version"`
			Optional bool   `yaml:"optional"`
		} `yaml:"plugins"`
	} `yaml:"dependencies"`
}

// ValidatePluginPackage validates a plugin package file
func (v *ComplianceValidator) ValidatePluginPackage(packagePath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Open the package file
	file, err := os.Open(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open package: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Track required files
	requiredFiles := map[string]bool{
		"plugin.yaml": false,
		"bin/":        false,
		"proto/":      false,
	}

	var manifest *PluginManifest
	foundProtoService := false

	// Read through the archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar: %w", err)
		}

		// Check for required files/directories
		for required := range requiredFiles {
			if strings.HasPrefix(header.Name, required) || header.Name == required {
				requiredFiles[required] = true
			}
		}

		// Parse plugin.yaml
		if header.Name == "plugin.yaml" || strings.HasSuffix(header.Name, "/plugin.yaml") {
			data, err := io.ReadAll(tarReader)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to read plugin.yaml: %v", err))
				result.Valid = false
				continue
			}

			manifest = &PluginManifest{}
			if err := yaml.Unmarshal(data, manifest); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to parse plugin.yaml: %v", err))
				result.Valid = false
			}
		}

		// Check for proto files with service definitions
		if strings.HasSuffix(header.Name, ".proto") {
			data, err := io.ReadAll(tarReader)
			if err == nil && strings.Contains(string(data), "service") {
				foundProtoService = true
			}
		}
	}

	// Check for missing required files
	for file, found := range requiredFiles {
		if !found {
			result.Errors = append(result.Errors, fmt.Sprintf("Required file/directory missing: %s", file))
			result.Valid = false
		}
	}

	// Validate manifest
	if manifest != nil {
		v.validateManifest(manifest, result)
	} else {
		result.Errors = append(result.Errors, "plugin.yaml not found or could not be parsed")
		result.Valid = false
	}

	// Check for gRPC service definition
	if !foundProtoService && v.strictMode {
		result.Errors = append(result.Errors, "No gRPC service definition found in proto files")
		result.Valid = false
	} else if !foundProtoService {
		result.Warnings = append(result.Warnings, "No gRPC service definition found in proto files")
	}

	return result, nil
}

// validateManifest validates the plugin manifest
func (v *ComplianceValidator) validateManifest(manifest *PluginManifest, result *ValidationResult) {
	// Required fields
	if manifest.Name == "" {
		result.Errors = append(result.Errors, "Plugin name is required")
		result.Valid = false
	}

	if manifest.Version == "" {
		result.Errors = append(result.Errors, "Plugin version is required")
		result.Valid = false
	}

	if manifest.Description == "" {
		result.Errors = append(result.Errors, "Plugin description is required")
		result.Valid = false
	}

	// Resource requirements
	if manifest.Resources.MinMemory == "" || manifest.Resources.MaxMemory == "" {
		result.Errors = append(result.Errors, "Memory resource limits are required")
		result.Valid = false
	}

	if manifest.Resources.MinCPU == 0 || manifest.Resources.MaxCPU == 0 {
		result.Errors = append(result.Errors, "CPU resource limits are required")
		result.Valid = false
	}

	// Capabilities
	if len(manifest.Capabilities) == 0 {
		result.Warnings = append(result.Warnings, "No capabilities declared")
	}

	// Mesh compliance
	if manifest.Mesh.Enabled {
		if manifest.Mesh.ServiceName == "" {
			result.Errors = append(result.Errors, "Mesh service name is required when mesh is enabled")
			result.Valid = false
		}

		if manifest.GRPC.Service == "" {
			result.Errors = append(result.Errors, "gRPC service definition is required for mesh-enabled plugins")
			result.Valid = false
		}

		if len(manifest.Mesh.Subscribe) == 0 && len(manifest.Mesh.Publish) == 0 {
			result.Warnings = append(result.Warnings, "Mesh-enabled plugin doesn't subscribe or publish any events")
		}
	} else if v.strictMode {
		result.Warnings = append(result.Warnings, "Plugin is not mesh-enabled")
	}

	// Check for direct plugin dependencies
	for _, dep := range manifest.Dependencies.Plugins {
		if !dep.Optional && v.strictMode {
			result.Warnings = append(result.Warnings, 
				fmt.Sprintf("Direct dependency on plugin '%s' may violate loose coupling", dep.Name))
		}
	}
}

// ValidateLoadedPlugin validates a loaded plugin at runtime
func (v *ComplianceValidator) ValidateLoadedPlugin(manifest *PluginManifest, pluginPath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Check for required directories
	requiredDirs := []string{
		filepath.Join(pluginPath, "types"),
		filepath.Join(pluginPath, "proto", "v1"),
	}

	for _, dir := range requiredDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Expected directory not found: %s", dir))
		}
	}

	// Check for typed errors
	errorsFile := filepath.Join(pluginPath, "types", "errors.go")
	if _, err := os.Stat(errorsFile); os.IsNotExist(err) {
		result.Errors = append(result.Errors, "types/errors.go not found - typed errors are required")
		result.Valid = false
	}

	// Validate manifest
	v.validateManifest(manifest, result)

	return result, nil
}

// Helper function to check if plugin follows communication rules
func (v *ComplianceValidator) CheckCommunicationCompliance(pluginCode []byte) []string {
	violations := []string{}
	codeStr := string(pluginCode)

	// Check for direct plugin imports
	if strings.Contains(codeStr, `"core/pkg/plugins/`) {
		violations = append(violations, "Direct plugin imports detected - use mesh communication instead")
	}

	// Check for standard log usage
	if strings.Contains(codeStr, `import "log"`) || strings.Contains(codeStr, `import log "log"`) {
		violations = append(violations, "Standard log package imported - use structured logging instead")
	}

	// Check for generic error usage
	if strings.Contains(codeStr, `errors.New(`) && !strings.Contains(codeStr, `type.*Error struct`) {
		violations = append(violations, "Generic errors used - define typed errors instead")
	}

	return violations
}