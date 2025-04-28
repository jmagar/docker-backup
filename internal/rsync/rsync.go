package rsync

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"

	// Use shlex for potentially safer splitting of options string
	"github.com/google/shlex"
)

// Transfer executes the rsync command to transfer a file.
func Transfer(sourceFile, destination, options, rsyncCmd string) error {
	// Split the options string into arguments respecting quotes
	// This allows options like -e "ssh -p 2222" to be parsed correctly.
	optsArgs, err := shlex.Split(options)
	if err != nil {
		return fmt.Errorf("failed to parse rsync options string '%s': %w", options, err)
	}

	// Construct the full rsync command arguments
	args := append(optsArgs, sourceFile, destination)

	// Use the provided rsync command path
	cmd := exec.Command(rsyncCmd, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	log.Printf("    Running rsync: rsync %s\n", strings.Join(args, " "))

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("rsync command failed: %w\nStderr: %s", err, stderr.String())
	}

	log.Printf("    Rsync transfer successful for %s.\n", sourceFile)
	return nil
}
