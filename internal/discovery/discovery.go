package discovery

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Project represents a discovered Docker Compose project.
// We might add more fields later (like the specific compose file used).
type Project struct {
	Name            string
	Path            string
	ComposeFilePath string
}

// FindComposeProjects scans a directory for potential Docker Compose project subdirectories.
func FindComposeProjects(composeDir string) ([]Project, error) {
	projects := []Project{}

	entries, err := os.ReadDir(composeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose directory '%s': %w", composeDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			projectPath := filepath.Join(composeDir, entry.Name())
			composeFile, err := FindFirstComposeFile(projectPath)
			if err != nil {
				// Log this? Or let the main loop handle it?
				// Let main loop log the error from FindFirstComposeFile if needed
				log.Printf("Debug: Error finding compose file in %s: %v. Skipping directory.", projectPath, err)
				continue
			}
			if composeFile != "" {
				projects = append(projects, Project{
					Name:            entry.Name(),
					Path:            projectPath,
					ComposeFilePath: composeFile,
				})
			}
		}
	}

	if len(projects) == 0 {
		// Return an empty slice, not an error, if the directory is just empty
		// Main loop handles the "no projects found" message.
		log.Printf("Debug: No projects with compose files found directly in %s", composeDir)
	}

	return projects, nil
}

// FindFirstComposeFile finds the first file with a .yml or .yaml extension in a directory.
func FindFirstComposeFile(dirPath string) (string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", fmt.Errorf("failed to read project directory '%s': %w", dirPath, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			ext := filepath.Ext(entry.Name())
			if ext == ".yml" || ext == ".yaml" {
				return filepath.Join(dirPath, entry.Name()), nil // Found one
			}
		}
	}

	// No compose file found - this is not necessarily an error for FindComposeProjects,
	// but the caller (main loop) should know.
	// log.Printf("Debug: No compose file found in %s", dirPath) // Reduce noise, main logs this
	return "", nil // Return empty string, not an error, if not found
}
