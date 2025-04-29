# Go Docker Backup Tool

A utility written in Go to back up Docker Compose projects, including specified application data volumes, with options for restarting stacks and transferring backups via rsync.

## Features

*   Discovers Docker Compose projects in a specified directory.
*   Finds the first `*.yaml` or `*.yml` file in each project directory.
*   Stops the associated Docker Compose stack (`docker compose down` or `docker-compose down`).
*   Parses the compose file to identify host volume paths located within a specified application data directory.
*   Creates a timestamped zip archive (`<project_name>_YYYYMMDD.zip`) containing:
    *   `compose/<project_name>/...` (Contents of the compose project directory)
    *   `appdata/<volume_base_name>/...` (Contents of each identified appdata volume)
*   Supports excluding files/directories using glob patterns.
*   Optionally pulls latest images and restarts the stack after a successful backup.
*   Optionally transfers the created zip archive to a remote destination using `rsync`.
*   Configuration via command-line flags, environment variables, and/or a `config.yaml` file.
*   Basic logging with verbose option.
*   **Enhanced Logging:**
    *   Colorized terminal output for improved readability (Info/Warn/Error/Success/Debug).
    *   Simultaneous logging to a file.
    *   Configurable log file path (`--log-file`, `DOCKER_BACKUP_LOG_FILE`, `log_file` in config).
    *   Automatic log rotation based on size, age, and number of backups (configurable in `config.yaml`).
    *   Verbose option (`-v`, `--verbose`, `DOCKER_BACKUP_VERBOSE`) enables debug messages.

## Installation

1.  **Prerequisites:**
    *   Go 1.18 or higher (for build)
    *   Docker (`docker` CLI)
    *   Docker Compose (`docker compose` v2 or `docker-compose` v1)
    *   `rsync` (if using the rsync feature)
2.  **Build:**
    ```bash
    go build -o backup-tool ./cmd/backup-tool
    ```
3.  Place the compiled `backup-tool` binary in your PATH or run it directly (`./backup-tool`).

## Configuration

Configuration is loaded in the following order of precedence:

1.  Command-line Flags
2.  Environment Variables (prefixed with `DOCKER_BACKUP_`)
3.  Configuration File (`config.yaml`)

### Configuration File (`config.yaml`)

Create a `config.yaml` file either in the current directory or `$HOME/.config/`. See `config.example.yaml` for all options.

Example `config.yaml`:

```yaml
compose_dir: /srv/docker/compose
appdata_dir: /srv/docker/appdata
backup_dir: /mnt/backups/docker
restart_after_backup: true
pull_before_restart: true
verbose: false
exclude:
  - ".git/*"
  - cache/*
  - "*.log"

rsync:
  enabled: true
  destination: "user@backup-server:/srv/docker-backups/"
  options: "--archive --compress -e 'ssh -i /home/user/.ssh/id_rsa'"
  command: "rsync"

# Logging Configuration
log_file: "/var/log/backup-tool.log"
log_rotation_max_size_mb: 100
log_rotation_max_backups: 5
log_rotation_max_age_days: 30
log_rotation_compress: true
```

### Environment Variables

Set environment variables prefixed with `DOCKER_BACKUP_`. Nested keys use underscores. Underscores in keys map to underscores in env vars (e.g., `restart_after_backup` -> `DOCKER_BACKUP_RESTART_AFTER_BACKUP`).

Example:

```bash
export DOCKER_BACKUP_COMPOSE_DIR="/srv/docker/compose"
export DOCKER_BACKUP_RESTART_AFTER_BACKUP=true
export DOCKER_BACKUP_RSYNC_ENABLED=true
export DOCKER_BACKUP_RSYNC_DESTINATION="user@host:/path"
export DOCKER_BACKUP_LOG_FILE="/logs/backup.log"
```

### Command-line Flags

Run `backup-tool --help` to see all available flags (output may vary slightly):

```text
Usage of backup-tool:
      --appdata-dir string     Base directory containing application data volumes (default "/home/server/appdata")
      --backup-dir string      Directory to store backup zip files (default "./docker_backups")
      --compose-dir string     Directory containing docker compose project subfolders (default "/home/server/compose")
      --config string          Path to configuration file (optional)
      --exclude stringSlice    Glob patterns to exclude from backup (can be specified multiple times)
      --log-file string        Path to log file (defaults to backup-tool.log in current dir)
      --pull                   Pull latest images before restarting stacks (only if --restart is true)
      --restart                Restart stacks after successful backup
      --rsync-cmd string       Path to the rsync command executable (default "rsync")
      --rsync-dest string      Rsync destination (e.g., user@host:/path/)
      --rsync-enabled          Enable rsync transfer of backup files
      --rsync-opts string      Additional options for the rsync command (default "--archive --partial --compress --delete")
  -v, --verbose                Enable verbose logging
```

## Usage

Run the compiled binary. Configure using flags, environment variables, or a config file.

```bash
# Basic usage with defaults (logs to ./backup-tool.log)
./backup-tool

# Using flags
./backup-tool --compose-dir /opt/stacks --backup-dir /backups --restart --rsync-enabled --rsync-dest myuser@remote:/backups/ --log-file /var/log/docker_backup.log

# Using a config file
./backup-tool --config /etc/docker-backup/config.yaml

# Using verbose logging (includes DEBUG messages)
./backup-tool -v

# Redirecting output (colors will be disabled in the file)
./backup-tool > output.txt
``` 