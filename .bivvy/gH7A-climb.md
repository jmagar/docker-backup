# PRD: Fix Exclusions & Add Sudo Check (Climb gH7A)

**Type:** Bug/Feature

**Goal:** Ensure exclusion patterns work correctly for nested directories and warn the user if the tool is not run with sufficient privileges (likely root/sudo) for accessing potentially restricted appdata files.

**Problem Being Solved:**

1.  The exclusion pattern `*/cache/*` failed to exclude contents within `appdata/tautulli/cache`, suggesting an issue with how relative paths or patterns are handled by `util.MatchesExclude`.
2.  Running the tool without root privileges frequently leads to `permission denied` errors when backing up appdata volumes owned by different users/groups, potentially resulting in incomplete backups without clear upfront warning.

**Requirements:**

1.  **Fix Exclusion Matching:**
    *   Investigate the logic within `internal/util/util.go` (`MatchesExclude`) and potentially how paths are passed to it from `internal/backup/backup.go` (`copyDirectoryContents`, `zipDirectory`).
    *   Modify the logic to ensure patterns like `*/cache/*` correctly match and exclude nested directories (e.g., `appdata/tautulli/cache`).
    *   Ensure other patterns (e.g., `*/.git/*`, `*.log`) continue to function correctly.
2.  **Implement Sudo Check:**
    *   At the beginning of the `main` function in `cmd/backup-tool/main.go` (after logger initialization), check if the effective user ID (EUID) is 0 (root).
    *   Use the `os.Geteuid()` function from the standard `os` package.
3.  **Implement User Warning:**
    *   If the EUID is *not* 0, log a prominent `WARN` level message stating that the tool is not running as root and may encounter permission errors backing up appdata, recommending `sudo` for complete backups.
    *   This warning should appear early in the program execution.
4.  **Documentation:** Update `README.md` to mention the non-root user warning.

**Non-Goals:**

*   Forcing the user to run as root.
*   Changing any file permissions or ownership.
*   Adding specific logic to handle different kinds of permission errors during backup (the warning is the primary mechanism). 