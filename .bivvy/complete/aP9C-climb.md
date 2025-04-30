<Climb>
  <header>
    <id>aP9C</id>
    <type>bug</type>
    <description>Backup zip files created by `backup-tool` only contain the `compose/<project_name>/` directory and its contents. The expected `appdata/<volume_base_name>/` directories and their contents are missing from the archives.</description>
  </header>
  <problemBeingSolved>The primary purpose of the tool is to back up *both* the compose project files and the associated application data. Currently, the application data is not being included, making the backups incomplete and potentially useless for restore purposes.</problemBeingSolved>
  <successMetrics>
    - Running `./backup-tool` on projects with correctly defined host volume paths pointing within `test-env/appdata/` (e.g., Project A, B, E, F, G, H, I, J) results in zip files containing both `compose/<project_name>/...` and `appdata/<volume_base_name>/...` structures.
    - The contents within the `appdata/` directories in the zip file match the contents of the corresponding directories in `test-env/appdata/`.
  </successMetrics>
  <requirements>
    - The `backup.ParseVolumes` function must correctly identify host paths specified in docker-compose files that fall under the configured `appdata_dir`.
    - The `backup.CreateBackup` function must correctly copy the identified `appdataPaths` into the temporary backup directory under the `appdata/` subdirectory structure.
    - The `backup.zipDirectory` function must correctly include the `appdata/` subdirectory and its contents from the temporary directory into the final zip archive.
  </requirements>
  <newDependencies>None expected.</newDependencies>
  <prerequisiteChanges>Climbs `bU6G` (config loading) and `eX7A` (exclude patterns) should ideally be complete, as they might affect testing or related logic.</prerequisiteChanges>
  <relevantFiles>
    - `internal/backup/backup.go`: Contains `ParseVolumes`, `CreateBackup`, `copyPath`, `copyDirectoryContents`, and `zipDirectory`. The bug likely lies within the interaction of these functions, specifically how `appdataPaths` are handled during the copy and zip stages.
    - `cmd/backup-tool/main.go`: Calls `ParseVolumes` and `CreateBackup`.
    - `test-env/appdata/`: Contains the source application data.
    - `test-env/compose/`: Contains the compose files defining the volumes.
    - `test-env/backups/`: Contains the resulting (currently incomplete) zip files.
  </relevantFiles>
  <testingApproach>
    1. Run `./backup-tool` on the test environment.
    2. Select a project that *should* have appdata (e.g., `project-f`).
    3. Unzip the corresponding backup file (e.g., `test-env/backups/project-f_*.zip`).
    4. Verify if the `appdata/project-f/` directory exists within the zip and contains the expected files/directories (e.g., `data.txt`, excluding the `cache` dir due to exclude patterns).
    5. If missing, add debug logging in `internal/backup/backup.go`:
        - Log the `appdataPaths` returned by `ParseVolumes`.
        - Log the source and destination paths during the appdata copy loop in `CreateBackup`.
        - Log paths being added during `zipDirectory`.
    6. Rerun and analyze logs to pinpoint where the appdata path is lost or skipped.
  </testingApproach>
  <constraints>Must correctly handle exclude patterns when copying appdata.</constraints>
</Climb> 