package helpers

import (
	"os"
	"path/filepath"
	"strings"
)

func ResolveGrootHome() (string, error) {
	var root string

	if env := os.Getenv("GROOT_HOME"); env != "" {
		root = env
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, ".groot")
	}

	// Expand ~ manually if user passed it
	if strings.HasPrefix(root, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, root[1:])
	}

	// Convert to absolute path
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	return filepath.Clean(abs), nil
}
