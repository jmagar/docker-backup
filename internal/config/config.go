package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config stores all configuration for the application.
// The values are read by viper from a config file, environment variables, or flags.
type Config struct {
	ComposeDir         string
	AppdataDir         string
	BackupDir          string
	RestartAfterBackup bool
	PullBeforeRestart  bool
	Exclude            []string
	Verbose            bool
	DryRun             bool

	// --- Logging Configuration ---
	LogFile               string
	LogRotationMaxSizeMB  int
	LogRotationMaxBackups int
	LogRotationMaxAgeDays int
	LogRotationCompress   bool

	Rsync struct {
		Enabled     bool
		Destination string
		Options     string
		Command     string
	}
}

// Intermediate structure for unmarshalling YAML, matching YAML keys
type yamlConfig struct {
	ComposeDir            string   `yaml:"compose_dir"`
	AppdataDir            string   `yaml:"appdata_dir"`
	BackupDir             string   `yaml:"backup_dir"`
	RestartAfterBackup    bool     `yaml:"restart_after_backup"`
	PullBeforeRestart     bool     `yaml:"pull_before_restart"`
	Exclude               []string `yaml:"exclude_patterns"` // Match YAML key
	Verbose               bool     `yaml:"verbose"`
	DryRun                bool     `yaml:"dry_run"`
	LogFile               string   `yaml:"log_file"`
	LogRotationMaxSizeMB  int      `yaml:"log_rotation_max_size_mb"`
	LogRotationMaxBackups int      `yaml:"log_rotation_max_backups"`
	LogRotationMaxAgeDays int      `yaml:"log_rotation_max_age_days"`
	LogRotationCompress   bool     `yaml:"log_rotation_compress"`
	Rsync                 struct {
		Enabled     bool   `yaml:"enabled"`
		Destination string `yaml:"destination"`
		Options     string `yaml:"options"`
		Command     string `yaml:"command"`
	} `yaml:"rsync"`
}

// LoadConfig reads configuration using standard libraries and godotenv.
// Precedence: Flags > Environment Variables > Config File > Defaults
func LoadConfig() (Config, error) {
	var cfg Config

	// --- 1. Defaults ---
	defaults := Config{
		ComposeDir:            "/home/server/compose",
		AppdataDir:            "/home/server/appdata",
		BackupDir:             "./docker_backups",
		RestartAfterBackup:    false,
		PullBeforeRestart:     false,
		Exclude:               []string{}, // Corrected YAML key is exclude_patterns
		Verbose:               false,
		DryRun:                false,
		LogFile:               "backup-tool.log",
		LogRotationMaxSizeMB:  100,
		LogRotationMaxBackups: 3,
		LogRotationMaxAgeDays: 28,
		LogRotationCompress:   false,
		Rsync: struct {
			Enabled     bool
			Destination string
			Options     string
			Command     string
		}{
			Enabled:     false,
			Destination: "",
			Options:     "--archive --partial --compress --delete",
			Command:     "rsync",
		},
	}
	cfg = defaults // Start with defaults

	// --- 2. Config File --- (Requires flag parsing first to find potential custom path)
	// Define flags here but parse later
	configFilePath := flag.String("config", "", "Path to configuration file (e.g., config.yaml)")

	// Define other flags, using default values from the 'defaults' struct
	composeDirFlag := flag.String("compose-dir", defaults.ComposeDir, "Directory containing docker compose project subfolders")
	appdataDirFlag := flag.String("appdata-dir", defaults.AppdataDir, "Base directory containing application data volumes")
	backupDirFlag := flag.String("backup-dir", defaults.BackupDir, "Directory to store backup zip files")
	restartFlag := flag.Bool("restart", defaults.RestartAfterBackup, "Restart stacks after successful backup")
	pullFlag := flag.Bool("pull", defaults.PullBeforeRestart, "Pull latest images before restarting stacks (only if --restart is true)")
	// Note: StringSlice isn't standard; handle exclude flag manually if needed, or rely on env/config file.
	verboseFlag := flag.Bool("verbose", defaults.Verbose, "Enable verbose logging (shorthand -v)")
	flag.BoolVar(verboseFlag, "v", defaults.Verbose, "Enable verbose logging (shorthand for --verbose)") // Shorthand
	dryRunFlag := flag.Bool("dry-run", defaults.DryRun, "Perform a dry run, showing actions without executing them")
	logFileFlag := flag.String("log-file", defaults.LogFile, "Path to log file")
	rsyncEnabledFlag := flag.Bool("rsync-enabled", defaults.Rsync.Enabled, "Enable rsync transfer")
	rsyncDestFlag := flag.String("rsync-dest", defaults.Rsync.Destination, "Rsync destination (e.g., user@host:/path/)")
	rsyncOptsFlag := flag.String("rsync-opts", defaults.Rsync.Options, "Additional options for the rsync command")
	rsyncCmdFlag := flag.String("rsync-cmd", defaults.Rsync.Command, "Path to the rsync command executable")

	flag.Parse()

	// Determine effective config file path
	cfgFile := *configFilePath
	if cfgFile == "" { // If flag not set, check env
		cfgFile = os.Getenv("DOCKER_BACKUP_CONFIG_FILE")
	}
	if cfgFile == "" { // If flag and env not set, use default name
		cfgFile = "config.yaml" // Default config file name
	}

	// Attempt to read config file
	yamlData, err := os.ReadFile(cfgFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Configuration file '%s' not found. Using defaults/env/flags.", cfgFile)
		} else {
			return cfg, fmt.Errorf("error reading config file '%s': %w", cfgFile, err)
		}
	} else {
		log.Printf("Using configuration file: %s", cfgFile)
		var yamlCfg yamlConfig
		err = yaml.Unmarshal(yamlData, &yamlCfg)
		if err != nil {
			return cfg, fmt.Errorf("error unmarshalling config file '%s': %w", cfgFile, err)
		}

		// Override defaults with YAML values where they exist
		if yamlCfg.ComposeDir != "" {
			cfg.ComposeDir = yamlCfg.ComposeDir
		}
		if yamlCfg.AppdataDir != "" {
			cfg.AppdataDir = yamlCfg.AppdataDir
		}
		if yamlCfg.BackupDir != "" {
			cfg.BackupDir = yamlCfg.BackupDir
		}
		if yamlCfg.RestartAfterBackup {
			cfg.RestartAfterBackup = yamlCfg.RestartAfterBackup
		}
		if yamlCfg.PullBeforeRestart {
			cfg.PullBeforeRestart = yamlCfg.PullBeforeRestart
		}
		if len(yamlCfg.Exclude) > 0 {
			cfg.Exclude = yamlCfg.Exclude
		}
		if yamlCfg.Verbose {
			cfg.Verbose = yamlCfg.Verbose
		}
		if yamlCfg.DryRun {
			cfg.DryRun = yamlCfg.DryRun
		}
		if yamlCfg.LogFile != "" {
			cfg.LogFile = yamlCfg.LogFile
		}
		if yamlCfg.LogRotationMaxSizeMB != 0 {
			cfg.LogRotationMaxSizeMB = yamlCfg.LogRotationMaxSizeMB
		}
		if yamlCfg.LogRotationMaxBackups != 0 {
			cfg.LogRotationMaxBackups = yamlCfg.LogRotationMaxBackups
		}
		if yamlCfg.LogRotationMaxAgeDays != 0 {
			cfg.LogRotationMaxAgeDays = yamlCfg.LogRotationMaxAgeDays
		}
		if yamlCfg.LogRotationCompress {
			cfg.LogRotationCompress = yamlCfg.LogRotationCompress
		}

		if yamlCfg.Rsync.Enabled {
			cfg.Rsync.Enabled = yamlCfg.Rsync.Enabled
		}
		if yamlCfg.Rsync.Destination != "" {
			cfg.Rsync.Destination = yamlCfg.Rsync.Destination
		}
		if yamlCfg.Rsync.Options != "" {
			cfg.Rsync.Options = yamlCfg.Rsync.Options
		}
		if yamlCfg.Rsync.Command != "" {
			cfg.Rsync.Command = yamlCfg.Rsync.Command
		}
	}

	// --- 3. Environment Variables --- (Load .env first)
	// Ignore "file not found" error for .env
	_ = godotenv.Load() // Loads .env file into environment variables

	// Override config/defaults with ENV vars
	if envVal := os.Getenv("DOCKER_BACKUP_COMPOSE_DIR"); envVal != "" {
		cfg.ComposeDir = envVal
	}
	if envVal := os.Getenv("DOCKER_BACKUP_APPDATA_DIR"); envVal != "" {
		cfg.AppdataDir = envVal
	}
	if envVal := os.Getenv("DOCKER_BACKUP_BACKUP_DIR"); envVal != "" {
		cfg.BackupDir = envVal
	}
	if envVal := os.Getenv("DOCKER_BACKUP_RESTART_AFTER_BACKUP"); envVal != "" {
		if b, err := strconv.ParseBool(envVal); err == nil {
			cfg.RestartAfterBackup = b
		}
	}
	if envVal := os.Getenv("DOCKER_BACKUP_PULL_BEFORE_RESTART"); envVal != "" {
		if b, err := strconv.ParseBool(envVal); err == nil {
			cfg.PullBeforeRestart = b
		}
	}
	if envVal := os.Getenv("DOCKER_BACKUP_VERBOSE"); envVal != "" {
		if b, err := strconv.ParseBool(envVal); err == nil {
			cfg.Verbose = b
		}
	}
	if envVal := os.Getenv("DOCKER_BACKUP_DRY_RUN"); envVal != "" {
		if b, err := strconv.ParseBool(envVal); err == nil {
			cfg.DryRun = b
		}
	}
	if envVal := os.Getenv("DOCKER_BACKUP_LOG_FILE"); envVal != "" {
		cfg.LogFile = envVal
	}
	// Add parsing for other log rotation env vars if needed (e.g., using strconv.Atoi)

	if envVal := os.Getenv("DOCKER_BACKUP_RSYNC_ENABLED"); envVal != "" {
		if b, err := strconv.ParseBool(envVal); err == nil {
			cfg.Rsync.Enabled = b
		}
	}
	if envVal := os.Getenv("DOCKER_BACKUP_RSYNC_DESTINATION"); envVal != "" {
		cfg.Rsync.Destination = envVal
	}
	if envVal := os.Getenv("DOCKER_BACKUP_RSYNC_OPTIONS"); envVal != "" {
		cfg.Rsync.Options = envVal
	}
	if envVal := os.Getenv("DOCKER_BACKUP_RSYNC_COMMAND"); envVal != "" {
		cfg.Rsync.Command = envVal
	}
	// Note: Handling exclude list via ENV is complex; recommend using config file.

	// --- 4. Flags --- (Override all previous values if flag was set)
	// Check if a flag was actually set on the command line
	flagSet := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { flagSet[f.Name] = true })

	if flagSet["compose-dir"] {
		cfg.ComposeDir = *composeDirFlag
	}
	if flagSet["appdata-dir"] {
		cfg.AppdataDir = *appdataDirFlag
	}
	if flagSet["backup-dir"] {
		cfg.BackupDir = *backupDirFlag
	}
	if flagSet["restart"] {
		cfg.RestartAfterBackup = *restartFlag
	}
	if flagSet["pull"] {
		cfg.PullBeforeRestart = *pullFlag
	}
	if flagSet["verbose"] || flagSet["v"] {
		cfg.Verbose = *verboseFlag
	}
	if flagSet["dry-run"] {
		cfg.DryRun = *dryRunFlag
	}
	if flagSet["log-file"] {
		cfg.LogFile = *logFileFlag
	}
	if flagSet["rsync-enabled"] {
		cfg.Rsync.Enabled = *rsyncEnabledFlag
	}
	if flagSet["rsync-dest"] {
		cfg.Rsync.Destination = *rsyncDestFlag
	}
	if flagSet["rsync-opts"] {
		cfg.Rsync.Options = *rsyncOptsFlag
	}
	if flagSet["rsync-cmd"] {
		cfg.Rsync.Command = *rsyncCmdFlag
	}
	// Handle exclude flag if implemented (would require custom parsing)

	return cfg, nil
}
