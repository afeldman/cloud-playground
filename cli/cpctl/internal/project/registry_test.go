package project

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectValidation(t *testing.T) {
	tests := []struct {
		name    string
		project *Project
		wantErr bool
	}{
		{
			name: "valid project",
			project: &Project{
				Name:          "test-project",
				Path:          "/tmp/test",
				TerraformPath: "/tmp/test/terraform",
				HelmChartPath: "/tmp/test/charts",
				Status:        StatusReady,
				Namespace:     "dhw2-test-project",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			project: &Project{
				Name:          "",
				Path:          "/tmp/test",
				TerraformPath: "/tmp/test/terraform",
				HelmChartPath: "/tmp/test/charts",
				Status:        StatusReady,
				Namespace:     "dhw2-test-project",
			},
			wantErr: true,
		},
		{
			name: "missing path",
			project: &Project{
				Name:          "test-project",
				Path:          "",
				TerraformPath: "/tmp/test/terraform",
				HelmChartPath: "/tmp/test/charts",
				Status:        StatusReady,
				Namespace:     "dhw2-test-project",
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			project: &Project{
				Name:          "test-project",
				Path:          "/tmp/test",
				TerraformPath: "/tmp/test/terraform",
				HelmChartPath: "/tmp/test/charts",
				Status:        "invalid-status",
				Namespace:     "dhw2-test-project",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProject(tt.project)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegistrySaveLoad(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "projects.yaml")

	// Create a registry with test data
	registry := &Registry{
		Projects: make(map[string]*Project),
		filePath: registryPath,
	}

	// Add test projects
	now := time.Now()
	projects := []*Project{
		{
			Name:           "airbyte",
			Path:           "/tmp/dhw2/airbyte",
			TerraformPath:  "/tmp/dhw2/airbyte/terraform",
			HelmChartPath:  "/tmp/dhw2/airbyte/charts",
			Status:         StatusReady,
			Namespace:      "dhw2-airbyte",
			LastActive:     now,
			LastDeployed:   now.Add(-2 * time.Hour),
			DeployedBy:     "test-user",
		},
		{
			Name:           "datalynq-alfa",
			Path:           "/tmp/dhw2/datalynq-alfa",
			TerraformPath:  "/tmp/dhw2/datalynq-alfa/terraform",
			HelmChartPath:  "/tmp/dhw2/datalynq-alfa/charts",
			Status:         StatusDeployed,
			Namespace:      "dhw2-datalynq-alfa",
			LastActive:     now.Add(-1 * time.Hour),
		},
	}

	for _, p := range projects {
		registry.Projects[p.Name] = p
	}

	// Save the registry
	err := registry.Save()
	require.NoError(t, err, "should save registry without error")

	// Verify file exists
	_, err = os.Stat(registryPath)
	require.NoError(t, err, "registry file should exist")

	// Load into a new registry
	loadedRegistry := &Registry{
		Projects: make(map[string]*Project),
		filePath: registryPath,
	}
	err = loadedRegistry.load()
	require.NoError(t, err, "should load registry without error")

	// Verify loaded data matches saved data
	assert.Equal(t, len(registry.Projects), len(loadedRegistry.Projects), "should have same number of projects")

	for name, expectedProject := range registry.Projects {
		actualProject, exists := loadedRegistry.Projects[name]
		require.True(t, exists, "project %s should exist in loaded registry", name)
		
		assert.Equal(t, expectedProject.Name, actualProject.Name)
		assert.Equal(t, expectedProject.Path, actualProject.Path)
		assert.Equal(t, expectedProject.TerraformPath, actualProject.TerraformPath)
		assert.Equal(t, expectedProject.HelmChartPath, actualProject.HelmChartPath)
		assert.Equal(t, expectedProject.Status, actualProject.Status)
		assert.Equal(t, expectedProject.Namespace, actualProject.Namespace)
		assert.Equal(t, expectedProject.DeployedBy, actualProject.DeployedBy)
		
		// Compare timestamps with tolerance
		assert.WithinDuration(t, expectedProject.LastActive, actualProject.LastActive, time.Second)
		if !expectedProject.LastDeployed.IsZero() {
			assert.WithinDuration(t, expectedProject.LastDeployed, actualProject.LastDeployed, time.Second)
		}
	}
}

func TestRegistryOperations(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "projects.yaml")

	registry := &Registry{
		Projects: make(map[string]*Project),
		filePath: registryPath,
	}

	// Test GetProject on empty registry
	_, err := registry.GetProject("nonexistent")
	assert.Error(t, err, "should error when project not found")

	// Test UpdateProject
	project := &Project{
		Name:          "test-project",
		Path:          "/tmp/test",
		TerraformPath: "/tmp/test/terraform",
		HelmChartPath: "/tmp/test/charts",
		Status:        StatusReady,
		Namespace:     "dhw2-test-project",
	}

	err = registry.UpdateProject(project)
	require.NoError(t, err, "should update project without error")

	// Test GetProject
	retrieved, err := registry.GetProject("test-project")
	require.NoError(t, err, "should get project without error")
	assert.Equal(t, project.Name, retrieved.Name)
	assert.Equal(t, project.Status, retrieved.Status)

	// Test ListProjects
	projects := registry.ListProjects()
	assert.Len(t, projects, 1, "should have one project")
	assert.Equal(t, "test-project", projects[0].Name)

	// Test DeleteProject
	err = registry.DeleteProject("test-project")
	require.NoError(t, err, "should delete project without error")

	_, err = registry.GetProject("test-project")
	assert.Error(t, err, "should error after deletion")

	// Test deleting non-existent project
	err = registry.DeleteProject("nonexistent")
	assert.Error(t, err, "should error when deleting non-existent project")
}

func TestIsProjectDirectory(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create a valid project directory
	projectDir := filepath.Join(tmpDir, "valid-project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "terraform"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "charts"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "Taskfile.yml"), []byte("tasks: []"), 0644))

	// Create an invalid project directory (missing Taskfile.yml)
	invalidDir1 := filepath.Join(tmpDir, "invalid-project-1")
	require.NoError(t, os.MkdirAll(filepath.Join(invalidDir1, "terraform"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(invalidDir1, "charts"), 0755))

	// Create another invalid project directory (missing charts)
	invalidDir2 := filepath.Join(tmpDir, "invalid-project-2")
	require.NoError(t, os.MkdirAll(filepath.Join(invalidDir2, "terraform"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(invalidDir2, "Taskfile.yml"), []byte("tasks: []"), 0644))

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"valid project", projectDir, true},
		{"missing Taskfile", invalidDir1, false},
		{"missing charts", invalidDir2, false},
		{"non-existent directory", filepath.Join(tmpDir, "nonexistent"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isProjectDirectory(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAutoDiscover(t *testing.T) {
	// Create a temporary DHW2-like directory structure
	tmpDir := t.TempDir()
	dhw2Dir := filepath.Join(tmpDir, "dhw2")

	// Create valid projects
	projects := []string{"airbyte", "datalynq-alfa", "zeroetl"}
	for _, project := range projects {
		projectDir := filepath.Join(dhw2Dir, project)
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "terraform"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "charts"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(projectDir, "Taskfile.yml"), []byte("tasks: []"), 0644))
	}

	// Create a non-project directory
	nonProjectDir := filepath.Join(dhw2Dir, "docs")
	require.NoError(t, os.MkdirAll(nonProjectDir, 0755))

	// Create a nested structure (should be skipped)
	nestedDir := filepath.Join(dhw2Dir, "airbyte", "src")
	require.NoError(t, os.MkdirAll(nestedDir, 0755))

	// Create registry
	tmpRegistryPath := filepath.Join(t.TempDir(), "projects.yaml")
	registry := &Registry{
		Projects: make(map[string]*Project),
		filePath: tmpRegistryPath,
	}

	// Run auto-discovery
	err := registry.AutoDiscover(dhw2Dir)
	require.NoError(t, err, "auto-discovery should not error")

	// Verify discovered projects
	assert.Len(t, registry.Projects, 3, "should discover 3 projects")

	for _, projectName := range projects {
		project, exists := registry.Projects[projectName]
		require.True(t, exists, "project %s should be discovered", projectName)
		assert.Equal(t, projectName, project.Name)
		assert.Equal(t, filepath.Join(dhw2Dir, projectName), project.Path)
		assert.Equal(t, fmt.Sprintf("dhw2-%s", projectName), project.Namespace)
		assert.Equal(t, StatusReady, project.Status)
	}

	// Verify non-project directory was not added
	_, exists := registry.Projects["docs"]
	assert.False(t, exists, "non-project directory should not be discovered")
}

// validateProject is a helper for testing project validation
func validateProject(p *Project) error {
	if p.Name == "" {
		return fmt.Errorf("project name is required")
	}
	if p.Path == "" {
		return fmt.Errorf("project path is required")
	}
	
	// Check if status is valid
	validStatuses := map[ProjectStatus]bool{
		StatusUnknown:   true,
		StatusReady:     true,
		StatusDeployed:  true,
		StatusDeploying: true,
		StatusFailed:    true,
		StatusTearingDown: true,
	}
	
	if !validStatuses[p.Status] {
		return fmt.Errorf("invalid project status: %s", p.Status)
	}
	
	return nil
}