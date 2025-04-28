package main

import (
	"log"
	"os"
	"os/exec" // Needed for LookPath

	// Use the actual module path defined in go.mod
	"docker-backup-tool/internal/config"
	// Import the new discovery package
	"docker-backup-tool/internal/backup"
	"docker-backup-tool/internal/discovery"

	// Import the docker package
	"docker-backup-tool/internal/docker"
	// Import the rsync package
	"docker-backup-tool/internal/rsync"
)

func main() {
	// Setup basic logging
	log.SetFlags(log.Ldate | log.Ltime)
	log.Println("Starting Docker Backup Tool...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v\n", err)
	}

	// --- Dependency Checks ---
	log.Println("Checking external dependencies...")
	// Check for docker compose (v2 or v1)
	_, errV2 := exec.LookPath("docker")
	lookPathErr := false
	if errV2 == nil {
		// Found docker, check for 'docker compose' subcommand implicitly via docker.go init()
		// This is slightly indirect, maybe improve later if docker.go init fails?
	} else {
		_, errV1 := exec.LookPath("docker-compose")
		if errV1 != nil {
			log.Println("ERROR: Neither 'docker' (for compose v2) nor 'docker-compose' (v1) found in PATH.")
			lookPathErr = true
		}
	}
	// Check for rsync if enabled
	if cfg.Rsync.Enabled {
		if _, err := exec.LookPath("rsync"); err != nil {
			log.Println("ERROR: Rsync is enabled but 'rsync' command not found in PATH.")
			lookPathErr = true
		}
	}
	if lookPathErr {
		log.Fatalln("Required external dependencies not found. Exiting.")
	}
	log.Println("External dependencies found.")

	// Optionally print loaded config if verbose
	if cfg.Verbose {
		log.Printf("Configuration loaded: %+v\n", cfg)
	}

	// Always log basic paths
	log.Printf("Using Compose Dir: %s", cfg.ComposeDir)
	log.Printf("Using Appdata Dir: %s", cfg.AppdataDir)
	log.Printf("Using Backup Dir : %s", cfg.BackupDir)

	log.Println("Starting project discovery...")

	projects, err := discovery.FindComposeProjects(cfg.ComposeDir)
	if err != nil {
		log.Fatalf("Error finding compose projects in '%s': %v\n", cfg.ComposeDir, err)
	}

	if len(projects) == 0 {
		log.Printf("No valid projects with compose files found in %s.\n", cfg.ComposeDir)
		os.Exit(0)
	}

	log.Printf("Found %d projects. Starting backup process...\n", len(projects))
	successfulBackups := 0
	failedBackups := 0

	for _, p := range projects {
		log.Printf("--- Processing project: %s ---", p.Name)
		projectFailed := false

		// 1. Stop Stack
		log.Printf("  Stopping stack for %s...", p.Name)
		if err := docker.Down(p.Path); err != nil {
			log.Printf("  Warning: Failed to stop stack for %s: %v. Will check status.", p.Name, err)
			// Don't necessarily fail the project yet, maybe it's already down.
		}

		// 2. Verify Stack Down
		log.Printf("  Verifying stack is down for %s...", p.Name)
		running, err := docker.PsQuiet(p.Path)
		if err != nil {
			log.Printf("  ERROR: Failed to check stack status for %s: %v. Skipping backup.", p.Name, err)
			projectFailed = true
		} else if running {
			log.Printf("  ERROR: Stack for %s is still running after 'down' command. Skipping backup.", p.Name)
			projectFailed = true
		} else {
			log.Printf("  Stack verified down for %s.", p.Name)
		}

		// If project failed during stop/check, skip to next project
		if projectFailed {
			failedBackups++
			log.Printf("--- Finished project %s with ERRORS (Before Backup Attempt) ---", p.Name)
			log.Println("") // Space for readability
			continue        // Move to the next project in the loop
		}

		// 3. Parse Volumes
		var appdataPaths []string
		log.Printf("  Parsing compose file %s for appdata volumes...", p.ComposeFilePath)
		appdataPaths, err = backup.ParseVolumes(p.ComposeFilePath, cfg.AppdataDir)
		if err != nil {
			log.Printf("  ERROR: Failed to parse volumes from %s: %v. Backup will not include appdata.", p.ComposeFilePath, err)
			// Continue without appdata, don't fail the whole project for this.
		}
		if cfg.Verbose {
			log.Printf("  [DEBUG Appdata] Parsed appdata paths: %v", appdataPaths)
		}
		if cfg.Verbose && len(appdataPaths) > 0 {
			log.Printf("  Found %d appdata paths to include:", len(appdataPaths))
			for _, ap := range appdataPaths {
				log.Printf("    - %s", ap)
			}
		}

		// 4. Create Backup
		var backupFilePath string
		log.Printf("  Creating backup for %s...", p.Name)
		backupFilePath, err = backup.CreateBackup(p.Name, p.Path, cfg.BackupDir, appdataPaths, cfg.Exclude)
		if err != nil {
			log.Printf("  ERROR: Failed to create backup for %s: %v. Skipping project.", p.Name, err)
			projectFailed = true
		} else {
			log.Printf("  Successfully created backup: %s", backupFilePath)

			// --- Rsync Transfer ---
			if cfg.Rsync.Enabled {
				if cfg.Rsync.Destination == "" {
					log.Printf("  Warning: Rsync is enabled but no destination is set for project %s. Skipping rsync.", p.Name)
				} else {
					log.Printf("  Rsync enabled. Transferring %s to %s...", backupFilePath, cfg.Rsync.Destination)
					if err := rsync.Transfer(backupFilePath, cfg.Rsync.Destination, cfg.Rsync.Options, cfg.Rsync.Command); err != nil {
						log.Printf("  ERROR: Rsync transfer failed for %s: %v", backupFilePath, err)
					} else {
						log.Printf("  Rsync transfer successful for %s.", backupFilePath)
					}
				}
			}
		}

		// 5. Optional: Restart Stack (Only if backup was successful and enabled)
		if !projectFailed && cfg.RestartAfterBackup {
			log.Printf("  Restart requested for %s.", p.Name)

			// 5a. Optional Pull
			if cfg.PullBeforeRestart {
				log.Printf("  Pulling latest images for %s...", p.Name)
				if err := docker.Pull(p.Path); err != nil {
					log.Printf("  Warning: Failed to pull images for %s: %v", p.Name, err)
				} else {
					log.Printf("  Image pull successful for %s.", p.Name)
				}
			}

			// 5b. Restart (Up -d)
			log.Printf("  Starting stack %s...", p.Name)
			if err := docker.UpDetached(p.Path); err != nil {
				log.Printf("  ERROR: Failed to start stack %s after backup: %v", p.Name, err)
				// Mark project as failed overall if restart fails?
				// Maybe add a separate counter for restart failures?
				// For now, just log it. The backup itself was successful.
			} else {
				log.Printf("  Stack %s started successfully.", p.Name)
			}
		} else if projectFailed && cfg.RestartAfterBackup {
			log.Printf("  Skipping restart for %s because backup failed.", p.Name)
		} else if !projectFailed && !cfg.RestartAfterBackup {
			log.Printf("  Restart not requested for %s.", p.Name)
		}

		if projectFailed {
			failedBackups++
			log.Printf("--- Finished project %s with ERRORS ---", p.Name)
		} else {
			successfulBackups++
			log.Printf("--- Finished project %s successfully ---", p.Name)
		}
		// Space for readability
		log.Println("")

	} // End project loop

	log.Println("=================================================")
	log.Printf("Backup process finished. Successful: %d, Failed: %d", successfulBackups, failedBackups)
	log.Println("=================================================")

	// Exit with non-zero status if any backups failed
	if failedBackups > 0 {
		os.Exit(1)
	}
}
