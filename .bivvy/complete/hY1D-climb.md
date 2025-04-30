<Climb>
  <header>
    <id>hY1D</id>
    <type>bug</type>
    <description>The exclude pattern matching logic in `internal/backup/backup.go` is ineffective. Files and directories matching configured `exclude` patterns (e.g., `.git/**`, `cache/**`) are still being copied into the temporary backup directory and included in the final zip archive.</description>
  </header>
  <problemBeingSolved>Despite correctly loading exclude patterns (fixed in `bU6G`) and attempting to skip excluded paths during file copying (attempted fix in `eX7A`), the exclusion is not fully working. This leads to bloated backups and potential permission errors if excluded files cannot be read (though the previous fix might mask the permission error by skipping later).</problemBeingSolved>
  <successMetrics>
    - Running `./backup-tool` with `exclude: [".git/**", "cache/**", "**/*.log"]` completes successfully for Project F.
    - Inspecting the resulting `test-env/backups/project-f_*.zip` file confirms that the `.git/` directory (and its contents), the `appdata/project-f/cache/` directory (and its contents), and `appdata/project-f/service.log` are **absent** from the archive.
    - Other non-excluded files (like `compose/project-f/compose.yml`, `appdata/project-f/important.dat`) are still present.
  </successMetrics>
  <requirements>
    - The `copyDirectoryContents` function in `internal/backup/backup.go` must reliably prevent excluded files and directories from being copied to the temporary location.
    - The use of `filepath.SkipDir` must correctly stop `filepath.WalkDir` from descending into excluded directories.
    - The matching logic needs to correctly handle patterns like `**/*.log`, `cache/**`, and `.git/**` against the relative paths generated during the walk.
  </requirements>
  <newDependencies>None expected.</newDependencies>
  <prerequisiteChanges>Climbs `bU6G` (config loading) and `aP9C` (appdata inclusion) must be complete.</prerequisiteChanges>
  <relevantFiles>
    - `internal/backup/backup.go`: Specifically the `copyDirectoryContents` function and its `filepath.WalkDir` callback.
    - `internal/util/util.go`: The `MatchesExclude` helper function.
    - `config.yaml` / `.env`: Source of the `exclude` patterns.
    - `test-env/compose/project-f/`: Contains `.git`.
    - `test-env/appdata/project-f/`: Contains `cache/` and `service.log`.
  </relevantFiles>
  <testingApproach>
    1. Ensure `exclude: [".git/**", "cache/**", "**/*.log"]` is configured.
    2. Run `./backup-tool`.
    3. Verify Project F completes successfully.
    4. Unzip `test-env/backups/project-f_*.zip`.
    5. Check the contents:
        - `.git/` directory MUST NOT exist in `compose/project-f/`.
        - `cache/` directory MUST NOT exist in `appdata/project-f/`.
        - `service.log` file MUST NOT exist in `appdata/project-f/`.
        - Other files (`compose.yml`, `important.dat`) MUST exist.
  </testingApproach>
  <constraints>Must continue using Go's standard `filepath.Match`.</constraints>
</Climb> 