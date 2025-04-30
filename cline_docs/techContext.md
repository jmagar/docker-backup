# Tech Context

**Technologies Used:**
- **Go:** (v1.18+) Primary implementation language.
- **Go Modules:** For dependency management.
- **Libraries:**
    - `github.com/spf13/viper`: Configuration management.
    - `gopkg.in/yaml.v3`: YAML parsing (used for `docker compose config` output).
    - `github.com/google/shlex`: Shell-like string splitting (for rsync options).
    - `github.com/fatih/color`: Terminal color output (Climb lG8r).
    - `gopkg.in/natefinch/lumberjack.v2`: Log file rotation (Climb lG8r).
    - `github.com/mattn/go-isatty`: TTY detection for color output (Climb lG8r).
    - Go Standard Library: `os`, `os/exec`, `path/filepath`, `archive/zip`, `log`, `fmt`, `strings`, `io`, `io/fs`, `time`, `bytes`.
- **External Tools (Runtime Dependencies):**
    - Docker Engine.
    - Docker Compose (`docker compose` v2 CLI or `docker-compose` v1 CLI) - Used for stack operations and configuration resolution.
    - `rsync` (if rsync feature is enabled).

**Development Setup / Requirements:**
- Go v1.18+ toolchain installed.
- Docker Engine installed and running.
- Docker Compose (v1 or v2) installed.
- `rsync` installed (if testing rsync feature).
- Git (for version control).

**Operating Environment:**
- Primarily Linux (due to reliance on Docker, typical paths, `rsync`). Assumes POSIX-style paths internally after normalization.
- Requires permissions to:
    - Read compose directory and project subdirectories.
    - Read identified appdata directories/volumes.
    - Execute `docker` / `docker-compose` commands (user often needs to be in `docker` group).
    - Write to the backup directory.
    - Write to the configured log file path (defaults to CWD).
    - Write to a temporary directory (defaults to system temp).
    - Execute `rsync` (if enabled).
    - Connect to the remote `rsync` destination (if enabled, requires appropriate SSH keys/credentials configured outside this tool).

**Technical Constraints:**
- Relies on external `docker compose` and `rsync` commands found in PATH.
- Relies on the output format of `docker compose config` for volume parsing.
- Error handling aims to be robust but depends on correct interpretation of external command failures and file system errors.
- Exclude patterns use Go's `filepath.Match` which has specific syntax rules.
- Log file rotation depends on `lumberjack` behavior. 