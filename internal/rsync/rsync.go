package rsync

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"docker-backup-tool/internal/config"
	"docker-backup-tool/internal/logutil"
	// Use shlex for potentially safer splitting of options string
	"github.com/google/shlex"
)

// Transfer executes the rsync command to transfer a file.
// It now accepts the config.Rsync struct for clarity.
func TransferBackup(cfg config.Config, sourceFile string) error {
	// Split the options string into arguments respecting quotes
	// This allows options like -e "ssh -p 2222" to be parsed correctly.
	optsArgs, err := shlex.Split(cfg.Rsync.Options)
	if err != nil {
		return fmt.Errorf("failed to parse rsync options string '%s': %w", cfg.Rsync.Options, err)
	}

	// Construct the full rsync command arguments
	args := append(optsArgs, sourceFile, cfg.Rsync.Destination)

	// Use the provided rsync command path
	cmd := exec.Command(cfg.Rsync.Command, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	logutil.Debug("Running rsync: %s %s", cfg.Rsync.Command, strings.Join(args, " "))

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("rsync command failed: %w\nStderr: %s", err, stderr.String())
	}

	return nil
}
