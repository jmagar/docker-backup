# System Patterns

**Architecture:**
- Command-Line Application written in Go (`backup-tool`).
- Modular structure with internal packages (`config`, `discovery`, `docker`, `backup`, `rsync`, `util`, `logutil`).
- Main application logic resides in `cmd/backup-tool/main.go`.
- Relies on executing external commands (`docker compose`/`docker-compose`, `rsync`) via `os/exec`.

**Key Technical Decisions / Patterns:**
- **Go:** Chosen for static typing, performance, standard library capabilities, and single binary distribution.
- **Viper:** Used for flexible configuration management (flags > env vars > config file). Resolved issues with boolean/slice loading by aligning struct field names with keys.
- **Volume Parsing:** Uses `docker compose config` output (parsed via `gopkg.in/yaml.v3`) to reliably determine resolved host paths, handling variable substitution and complex volume syntax.
- **Exclusion Handling:** Uses `filepath.WalkDir` in both copy (`copyDirectoryContents`) and zip (`zipDirectory`) phases. Compares relative paths (normalized to `/`) against glob patterns using `util.MatchesExclude` (which wraps `filepath.Match` with specific handling for `dir/**` and `**/*` patterns). Returns `filepath.SkipDir` for excluded directories.
- **Zip Archiving:** Uses standard `archive/zip` library.
- **Error Handling:** Utilizes Go's standard `error` interface. Uses custom `logutil` for logging errors and warnings. Aims to be non-fatal for single project errors where appropriate. Returns errors from `WalkDir` callbacks to halt processing on critical failures.
- **Logging (Climb lG8r):**
    - Centralized utility package `internal/logutil`.
    - Uses `github.com/fatih/color` for console colors (with TTY detection).
    - Uses `gopkg.in/natefinch/lumberjack.v2` for rotating file logs.
    - Separate loggers for file and console to allow different formatting (custom timestamp for file, standard for console).
    - `logutil.Init` called early in `main.go`, configured via Viper (but `config.LoadConfig` itself uses standard `log` to avoid import cycle).
- **Concurrency:** Processes projects sequentially.
- **External Commands:** Shells out to `docker compose`/`docker-compose` (command path determined in `main.go` and passed down) and `rsync` rather than using native Go libraries.
- **Shlex:** Uses `github.com/google/shlex` to safely parse user-provided rsync options string.
- **Deferred Cleanup:** Uses `defer` statements for reliable cleanup of temporary directories and log files. 