# PRD: Ensure Backup Directory Exists & Handle Permissions (Climb fK9B)

**Type:** Bug

**Goal:** Prevent backup failures caused by a non-existent target backup directory and provide guidance on handling permission errors during appdata copying.

**Problem Being Solved:**

1.  The tool currently attempts to create the zip file directly in the configured `backup_dir` without verifying if the directory exists, leading to `open ...: no such file or directory` errors.
2.  Copying files from certain `appdata` volumes fails due to `permission denied` errors, likely because the tool runs as a user without sufficient privileges for those specific files/directories.

**Requirements:**

1.  **Ensure Backup Directory:** Before attempting to create the zip file (`zipDirectory` function call or within it), the tool MUST check if the configured `backup_dir` exists.
2.  **Create Backup Directory:** If the `backup_dir` does not exist, the tool MUST attempt to create it recursively (including any necessary parent directories) using appropriate permissions (e.g., `os.MkdirAll(backupDir, 0755)`).
3.  **Error Handling (Directory Creation):** If creating the `backup_dir` fails, the tool MUST log a clear error and fail the backup process for that project gracefully.
4.  **Permission Error Guidance:** The `README.md` MUST be updated to include a note in the Usage or Troubleshooting section explaining that `permission denied` errors might occur when backing up appdata volumes and that running the tool with `sudo ./backup-tool ...` might be necessary to resolve them.

**Non-Goals:**

*   Automatically running the tool with `sudo`.
*   Changing file/directory ownership or permissions automatically.
*   Implementing complex permission checking beyond the directory creation. 