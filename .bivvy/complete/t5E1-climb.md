<Climb>
  <header>
    <id>t5E1</id>
    <type>task</type>
    <description>Set up a test environment under `test-env/` to validate the `docker-backup` Go application against various scenarios.</description>
  </header>
  <newDependencies>
    - None (relies on existing Docker setup).
  </newDependencies>
  <prerequisitChanges>
    - The `test-env/` directory should exist (user confirmed this).
    - The `docker-backup` binary should be built.
  </prerequisitChanges>
  <relevantFiles>
    - All files created under `test-env/compose/`
    - All files created under `test-env/appdata/`
  </relevantFiles>
  <everythingElse>
    **1. Purpose:**
      - To create a controlled environment for testing the `docker-backup` Go utility.
      - To cover various configurations and edge cases identified as important for validation.

    **2. Structure:**
      - Compose project definitions will reside in subdirectories under `test-env/compose/`.
      - Corresponding application data will reside under `test-env/appdata/`.

    **3. Test Cases to Implement:**
      - **Case A (Basic):** `test-env/compose/project-a/compose.yml` and `.env` file. Single service mounting `test-env/appdata/project-a`.
      - **Case B (Multi-service/Multi-appdata):** `test-env/compose/project-b/docker-compose.yaml` and `.env` file. Multiple services mounting separate appdata dirs (`test-env/appdata/project-b-svc1`, `test-env/appdata/project-b-svc2`).
      - **Case C (Custom Compose Name):** `test-env/compose/project-c/custom-name.yml` and `.env` file. Single service mounting `test-env/appdata/project-c`. Tests discovery of non-standard compose filename.
      - **Case D (No Compose File):** An empty directory `test-env/compose/project-d/`. The backup tool should skip this.
      - **Case E (Appdata Variations):** `test-env/compose/project-e/compose.yaml` and `.env` file. Service(s) with volumes pointing to:
          - An appdata path *outside* `test-env/appdata/` (e.g., `/tmp/project-e-outside`, should be ignored by backup).
          - An appdata path *within* `test-env/appdata/` that *does not actually exist* (e.g., `test-env/appdata/project-e-nonexistent`, should be warned about and skipped by backup).
          - A *valid* path within `test-env/appdata/project-e` (to ensure it still gets backed up alongside the variations).
      - **Case F (Excludes):** `test-env/compose/project-f/compose.yml` and `.env` file, with corresponding appdata `test-env/appdata/project-f`. Include files/dirs in both compose and appdata locations that should be excluded by patterns (e.g., `.git/config`, `cache/temp.dat`, `file.log`).
      - **Case G (No Relevant Appdata Volume):** `test-env/compose/project-g/compose.yml` and `.env` file. Service(s) mount volumes, but *none* from within `test-env/appdata/` (e.g., use named volumes or host paths outside the target appdata dir). Tests that backup proceeds but includes no `appdata/` content.
      - **Case H (Symlinks):** `test-env/compose/project-h/compose.yml` and `.env` file, mounting `test-env/appdata/project-h`. This appdata dir should contain a symbolic link. Tests symlink handling during copy/zip.
      - **Case I (Special Characters):** `test-env/compose/project-i/compose.yml` and `.env` file, mounting `test-env/appdata/project-i`. This appdata dir should contain files/subdirs with spaces or special characters in names. Tests path handling.
      - **Case J (Empty Appdata Directory):** `test-env/compose/project-j/compose.yml` and `.env` file, mounting `test-env/appdata/project-j`. This appdata dir should exist but be empty. Tests handling of empty source dirs.
      - **Case K (Invalid YAML):** `test-env/compose/project-k/broken.yaml` containing invalid YAML syntax. Tool should log a parsing error and skip this project.

    **4. Content Details:**
      - **Compose Files:** Use simple, functional services like `alpine:latest` with a `sleep infinity` command or `hello-world`. Define volumes according to the test case requirements, mapping to dummy container paths (e.g., `/data`). Include a basic `.env` file (e.g., `VAR=value`) in each project directory (A, B, C, E, F, G, H, I, J). For Case K, create an invalid YAML file.
      - **Appdata:** Create simple placeholder files (e.g., `touch test-env/appdata/project-a/data.txt`) to represent actual data. For Case F, create the specific files/dirs intended for exclusion testing. For Case H, create a file and a symlink pointing to it. For Case I, create files/dirs with names like `"my data file.txt"` or `"special(char)"`. For Case J, create an empty directory.

    **5. Execution:**
      - After setup, the `docker-backup` tool will be run against this environment using `test-env/compose` and `test-env/appdata` as the respective source directories. Configuration variations (excludes, restart, etc.) will be applied during testing runs.
  </everythingElse>
</Climb> 