package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config stores all configuration for the application.
// The values are read by viper from a config file, environment variables, or flags.
type Config struct {
	ComposeDir         string   `mapstructure:"compose_dir"`
	AppdataDir         string   `mapstructure:"appdata_dir"`
	BackupDir          string   `mapstructure:"backup_dir"`
	RestartAfterBackup bool     `mapstructure:"restart_after_backup"`
	PullBeforeRestart  bool     `mapstructure:"pull_before_restart"`
	Exclude            []string `mapstructure:"exclude"`
	Verbose            bool     `mapstructure:"verbose"`

	Rsync struct {
		Enabled     bool   `mapstructure:"enabled"`
		Destination string `mapstructure:"destination"`
		Options     string `mapstructure:"options"`
		Command     string `mapstructure:"command"`
	} `mapstructure:"rsync"`
}

// LoadConfig reads configuration from file, env vars, and flags using Viper
func LoadConfig() (cfg Config, err error) {
	// --- Defaults ---
	vp := viper.New()
	vp.SetDefault("compose_dir", "/home/server/compose")
	vp.SetDefault("appdata_dir", "/home/server/appdata")
	vp.SetDefault("backup_dir", "./docker_backups")
	vp.SetDefault("restart_after_backup", false)
	vp.SetDefault("pull_before_restart", false)
	vp.SetDefault("exclude", []string{})
	vp.SetDefault("verbose", false)
	vp.SetDefault("rsync.enabled", false)
	vp.SetDefault("rsync.destination", "")
	vp.SetDefault("rsync.options", "--archive --partial --compress --delete")
	vp.SetDefault("rsync.command", "rsync")

	// --- Flags ---
	pflag.String("compose-dir", vp.GetString("compose_dir"), "Directory containing docker compose project subfolders")
	pflag.String("appdata-dir", vp.GetString("appdata_dir"), "Base directory containing application data volumes")
	pflag.String("backup-dir", vp.GetString("backup_dir"), "Directory to store backup zip files")
	pflag.Bool("restart", vp.GetBool("restart_after_backup"), "Restart stacks after successful backup")
	pflag.Bool("pull", vp.GetBool("pull_before_restart"), "Pull latest images before restarting stacks (only if --restart is true)")
	pflag.StringSlice("exclude", vp.GetStringSlice("exclude"), "Glob patterns to exclude from backup (can be specified multiple times)")
	pflag.BoolP("verbose", "v", vp.GetBool("verbose"), "Enable verbose logging")
	pflag.Bool("rsync-enabled", vp.GetBool("rsync.enabled"), "Enable rsync transfer of backup files")
	pflag.String("rsync-dest", vp.GetString("rsync.destination"), "Rsync destination (e.g., user@host:/path/)")
	pflag.String("rsync-opts", vp.GetString("rsync.options"), "Additional options for the rsync command")
	pflag.String("rsync-cmd", vp.GetString("rsync.command"), "Path to the rsync command executable")
	configFile := pflag.String("config", "", "Path to configuration file (optional)")
	pflag.Parse()
	// Bind flags to viper
	err = vp.BindPFlags(pflag.CommandLine)
	if err != nil {
		return cfg, err
	}

	// --- Environment Variables ---
	// Explicitly bind problematic env vars BEFORE AutomaticEnv
	vp.BindEnv("restart_after_backup", "DOCKER_BACKUP_RESTART_AFTER_BACKUP")
	vp.BindEnv("pull_before_restart", "DOCKER_BACKUP_PULL_BEFORE_RESTART")
	vp.BindEnv("exclude", "DOCKER_BACKUP_EXCLUDE") // Still might not parse slice correctly, but let's bind it.
	vp.BindEnv("verbose", "DOCKER_BACKUP_VERBOSE") // Bind this too for consistency

	// Keep AutomaticEnv for others, but explicitly bound ones should take precedence if found
	vp.SetEnvPrefix("DOCKER_BACKUP")
	vp.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	vp.AutomaticEnv()
	// log.Printf("[DEBUG Config] Viper settings after AutomaticEnv:\n%+v\n", vp.AllSettings())
	// log.Printf("[DEBUG Config] DOCKER_BACKUP_RESTART_AFTER_BACKUP from env: %s (Viper GetString: %s)", os.Getenv("DOCKER_BACKUP_RESTART_AFTER_BACKUP"), vp.GetString("restart_after_backup"))
	// log.Printf("[DEBUG Config] DOCKER_BACKUP_PULL_BEFORE_RESTART from env: %s (Viper GetString: %s)", os.Getenv("DOCKER_BACKUP_PULL_BEFORE_RESTART"), vp.GetString("pull_before_restart"))
	// log.Printf("[DEBUG Config] DOCKER_BACKUP_EXCLUDE from env: %s (Viper GetString: %s)", os.Getenv("DOCKER_BACKUP_EXCLUDE"), vp.GetString("exclude")) // Commented out due to linter/apply issues

	// --- Config File ---
	cfgFile := *configFile
	if cfgFile != "" {
		vp.SetConfigFile(cfgFile)
	} else {
		vp.AddConfigPath(".")
		vp.AddConfigPath("$HOME/.config")
		vp.SetConfigName("config")
		vp.SetConfigType("yaml")
	}

	if err := vp.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if cfgFile != "" {
				return cfg, err
			}
			log.Println("No configuration file found. Using defaults/flags/env vars.")
		} else {
			return cfg, fmt.Errorf("error reading config file '%s': %w", vp.ConfigFileUsed(), err)
		}
	} else {
		log.Printf("Using configuration file: %s\n", vp.ConfigFileUsed())
	}
	// log.Printf("[DEBUG Config] Viper settings after ReadInConfig:\n%+v\n", vp.AllSettings())

	// --- Unmarshal to Struct ---
	err = vp.Unmarshal(&cfg)
	// if err != nil {
	// 	log.Printf("[DEBUG Config] Unmarshal error: %v", err)
	// }
	// log.Printf("[DEBUG Config] Final Config Struct:\n%+v\n", cfg)
	// log.Printf("[DEBUG Config] Viper settings after Unmarshal:\n%+v\n", vp.AllSettings())
	return cfg, err // Return original error from Unmarshal if any
}
