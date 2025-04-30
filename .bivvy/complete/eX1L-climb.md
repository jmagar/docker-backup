# PRD: Log Excluded Files (Climb eX1L)

**Feature:** Log Excluded and Included Files/Folders during Backup

**Goal:** Provide users with visibility into which specific files and directories are being skipped *and included* during the backup process, especially when verbose mode is enabled.

**Requirements:**

1.  **Activation:** Logging of excluded *and included* items MUST occur only when verbose mode is enabled (`--verbose` flag or `DOCKER_BACKUP_VERBOSE=true`).
2.  **Logging Location:** 
    * Excluded item logging SHOULD happen within the file copying (`copyDirectoryContents`) and zipping (`zipDirectory`) logic in `internal/backup/backup.go`, immediately after an item is identified as excluded.
    * Included item logging SHOULD happen within the zipping (`zipDirectory`) logic in `internal/backup/backup.go` just before or after a file is successfully added to the archive.
3.  **Log Format:** 
    * Excluded items: `[DEBUG] Excluding (copy/zip): <relativePath> (matches exclude pattern)`
    * Included items: `[DEBUG] Adding to zip: <relativePath>`
4.  **Verbosity Level:** Both exclusion and inclusion logs SHOULD use the `DEBUG` log level.
5.  **Clarity:** Ensure the log message makes it clear whether an exclusion happened during the file copying phase or the zipping phase. Inclusion logging applies only during zipping.
6.  **No Functional Change:** This change MUST NOT alter the actual exclusion or backup logic; it only adds logging.
7.  **Included File Logging:** When verbose mode is active, the tool MUST log the relative path of each file as it is successfully added to the zip archive.

**Non-Goals:**

*   Changing the exclusion patterns or matching logic itself.
*   Logging excluded/included items when verbose mode is off.
*   Adding configuration options to control this specific logging. 