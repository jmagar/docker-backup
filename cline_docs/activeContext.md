# Active Context

**Current Task:**
Completed **Climb lG8r: Enhance logging with terminal colors and rotating file output.** This involved implementing a new `internal/logutil` package and refactoring existing logging calls throughout the codebase.

**Recent Changes:**
- **Climb lG8r:**
    - Added dependencies: `github.com/fatih/color` (for terminal colors), `gopkg.in/natefinch/lumberjack.v2` (for log rotation).
    - Created `internal/logutil` package with separate loggers for console and file output.
    - Console output now uses colors for different log levels (INFO, WARN, ERROR, SUCCESS, DEBUG).
    - File output (`--log-file` flag, `DOCKER_BACKUP_LOG_FILE` env var, `log_file` config) logs simultaneously with console, using a 12-hour timestamp format `[YYYY/MM/DD HH:MM:SS AM/PM]`.
    - Implemented configurable log rotation (`log_rotation_*` settings in `config.yaml`).
    - Refactored logging calls in `main.go`, `config.go`, `discovery.go`, `docker.go`, `backup.go`, `rsync.go`.
    - Fixed a panic caused by `logutil` being called during config load before initialization by reverting specific log calls in `config.LoadConfig` to use the standard `log` package.
    - Resolved issue with unused `dockerComposeCmd` variable by passing it to relevant `internal/docker` functions.
    - Cleaned up temporary debug logs.
    - Updated `README.md` with new logging features and configuration.
- **Previous Climbs (t5E1, bU6G, aP9C, hY1D):** Details on test environment setup, config loading fixes, volume parsing fixes, and exclusion logic fixes remain relevant.
- Binary name is `backup-tool`.

**Next Steps:**
- **Testing:** Thoroughly test the new logging functionality (colors, file output, rotation, TTY detection, precedence) in various scenarios.
- **Code Refinement:** 
    - Consider adding a specific `logutil.ProjectHeader` function for better log structure.
    - Review overall error handling consistency.
- **Documentation:** Ensure `README.md` accurately reflects all current features and configuration after testing.
- **Future Enhancements (Deferred):** Consider future Climbs for features like Direct Docker API interaction, Restore functionality, or advanced Incremental backup strategies. 