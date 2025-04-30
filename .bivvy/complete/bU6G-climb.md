<Climb>
  <header>
    <id>bU6G</id>
    <type>bug</type>
    <description>The `docker-backup` binary fails to load all configuration settings (`exclude` list, `restart_after_backup`, `pull_before_restart`, `verbose`) from the `config.yaml` file when run. It appears to use default or empty values for some settings instead.</description>
  </header>
  <problemBeingSolved>Users cannot reliably configure the tool's behavior using `config.yaml` as intended. Settings like file exclusions and automatic stack restarts are ignored, leading to incorrect backup contents and behavior inconsistent with the configuration.</problemBeingSolved>
  <successMetrics>
    - Running `./docker-backup` with the current `config.yaml` shows the correct values for `exclude`, `restart_after_backup`, `pull_before_restart`, and `verbose` in the initial "Configuration loaded" log message.
    - Files/directories matching the `exclude` patterns in `config.yaml` (e.g., `.git/**`) are not present in the temporary backup directories or the final zip archives.
    - The tool attempts to pull images and restart stacks (`docker compose pull`, `docker compose up -d`) for successfully backed-up projects, as `restart_after_backup` and `pull_before_restart` are set to `true` in the config.
  </successMetrics>
  <requirements>
    - The application must correctly unmarshal *all* configuration fields defined in the Go `Config` struct (likely in `internal/config/config.go`) from the `config.yaml` file using the Viper library.
    - Viper's precedence order (flags > environment variables > config file) should still be respected.
  </requirements>
  <newDependencies>None expected.</newDependencies>
  <prerequisiteChanges>None expected.</prerequisiteChanges>
  <relevantFiles>
    - `cmd/backup-tool/main.go`: Entry point, likely where Viper is initialized and configuration is read/passed.
    - `internal/config/config.go`: Defines the `Config` struct and the `LoadConfig` function. This is the most likely location for the bug.
    - `config.yaml`: The configuration file being loaded.
    - `internal/backup/backup.go`: Uses the `ExcludePatterns` during the file copying/zipping process.
    - `internal/docker/docker.go`: Uses the `RestartStacks` and `PullImages` flags to determine post-backup actions.
  </relevantFiles>
  <testingApproach>
    1.  Run `./docker-backup` with the existing `config.yaml` (which has `verbose`, `restart_after_backup`, `pull_before_restart` set to `true`, and several `exclude` patterns).
    2.  Verify the initial log line "Configuration loaded:" accurately reflects *all* values from `config.yaml`.
    3.  Inspect the logs for successful backups (e.g., project-a) to confirm that "Pulling latest images..." and "Restarting stack..." messages appear.
    4.  Inspect the temporary directories created during the backup process (if possible, or add debug logging) or the contents of a successful backup zip file (e.g., `test-env/backups/project-f_*.zip` after fixing permissions/excludes) to ensure files matching `exclude` patterns (like `.git`) are absent.
  </testingApproach>
  <constraints>Must continue using Viper for configuration loading.</constraints>
</Climb> 