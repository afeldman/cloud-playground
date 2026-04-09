package project

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ProjectStatus represents the deployment status of a project
type ProjectStatus string

const (
	StatusUnknown   ProjectStatus = "unknown"
	StatusReady     ProjectStatus = "ready"
	StatusDeployed  ProjectStatus = "deployed"
	StatusDeploying ProjectStatus = "deploying"
	StatusFailed    ProjectStatus = "failed"
	StatusTearingDown ProjectStatus = "tearing_down"
)

// Project represents a DHW2 project with its configuration
type Project struct {
	Name           string        `yaml:"name"`
	Path           string        `yaml:"path"`
	TerraformPath  string        `yaml:"terraform_path"`
	HelmChartPath  string        `yaml:"helm_chart_path"`
	Status         ProjectStatus `yaml:"status"`
	Namespace      string        `yaml:"namespace"`
	LastActive     time.Time     `yaml:"last_active,omitempty"`
	LastDeployed   time.Time     `yaml:"last_deployed,omitempty"`
	DeployedBy     string        `yaml:"deployed_by,omitempty"`
}

// Registry manages the collection of projects
type Registry struct {
	Projects map[string]*Project `yaml:"projects"`
	filePath string
}

// NewRegistry creates a new registry instance
func NewRegistry() (*Registry, error) {
	// Determine config directory
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	filePath := filepath.Join(configDir, "projects.yaml")
	registry := &Registry{
		Projects: make(map[string]*Project),
		filePath: filePath,
	}

	// Try to load existing registry
	if err := registry.load(); err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("failed to load registry", "error", err)
		}
		// If file doesn't exist or can't be loaded, we'll auto-discover
	}

	// Auto-discover projects if registry is empty
	if len(registry.Projects) == 0 {
		slog.Info("registry empty, auto-discovering projects")
		if err := registry.AutoDiscover(""); err != nil {
			slog.Warn("auto-discovery failed", "error", err)
		}
	}

	return registry, nil
}

// AutoDiscover scans the DHW2 directory for projects
func (r *Registry) AutoDiscover(scannerPath string) error {
	if scannerPath == "" {
		// Default to DHW2 directory
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		scannerPath = filepath.Join(home, "Projects", "birdy", "dhw2")
	}

	slog.Debug("scanning for projects", "path", scannerPath)

	// Check if directory exists
	if _, err := os.Stat(scannerPath); os.IsNotExist(err) {
		slog.Warn("DHW2 directory not found", "path", scannerPath)
		return nil
	}

	// Walk directory to find projects
	err := filepath.WalkDir(scannerPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip non-directories
		if !d.IsDir() {
			return nil
		}

		// Check if this directory contains a project
		if isProjectDirectory(path) {
			projectName := filepath.Base(path)
			
			// Check if project already exists in registry
			if _, exists := r.Projects[projectName]; !exists {
				project := &Project{
					Name:           projectName,
					Path:           path,
					TerraformPath:  filepath.Join(path, "terraform"),
					HelmChartPath:  filepath.Join(path, "charts"),
					Status:         StatusReady,
					Namespace:      fmt.Sprintf("dhw2-%s", projectName),
					LastActive:     time.Now(),
				}
				r.Projects[projectName] = project
				slog.Debug("discovered project", "name", projectName, "path", path)
			}
			
			// Skip subdirectories of this project
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	slog.Info("auto-discovery complete", "projects_found", len(r.Projects))
	
	// Save the discovered projects
	if err := r.Save(); err != nil {
		return fmt.Errorf("failed to save discovered projects: %w", err)
	}

	return nil
}

// isProjectDirectory checks if a directory contains a project
func isProjectDirectory(path string) bool {
	// Check for required directories/files
	required := []string{
		filepath.Join(path, "terraform"),
		filepath.Join(path, "charts"),
		filepath.Join(path, "Taskfile.yml"),
	}

	for _, req := range required {
		if _, err := os.Stat(req); os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// Save persists the registry to disk
func (r *Registry) Save() error {
	data, err := yaml.Marshal(r)
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry file: %w", err)
	}

	slog.Debug("registry saved", "path", r.filePath, "projects", len(r.Projects))
	return nil
}

// load reads the registry from disk
func (r *Registry) load() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, r); err != nil {
		return fmt.Errorf("failed to unmarshal registry: %w", err)
	}

	slog.Debug("registry loaded", "path", r.filePath, "projects", len(r.Projects))
	return nil
}

// GetProject returns a project by name
func (r *Registry) GetProject(name string) (*Project, error) {
	project, exists := r.Projects[name]
	if !exists {
		return nil, fmt.Errorf("project not found: %s", name)
	}
	return project, nil
}

// UpdateProject updates a project in the registry
func (r *Registry) UpdateProject(project *Project) error {
	r.Projects[project.Name] = project
	project.LastActive = time.Now()
	return r.Save()
}

// DeleteProject removes a project from the registry
func (r *Registry) DeleteProject(name string) error {
	if _, exists := r.Projects[name]; !exists {
		return fmt.Errorf("project not found: %s", name)
	}
	delete(r.Projects, name)
	return r.Save()
}

// ListProjects returns all projects in the registry
func (r *Registry) ListProjects() []*Project {
	projects := make([]*Project, 0, len(r.Projects))
	for _, project := range r.Projects {
		projects = append(projects, project)
	}
	return projects
}

// getConfigDir returns the cpctl configuration directory
func getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cpctl"), nil
}