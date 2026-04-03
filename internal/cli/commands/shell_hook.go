package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/totoual/groot/internal/app"
)

type ShellHookCmd struct{}

const (
	shellHookStartMarker = "# >>> groot shell hook >>>"
	shellHookLine        = `eval "$(groot shell-hook)"`
	shellHookEndMarker   = "# <<< groot shell hook <<<"
)

func (c *ShellHookCmd) Name() string { return "shell-hook" }

func (c *ShellHookCmd) Help() string {
	return "Print shell exports for the current Groot workspace context or install the shell hook"
}

func (c *ShellHookCmd) Run(a *app.App, args []string) error {
	if len(args) > 0 && args[0] == "install" {
		return c.runInstall(args[1:])
	}

	fs := flag.NewFlagSet("shell-hook", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot shell-hook")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if fs.NArg() != 0 {
		fs.Usage()
		return fmt.Errorf("shell-hook does not accept arguments")
	}

	output, err := a.ShellHook()
	if err != nil {
		return fmt.Errorf("couldn't build shell hook: %w", err)
	}

	fmt.Fprint(os.Stdout, output)
	return nil
}

func (c *ShellHookCmd) runInstall(args []string) error {
	fs := flag.NewFlagSet("shell-hook install", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	var shellName string
	var rcPath string
	fs.StringVar(&shellName, "shell", "", "Shell to install for (zsh or bash)")
	fs.StringVar(&rcPath, "rcfile", "", "Override rc file path")

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot shell-hook install [--shell zsh|bash] [--rcfile <path>]")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), "Install the Groot shell hook into a shell rc file.")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return fmt.Errorf("shell-hook install does not accept positional arguments")
	}

	target, err := resolveShellHookInstallTarget(shellName, rcPath)
	if err != nil {
		return err
	}

	changed, err := installShellHook(target)
	if err != nil {
		return err
	}

	if changed {
		fmt.Fprintf(os.Stdout, "Installed Groot shell hook in %s\n", target)
		return nil
	}

	fmt.Fprintf(os.Stdout, "Groot shell hook already installed in %s\n", target)
	return nil
}

func resolveShellHookInstallTarget(shellName, rcPath string) (string, error) {
	if strings.TrimSpace(rcPath) != "" {
		if strings.HasPrefix(rcPath, "~") {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("resolve home directory: %w", err)
			}
			rcPath = filepath.Join(home, strings.TrimPrefix(rcPath, "~"))
		}
		absPath, err := filepath.Abs(filepath.Clean(rcPath))
		if err != nil {
			return "", fmt.Errorf("resolve rc file path %q: %w", rcPath, err)
		}
		return absPath, nil
	}

	shellName = strings.TrimSpace(shellName)
	if shellName == "" {
		shellName = filepath.Base(os.Getenv("SHELL"))
	}
	switch shellName {
	case "zsh":
		return filepath.Join(userHomeDirOrDot(), ".zshrc"), nil
	case "bash":
		return filepath.Join(userHomeDirOrDot(), ".bashrc"), nil
	default:
		return "", fmt.Errorf("unsupported shell %q (use --shell zsh|bash or --rcfile)", shellName)
	}
}

func userHomeDirOrDot() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	return home
}

func installShellHook(target string) (bool, error) {
	existing := []byte{}
	data, err := os.ReadFile(target)
	if err == nil {
		existing = data
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("read rc file %q: %w", target, err)
	}

	content := string(existing)
	if strings.Contains(content, shellHookStartMarker) || strings.Contains(content, shellHookLine) {
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return false, fmt.Errorf("create rc file dir %q: %w", filepath.Dir(target), err)
	}

	var b strings.Builder
	if len(content) > 0 {
		b.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	fmt.Fprintln(&b, shellHookStartMarker)
	fmt.Fprintln(&b, shellHookLine)
	fmt.Fprintln(&b, shellHookEndMarker)

	if err := os.WriteFile(target, []byte(b.String()), 0o600); err != nil {
		return false, fmt.Errorf("write rc file %q: %w", target, err)
	}

	return true, nil
}
