package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/totoual/groot/internal/app"
)

func main() {
	root, err := ResolveGrootHome()
	if err != nil {
		log.Fatalf(err.Error())
	}
	config := app.NewApp(root)

	fmt.Println(config)
}

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
