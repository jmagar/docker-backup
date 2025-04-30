# PRD: Dry Run Mode Enhancement (Climb qP5Z)

**Feature:** Dry Run Mode Enhancement

**Goal:** Allow users to simulate the backup process without making any changes to Docker stacks or the filesystem, providing an accurate preview of actions and configurations.

**Requirements:**

1.  **Activation:** Dry run mode MUST be activated via the `--dry-run` command-line flag or the `DOCKER_BACKUP_DRY_RUN=true` environment variable. Configuration loading precedence (flag > env > file > default) MUST be respected for this setting.
2.  **Simulation Only:** When active, the tool MUST NOT execute any actions that modify the system state or filesystem. It MUST only log the actions it *would* have taken. This includes:
    *   Stopping Docker Compose stacks (`docker compose down`).
    *   Verifying stack status (`docker compose ps`).
    *   Creating backup archives (copying files, zipping).
    *   Transferring backups via rsync.
    *   Pulling Docker images (`docker compose pull`).
    *   Starting Docker Compose stacks (`docker compose up -d`).
3.  **Read-Only Operations:** The tool MAY still perform read-only operations during a dry run, such as:
    *   Loading configuration (from all sources).
    *   Discovering projects.
    *   Parsing compose files (`docker compose config`) to determine volumes.
    *   Checking external command dependencies (`docker`, `docker-compose`, `rsync`).
    *   Logging all simulated actions and read-only steps performed.
4.  **Clear Logging:** Output logs MUST clearly distinguish simulated actions from read-only operations. A consistent prefix (e.g., `[DRY RUN]`) MUST be used for all simulated actions.
5.  **Logical Flow:** The dry run simulation MUST proceed logically through the backup steps for each project. It should assume simulated actions (like stopping the stack) were successful to allow subsequent simulation steps (like backup creation) to be previewed, unless a preceding read-only operation (like parsing the compose file) fails.
6.  **Simulation Detail:**
    *   Simulated backup creation MUST log the target project name, compose path, identified appdata paths, and the number of exclusion patterns that *would* be applied.
    *   Simulated rsync transfer MUST log the simulated source backup filename and the configured destination.
    *   Simulated stack verification MUST log that verification *would* occur, without actually running `docker compose ps`.

**Non-Goals:**

*   Changing the core backup logic outside of the dry run simulation.
*   Adding new configuration options beyond the dry run flag/env var. 