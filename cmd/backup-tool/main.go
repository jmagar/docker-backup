package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	// Use the actual module path defined in go.mod
	"docker-backup-tool/internal/config"
	// Import the new discovery package
	"docker-backup-tool/internal/backup"
	"docker-backup-tool/internal/discovery"

	// Import the docker package
	"docker-backup-tool/internal/docker"
	// Import the rsync package
	"docker-backup-tool/internal/rsync"
	// Import the pflag package
	// "github.com/spf13/pflag"

	"docker-backup-tool/internal/logutil"
)

// Global variable to store the detected docker compose command
var dockerComposeCmd string

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		logutil.Fatal("Error loading configuration: %v", err)
	}

	// --- Initialize Logger ---
	logutil.Init(
		cfg.LogFile,
		cfg.Verbose,
		cfg.LogRotationMaxSizeMB,
		cfg.LogRotationMaxBackups,
		cfg.LogRotationMaxAgeDays,
		cfg.LogRotationCompress,
	)
	defer logutil.Close() // Ensure log file is closed on exit

	// Dependency Checks
	// Docker Compose
	cmdV2, errV2 := exec.LookPath("docker") // Check for 'docker' first (implies v2+)
	cmdV1, errV1 := exec.LookPath("docker-compose")

	if errV2 == nil {
		// Check if 'docker compose' subcommand works (basic check)
		cmd := exec.Command(cmdV2, "compose", "version")
		if cmd.Run() == nil {
			dockerComposeCmd = cmdV2 // Use 'docker' command
			logutil.Info("Using Docker Compose v2+ (docker compose)")
		} else if errV1 == nil {
			dockerComposeCmd = cmdV1 // Fallback to v1
			logutil.Info("Using Docker Compose v1 (docker-compose)")
		} else {
			logutil.Fatal("Docker Compose command not found or not working. Please install Docker Compose (v1 or v2).")
		}
	} else if errV1 == nil {
		dockerComposeCmd = cmdV1 // Use v1 if v2 check failed but v1 exists
		logutil.Info("Using Docker Compose v1 (docker-compose)")
	} else {
		logutil.Fatal("Docker Compose command not found. Please install Docker Compose (v1 or v2).")
	}

	// Rsync (only if enabled)
	if cfg.Rsync.Enabled {
		if _, err := exec.LookPath(cfg.Rsync.Command); err != nil {
			logutil.Fatal("Rsync command '%s' not found in PATH. Please install rsync or correct the rsync.command configuration.", cfg.Rsync.Command)
		}
		logutil.Info("Rsync enabled. Using command: %s", cfg.Rsync.Command)
	}

	// Optionally print loaded config if verbose
	if cfg.Verbose {
		logutil.Info("Configuration loaded: %+v", cfg)
	}

	// Always log basic paths
	logutil.Info("Using Compose Dir: %s", cfg.ComposeDir)
	logutil.Info("Using Appdata Dir: %s", cfg.AppdataDir)
	logutil.Info("Using Backup Dir : %s", cfg.BackupDir)

	// --- Dry Run Check ---
	if cfg.DryRun {
		logutil.Warn("--- DRY RUN MODE ENABLED --- Actions will be logged but not executed.")
	}

	// --- Discover Projects ---
	logutil.Info("Starting project discovery...")

	projects, err := discovery.FindComposeProjects(cfg.ComposeDir)
	if err != nil {
		logutil.Fatal("Error finding compose projects in '%s': %v", cfg.ComposeDir, err)
	}

	if len(projects) == 0 {
		logutil.Fatal("No Docker Compose projects found in %s. Exiting.", cfg.ComposeDir)
	}

	logutil.Info("Discovered %d projects:", len(projects))

	// --- Main Processing Loop ---
	backupSuccess := true // Track overall success
	failedProjects := 0
	successfulProjects := 0

	for _, project := range projects {
		projectName := project.Name
		projectFailed := false // Track individual project failure
		logutil.Info("=== Processing Project: %s ===", projectName)

		// 1. Stop Stack
		if cfg.DryRun {
			logutil.Info("[DRY RUN] Would stop stack for project %s (path: %s)", projectName, project.Path)
		} else {
			logutil.Info("[%s] Stopping stack...", projectName)
			if err := docker.Down(project.Path, dockerComposeCmd); err != nil {
				logutil.Error("Error stopping stack for project %s: %v", projectName, err)
				projectFailed = true
				// Don't continue yet, still try to backup compose files etc.
			}
		}

		// 2. Verify Stack Down
		if !projectFailed { // Only check if stop didn't already report an error
			if !cfg.DryRun {
				// --- Execute real verification only if not in dry run ---
				logutil.Info("[%s] Verifying stack is down...", projectName)
				running, err := docker.PsQuiet(project.Path, dockerComposeCmd)
				if err != nil {
					logutil.Error("ERROR: Failed to check stack status for %s: %v. Skipping backup steps.", projectName, err)
					projectFailed = true
				} else if running {
					logutil.Error("ERROR: Stack for %s is still running after 'down' command. Skipping backup steps.", projectName)
					projectFailed = true
				} else {
					logutil.Info("Stack verified down for %s.", projectName)
				}
			}
		} else {
			logutil.Warn("[%s] Skipping stack verification because stop command failed.", projectName)
		}

		// Dry run simulation/override for verification step
		if cfg.DryRun {
			if projectFailed { // If the simulated stop failed (though currently it doesn't)
				logutil.Info("[DRY RUN] Skipping stack verification simulation as stop was skipped/failed.")
			} else {
				logutil.Info("[DRY RUN] Would verify stack is down for %s.", projectName)
				// Assume stack is down for dry run to proceed to backup simulation
				projectFailed = false // Reset potential failure from simulated stop
			}
		} // else: Real verification logic handled above within the !cfg.DryRun block

		// 3. Parse Volumes
		var appdataPaths []string
		if !projectFailed { // Only parse if stack is confirmed down
			logutil.Info("[%s] Parsing compose file %s for appdata volumes...", projectName, project.ComposeFilePath)
			appdataPaths, err = backup.ParseVolumes(project.ComposeFilePath, cfg.AppdataDir, dockerComposeCmd)
			if err != nil {
				logutil.Error("ERROR: Failed to parse volumes from %s: %v. Backup will not include appdata.", project.ComposeFilePath, err)
				// Continue without appdata, don't fail the whole project for this.
			}
			if cfg.Verbose {
				logutil.Debug("[DEBUG Appdata] Parsed appdata paths: %v", appdataPaths) // Use Debug for verbose
			}
			if len(appdataPaths) > 0 {
				logutil.Info("Found %d appdata paths to include.", len(appdataPaths)) // Keep as Info
				for _, ap := range appdataPaths {
					logutil.Info("    - %s", ap)
				}
			}
		} else {
			logutil.Warn("[%s] Skipping volume parsing because previous steps failed.", projectName)
		}

		// 4. Create Backup
		var backupFile string
		if !projectFailed { // Only create if stack is confirmed down or dry run
			if cfg.DryRun {
				logutil.Info("[DRY RUN] Would create backup for %s.", projectName)
				logutil.Info("[DRY RUN]   Compose Path: %s", project.Path)
				logutil.Info("[DRY RUN]   Appdata Paths (%d):", len(appdataPaths))
				for _, p := range appdataPaths {
					logutil.Info("[DRY RUN]     - %s", p)
				}
				logutil.Info("[DRY RUN]   Exclusions Applied: %d patterns", len(cfg.Exclude))
				// Simulate a successful backup file path for potential rsync dry run
				backupFile = filepath.Join(cfg.BackupDir, projectName+"_DRYRUN.zip")
			} else {
				logutil.Info("[%s] Creating backup...", projectName)
				// Pass the full cfg object
				backupFile, err = backup.CreateBackup(projectName, project.Path, cfg.BackupDir, appdataPaths, cfg)
				if err != nil {
					logutil.Error("ERROR: Failed to create backup for %s: %v.", projectName, err)
					projectFailed = true
				} else {
					logutil.Success("Successfully created backup: %s", backupFile) // Use Success
				}
			}
		} else {
			logutil.Warn("[%s] Skipping backup creation because previous steps failed.", projectName)
		}

		// --- Rsync Transfer (Optional) ---
		if cfg.Rsync.Enabled {
			if projectFailed {
				logutil.Warn("[%s] Skipping rsync because previous steps failed.", projectName)
			} else if backupFile == "" {
				logutil.Warn("[%s] Skipping rsync because backup file was not created (likely due to previous errors).", projectName)
			} else if cfg.Rsync.Destination == "" {
				logutil.Warn("[%s] Skipping rsync because rsync.destination is not set.", projectName)
			} else {
				if cfg.DryRun {
					logutil.Info("[DRY RUN] Would transfer %s to %s using rsync.", backupFile, cfg.Rsync.Destination)
					// Simulate success for dry run by not setting projectFailed=true
				} else {
					// --- Execute real rsync only if not in dry run ---
					logutil.Info("[%s] Rsync enabled. Transferring %s to %s...", projectName, backupFile, cfg.Rsync.Destination)
					if err := rsync.TransferBackup(cfg, backupFile); err != nil {
						logutil.Error("ERROR: Rsync transfer failed for %s: %v", projectName, err)
						projectFailed = true // Mark project as failed if rsync fails
					} else {
						logutil.Success("[%s] Rsync transfer successful.", projectName)
					}
				}
			}
		}

		// 5. Optional: Restart Stack
		if cfg.RestartAfterBackup {
			logutil.Info("[%s] Restart requested.", projectName)
			if projectFailed {
				logutil.Warn("[%s] Skipping restart because previous steps failed.", projectName)
			} else {
				if cfg.DryRun {
					if cfg.PullBeforeRestart {
						logutil.Info("[DRY RUN] Would pull latest images for %s.", projectName)
					}
					logutil.Info("[DRY RUN] Would start stack %s.", projectName)
					// Assume success for dry run
				} else {
					// Restart logic
					if cfg.PullBeforeRestart {
						logutil.Info("[%s] Pulling latest images...", projectName)
						if err := docker.Pull(project.Path, dockerComposeCmd); err != nil {
							logutil.Error("ERROR: Failed to pull images for %s: %v", projectName, err)
							// Continue to attempt restart even if pull fails
						} else {
							logutil.Info("[%s] Image pull successful.", projectName)
						}
					}
					logutil.Info("[%s] Starting stack...", projectName)
					if err := docker.UpDetached(project.Path, dockerComposeCmd); err != nil {
						logutil.Error("ERROR: Failed to start stack %s after backup: %v", projectName, err)
						projectFailed = true // Mark project as failed if restart fails
					} else {
						logutil.Success("[%s] Stack started successfully.", projectName)
					}
				}
			}
		} // End RestartAfterBackup

		// --- Final project status log ---
		if projectFailed {
			backupSuccess = false // Mark overall process as failed
			failedProjects++
			logutil.Error("--- Finished project %s with ERRORS ---", projectName)
		} else {
			successfulProjects++
			logutil.Success("--- Finished project %s successfully ---", projectName)
		}
		fmt.Println()
	}

	// --- Final Summary ---
	logutil.Info("=============================")
	logutil.Info("Backup process finished. Successful: %d, Failed: %d", successfulProjects, failedProjects)
	logutil.Info("=============================")
	if !backupSuccess {
		logutil.Warn("One or more projects failed to back up correctly. Check logs above.")
		os.Exit(1)
	} else {
		logutil.Success("All projects processed successfully.")
	}
}
