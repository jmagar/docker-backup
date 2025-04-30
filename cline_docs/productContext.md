# Product Context

**Purpose:**
This project provides a command-line utility written in Go (`backup-tool`) to automate the backup of Docker Compose projects and their associated application data volumes.

**Problem Solved:**
Manually stopping Docker Compose stacks, identifying relevant data volumes, backing up project files and volume data into a structured archive, optionally restarting stacks, and transferring backups is complex, tedious, and error-prone. This utility streamlines and automates the process, improving reliability.

**How it Works:**
1.  **Configuration:** Reads settings from flags, environment variables (prefixed `DOCKER_BACKUP_`), and optionally a `config.yaml` file (flags > env > file precedence). Configuration loading is handled by Viper.
2.  **Discovery:** Scans the configured `compose_dir` for subdirectories and identifies the first `*.yaml` or `*.yml` compose file within each.
3.  **Dependency Check:** Verifies required external commands (`docker compose`/`docker-compose`, `rsync` if enabled) are available.
4.  **Project Loop:** For each discovered project:
    a.  **Stop Stack:** Executes `docker compose down`.
    b.  **Verify Down:** Executes `docker compose ps -q` to ensure the stack is stopped.
    c.  **Parse Volumes:** Executes `docker compose config` to get the fully resolved configuration. Parses this output to identify host volume paths (`type: bind`) that fall under the configured `appdata_dir`.
    d.  **Create Backup:** Creates a temporary directory. Copies the compose project files to `temp/compose/<project>/` and the identified appdata volume contents to `temp/appdata/<volume_base>/`, respecting `exclude` glob patterns during the copy. Creates a timestamped zip archive (`<project>_YYYYMMDD.zip`) in the `backup_dir`, also respecting excludes.
    e.  **Rsync (Optional):** If enabled, transfers the created zip file to the configured remote destination using `rsync`.
    f.  **Restart (Optional):** If enabled (`restart_after_backup: true`) and backup was successful, optionally pulls latest images (`pull_before_restart: true` -> `docker compose pull`) and restarts the stack (`docker compose up -d`).
5.  **Logging:** Provides informative logs via standard `log` package, with increased detail via a `--verbose` flag.
6.  **Logging (Enhanced - Climb lG8r):** Provides enhanced logging via a dedicated `logutil` package:
    *   Colorized console output (INFO, WARN, ERROR, SUCCESS, DEBUG).
    *   Simultaneous logging to a file (configurable path via `--log-file`, env var, or config).
    *   Rotating log file support (configurable size, age, count).
    *   Verbose mode (`--verbose` / `-v`) enables DEBUG level messages.
7.  **Cleanup:** Ensures temporary directories are removed.
8.  **Exit Status:** Exits with status 0 on success, 1 if any project backups failed.

Configuration options (`COMPOSE_DIR`, `BACKUP_DIR`) can be set via environment variables or default values within the script. 