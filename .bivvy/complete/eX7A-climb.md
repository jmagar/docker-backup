<Climb>
  <header>
    <id>eX7A</id>
    <type>bug</type>
    <description>The `backup-tool` fails to correctly exclude files and directories specified in the `exclude` configuration list (e.g., `.git/**` fails to exclude the `.git` directory), leading to errors (like permission denied) or unnecessarily large backups.</description>
  </header>
  <problemBeingSolved>Users cannot reliably exclude unwanted files/directories (like `.git`, `node_modules`, logs, caches) from backups using glob patterns defined in the configuration. This results in failed backups due to permissions errors on excluded items or bloated backup archives.</problemBeingSolved>
  <successMetrics>
    - Running `./backup-tool` with `exclude: [".git/**", "cache/**"]` in the configuration successfully completes the backup for Project F (which contains a `.git` directory and a `cache` directory within its appdata).
    - Inspecting the temporary directory or the resulting zip archive for Project F confirms that neither the `.git` directory nor the `test-env/appdata/project-f/cache` directory (or their contents) were copied or included in the archive.
  </successMetrics>
  <requirements>
    - The file copying logic within `internal/backup/backup.go` (likely in `copyAndZip` or a related helper function using `filepath.WalkDir`) must correctly evaluate each file/directory path against the list of provided glob patterns (`cfg.Exclude`).
    - If a path matches *any* of the exclude patterns, it and its contents (if it's a directory) should be skipped entirely and not copied to the temporary directory or included in the zip archive.
    - The glob pattern matching should adhere to the standard Go `filepath.Match` behavior.
  </requirements>
  <newDependencies>None expected. May involve standard library packages like `path/filepath` if not already used correctly.</newDependencies>
  <prerequisiteChanges>Climb `bU6G` (fixing config loading) must be complete.</prerequisiteChanges>
  <relevantFiles>
    - `internal/backup/backup.go`: Contains the core backup creation logic (`CreateBackup`), including file discovery (`filepath.WalkDir` likely), filtering, copying, and zipping. The bug most likely resides here.
    - `config.yaml` / `.env`: Source of the `exclude` patterns list.
    - `test-env/compose/project-f/`: Test case with a `.git` directory.
    - `test-env/appdata/project-f/`: Contains a `cache/` directory to test excludes.
  </relevantFiles>
  <testingApproach>
    1. Configure `exclude: [".git/**", "cache/**"]` (using `.env` or `config.yaml`).
    2. Run `./backup-tool`.
    3. Verify Project F completes successfully without permission errors.
    4. (Optional but recommended) Add debug logging within the `filepath.WalkDir` callback in `internal/backup/backup.go` to print which files/dirs are being processed and which are being skipped due to exclusion match.
    5. (Optional but recommended) Manually inspect the `test-env/backups/project-f_*.zip` file to confirm the `.git` and `appdata/project-f/cache` directories are absent.
  </testingApproach>
  <constraints>Must continue using Go's standard `filepath.Match` for glob pattern matching.</constraints>
</Climb> 