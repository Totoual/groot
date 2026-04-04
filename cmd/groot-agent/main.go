package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/totoual/groot/internal/agent"
)

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(2)
	}

	projectPath := os.Args[2]
	svc := agent.NewService(agent.NewCLIContract(os.Getenv("GROOT_BINARY")))

	var (
		status agent.RuntimeStatus
		err    error
	)

	switch os.Args[1] {
	case "status":
		status, err = svc.InspectStatus(projectPath)
	case "setup":
		status, err = svc.OpenAndSetup(projectPath)
	default:
		printUsage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "groot-agent failed: %v\n", err)
		os.Exit(1)
	}

	output, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "groot-agent failed to marshal output: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, string(output))
}

func printUsage() {
	fmt.Fprintln(os.Stdout, "usage: groot-agent <status|setup> <path>")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Thin Groot agent entrypoint over the CLI + JSON contract.")
}
