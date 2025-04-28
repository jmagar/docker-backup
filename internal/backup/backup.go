package backup

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	// Import the util package

	"docker-backup-tool/internal/util"

	"gopkg.in/yaml.v3"
)

// --- Docker Compose Command Detection (copied from internal/docker) ---
var composeCommand = []string{"docker", "compose"} // Default to v2

func init() {
	// Check if `docker compose` (v2) exists, if not, fallback to `docker-compose` (v1)
	_, err := exec.LookPath("docker")
	if err != nil {
		return // Docker not found, main dependency check handles this
	}
	cmd := exec.Command("docker", "compose", "version")
	if cmd.Run() != nil {
		_, errV1 := exec.LookPath("docker-compose")
		if errV1 == nil {
			composeCommand = []string{"docker-compose"}
			// Log only if verbose is enabled? Need config here, maybe log later.
			// log.Println("Debug: Detected docker-compose (v1) for config parsing")
		} // else: Neither v2 nor v1 seems functional, let execution fail
	} // else: v2 works
}

// --- End Docker Compose Command Detection ---

// Minimal structure to unmarshal docker-compose YAML for volume extraction
// We only care about services and their volumes list.
type ComposeConfig struct {
	Services map[string]Service `yaml:"services"`
	// We could potentially add top-level 'volumes' definition support here too if needed
}

type Service struct {
	Volumes []interface{} `yaml:"volumes"` // Changed from []string to []interface{}
	// Add other fields if needed later, e.g., for direct API interaction
}

// ParseVolumes runs `docker compose config` and extracts unique, existing host paths
// from volume mounts that are prefixed with the specified appdataDir.
func ParseVolumes(composeFilePath string, appdataDir string) ([]string, error) {
	composeFileDir := filepath.Dir(composeFilePath)

	// --- Use docker compose config ---
	cmdArgs := append(composeCommand[1:], "config") // Build args based on detected command
	cmd := exec.Command(composeCommand[0], cmdArgs...)
	cmd.Dir = composeFileDir // Set working directory

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Printf("    Running command: %s %s (in %s)", cmd.Path, strings.Join(cmd.Args[1:], " "), composeFileDir)
	err := cmd.Run()
	if err != nil {
		// Return specific error if docker compose config fails
		return nil, fmt.Errorf("'docker compose config' failed in %s: %w\nStderr: %s", composeFileDir, err, stderr.String())
	}

	// --- Unmarshal the resolved config ---
	data := stdout.Bytes()
	var config ComposeConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		// If unmarshal fails on resolved config, it's a more serious issue
		return nil, fmt.Errorf("failed to unmarshal resolved YAML from 'docker compose config' output for %s: %w", composeFilePath, err)
	}

	appdataPaths := make(map[string]struct{}) // Use map for uniqueness

	for serviceName, service := range config.Services {
		for i, volumeEntry := range service.Volumes { // Iterate through []interface{}
			var hostPath string

			// Determine if volumeEntry is a simple string or a map (long syntax)
			switch v := volumeEntry.(type) {
			case string:
				// Simple volume format: host:container[:options]
				parts := strings.SplitN(v, ":", 2)
				if len(parts) < 2 {
					continue // Skip invalid format or named volumes
				}
				hostPath = strings.TrimSpace(parts[0])
			case map[string]interface{}: // Handle long syntax (map)
				// Check if it's a bind mount and has a source
				if typeVal, ok := v["type"]; ok && typeVal == "bind" {
					if sourceVal, ok := v["source"]; ok {
						if sourceStr, ok := sourceVal.(string); ok {
							hostPath = sourceStr
						} else {
							log.Printf("Warning: Volume entry %d for service '%s' has non-string source. Skipping.", i, serviceName)
							continue
						}
					} else {
						log.Printf("Warning: Bind volume entry %d for service '%s' missing source. Skipping.", i, serviceName)
						continue
					}
				} else {
					// Not a bind mount we care about or type is missing
					continue
				}
			default:
				log.Printf("Warning: Unknown volume format in service '%s': %T. Skipping.", serviceName, v)
				continue
			}

			// --- Process the extracted hostPath (same logic as before) ---
			if hostPath == "" { // Should not happen if logic above is correct, but safety check
				continue
			}

			// Check if it's an absolute path (it should be after config resolution)
			if !filepath.IsAbs(hostPath) {
				log.Printf("Warning: Resolved host path '%s' from service '%s' is not absolute. Skipping.", hostPath, serviceName)
				continue
			}

			cleanedHostPath := filepath.Clean(hostPath)
			cleanedAppdataDir, err := filepath.Abs(filepath.Clean(appdataDir))
			if err != nil {
				return nil, fmt.Errorf("failed to get absolute path for appdataDir '%s': %w", appdataDir, err)
			}

			if strings.HasPrefix(cleanedHostPath, cleanedAppdataDir) {
				_, err := os.Stat(cleanedHostPath)
				if err != nil {
					if os.IsNotExist(err) {
						log.Printf("Warning: Resolved appdata path '%s' from service '%s' does not exist. Skipping.\n", cleanedHostPath, serviceName)
					} else {
						log.Printf("Warning: Error checking resolved appdata path '%s' from service '%s': %v. Skipping.\n", cleanedHostPath, serviceName, err)
					}
					continue
				}
				appdataPaths[cleanedHostPath] = struct{}{}
			}
			// --- End hostPath processing ---
		}
	}

	// Convert map keys (unique paths) to a slice
	uniquePaths := make([]string, 0, len(appdataPaths))
	for path := range appdataPaths {
		uniquePaths = append(uniquePaths, path)
	}

	return uniquePaths, nil
}

// --- Backup Creation ---

// CreateBackup orchestrates the creation of a backup zip file for a project.
// It now accepts excludePatterns.
func CreateBackup(projectName, projectPath, backupDir string, appdataPaths []string, excludePatterns []string) (string, error) {
	// 1. Create Temporary Directory
	// Use os.MkdirTemp in the *parent* of the backupDir or a system temp location?
	// Using backupDir might pollute it if cleanup fails, using system temp is safer.
	// Let's use system temp dir for now.
	tempBackupRoot, err := os.MkdirTemp("", "docker-backup-"+projectName+"-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary backup directory: %w", err)
	}
	// Ensure cleanup happens even on errors
	defer func() {
		if r := recover(); r != nil {
			log.Printf("ERROR: Recovered from panic during backup cleanup for %s: %v\n", projectName, r)
			// Still attempt removal
			os.RemoveAll(tempBackupRoot)
		} else if err != nil {
			// If CreateBackup returns an error, attempt cleanup
			log.Printf("Debug: Cleaning up temp dir %s due to error\n", tempBackupRoot)
			os.RemoveAll(tempBackupRoot)
		} else {
			// If CreateBackup succeeds, defer takes care of it
			log.Printf("Debug: Cleaning up temp dir %s after success\n", tempBackupRoot)
			os.RemoveAll(tempBackupRoot)
		}
	}()

	fmt.Printf("  Using temporary directory: %s\n", tempBackupRoot)

	// 2. Prepare Structure in Temp Dir
	tempComposeTarget := filepath.Join(tempBackupRoot, "compose", projectName)
	tempAppdataParent := filepath.Join(tempBackupRoot, "appdata")

	if err := os.MkdirAll(tempComposeTarget, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp compose structure '%s': %w", tempComposeTarget, err)
	}
	if len(appdataPaths) > 0 {
		if err := os.MkdirAll(tempAppdataParent, 0755); err != nil {
			return "", fmt.Errorf("failed to create temp appdata structure '%s': %w", tempAppdataParent, err)
		}
	}

	// 3. Copy Compose Directory Contents (Respecting Excludes)
	fmt.Printf("  Copying compose directory '%s'...\n", projectPath)
	if err := copyDirectoryContents(projectPath, tempComposeTarget, excludePatterns); err != nil {
		return "", fmt.Errorf("failed to copy compose directory contents: %w", err)
	}

	// 4. Copy Appdata Directory Contents (Respecting Excludes)
	if len(appdataPaths) > 0 {
		fmt.Println("  Copying appdata directories...")
		for _, srcPath := range appdataPaths {
			// Target path preserves the original name inside the temp appdata parent
			targetPath := filepath.Join(tempAppdataParent, filepath.Base(srcPath))
			log.Printf("    [DEBUG Appdata Copy] src: %s, dst: %s", srcPath, targetPath)
			fmt.Printf("    - Copying '%s' to '%s'\n", srcPath, targetPath)
			if err := copyPath(srcPath, targetPath, excludePatterns); err != nil {
				// Log error but potentially continue? Or fail backup?
				// For now, let's fail the backup if any appdata copy fails.
				return "", fmt.Errorf("failed to copy appdata path '%s': %w", srcPath, err)
			}
		}
	}

	// 5. Create Zip Archive
	backupFileName := fmt.Sprintf("%s_%s.zip", projectName, time.Now().Format("20060102"))
	backupFilePath := filepath.Join(backupDir, backupFileName)

	fmt.Printf("  Creating zip archive: %s\n", backupFilePath)
	if err := zipDirectory(tempBackupRoot, backupFilePath); err != nil {
		return "", fmt.Errorf("failed to create zip archive: %w", err)
	}

	// If we reach here, backup succeeded (but cleanup is deferred)
	return backupFilePath, nil
}

// --- Helper Functions ---

// copyPath copies a file or directory recursively, respecting exclude patterns.
// It acts as a dispatcher based on whether the source is a file or directory.
func copyPath(src, dst string, excludePatterns []string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	// Check excludes for the top-level item being copied
	// Use filepath.Base(src) because patterns usually apply relative to the source root
	// (This might need refinement depending on how patterns are intended to be used -
	// matching full path vs relative path)
	// Let's assume patterns are relative to the *copy source root* for now.
	// The walk function handles relative paths internally.
	if info.IsDir() {
		return copyDirectoryContents(src, dst, excludePatterns)
	}
	// For single files, the exclude check needs to happen here if needed,
	// but copyDirectoryContents handles the walk.
	return copyFile(src, dst)
}

// copyDirectoryContents copies the contents of srcDir to dstDir recursively, respecting excludes.
func copyDirectoryContents(srcDir, dstDir string, excludePatterns []string) error {
	// Use WalkDir for better handling of directory skipping
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		// 1. Calculate Relative Path early
		relPath, pathErr := filepath.Rel(srcDir, path)
		if pathErr != nil {
			return fmt.Errorf("failed to calculate relative path for %s: %w", path, pathErr)
		}

		// 2. Check Excludes FIRST using the relative path
		excluded, patternErr := util.MatchesExclude(relPath, excludePatterns)
		if patternErr != nil {
			return fmt.Errorf("exclude pattern error: %w", patternErr) // Invalid pattern stops the walk
		}
		if excluded {
			fmt.Printf("    - Excluding: %s (matches pattern)\n", relPath)
			// Need to determine if it's a directory to skip. Stat the original path.
			info, statErr := os.Stat(path) // Use os.Stat on original path
			if statErr == nil && info.IsDir() {
				return filepath.SkipDir // Skip the entire directory
			}
			// If it's a file or stat failed (but pattern matched), just skip this entry.
			return nil
		}

		// 3. Handle initial WalkDir errors (like permission denied)
		if err != nil {
			// If the error is permission denied BUT we intended to exclude this path anyway, ignore the error.
			if os.IsPermission(err) && excluded {
				// We already logged the exclusion above. Return nil to continue the walk without this item.
				return nil
			}
			// Otherwise, log the error and decide whether to stop.
			fmt.Fprintf(os.Stderr, "Warning: Error accessing path '%s' during initial walk: %v\n", path, err)
			// Let's return the error to stop the walk on unexpected access issues.
			return err
		}

		// Skip the root source directory itself (can be checked after error/exclude checks)
		if relPath == "." {
			return nil
		}

		// Target path in the destination directory
		dstPath := filepath.Join(dstDir, relPath)

		if d.IsDir() {
			// Create the directory in the destination
			err := os.MkdirAll(dstPath, 0755) // Use a default mode, d.Type() might fail if err != nil initially
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to create directory '%s': %v\n", dstPath, err)
				// Decide if this should stop the backup? Let's return error for now.
				return err
			}
			return nil // Successfully created/ensured dir exists
		}

		// Copy the file
		err = copyFile(path, dstPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to copy file '%s' to '%s': %v\n", path, dstPath, err)
			// Return error to stop backup on file copy failure
			return err
		}
		return nil // Continue walk
	})
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// Get source file permissions
	info, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	// Set destination file permissions to match source
	err = os.Chmod(dst, info.Mode())
	if err != nil {
		// Log warning, but don't fail the whole copy? Or fail?
		fmt.Fprintf(os.Stderr, "Warning: Failed to set permissions on '%s': %v\n", dst, err)
	}

	return nil
}

// zipDirectory creates a zip archive of the source directory's contents.
func zipDirectory(sourceDir, targetZipFile string) error {
	zipFile, err := os.Create(targetZipFile)
	if err != nil {
		return fmt.Errorf("failed to create zip file '%s': %w", targetZipFile, err)
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	// Walk through the source directory
	err = filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == sourceDir {
			return nil
		}

		// Get header info from file info
		info, err := d.Info()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Set header name to be relative path inside the zip file
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath) // Use forward slashes for zip standard
		log.Printf("    [DEBUG Zip] Adding path: %s (Source: %s)", header.Name, path)

		// Specify compression method
		header.Method = zip.Deflate

		// Create writer for the file header
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		// If not a directory, copy file contents
		if !d.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		// If walk failed, attempt to remove the potentially corrupt zip file
		os.Remove(targetZipFile)
		return fmt.Errorf("failed during zip creation walk: %w", err)
	}

	return nil
}
