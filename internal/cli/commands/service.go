package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/totoual/groot/internal/app"
	"github.com/totoual/groot/internal/cli/cliutil"
	"github.com/totoual/groot/internal/cli/interfaces"
)

type ServiceCmd struct {
	subcmds map[string]interfaces.Cmd
}

func NewServiceCmd(cmds ...interfaces.Cmd) *ServiceCmd {
	if len(cmds) == 0 {
		cmds = defaultServiceCommands()
	}
	return &ServiceCmd{subcmds: serviceCommandMap(cmds...)}
}

func serviceCommandMap(cmds ...interfaces.Cmd) map[string]interfaces.Cmd {
	m := make(map[string]interfaces.Cmd, len(cmds))
	for _, c := range cmds {
		m[c.Name()] = c
	}
	return m
}

func defaultServiceCommands() []interfaces.Cmd {
	return []interfaces.Cmd{
		&serviceAddCmd{},
		&serviceRemoveCmd{},
		&serviceListDeclaredCmd{},
		&serviceStartCmd{},
		&serviceStatusCmd{},
		&serviceListCmd{},
		&serviceLogsCmd{},
		&serviceStopCmd{},
	}
}

func (c *ServiceCmd) commands() map[string]interfaces.Cmd {
	if c.subcmds == nil {
		c.subcmds = serviceCommandMap(defaultServiceCommands()...)
	}
	return c.subcmds
}

func (c *ServiceCmd) Name() string { return "service" }
func (c *ServiceCmd) Help() string {
	return "Run and inspect workspace-owned services for a workspace"
}

func (c *ServiceCmd) Run(a *app.App, args []string) error {
	if cliutil.IsHelpRequest(args) {
		c.PrintHelp(os.Stdout)
		return nil
	}
	if len(args) == 0 {
		c.PrintHelp(os.Stdout)
		return fmt.Errorf("service command required")
	}
	subcmd, ok := c.commands()[args[0]]
	if !ok {
		return fmt.Errorf("unknown service command %q (try: groot service -h)", args[0])
	}
	return subcmd.Run(a, args[1:])
}

func (c *ServiceCmd) PrintHelp(w io.Writer) {
	fmt.Fprintln(w, "usage: groot service <command> [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "commands:")
	var names []string
	for name := range c.commands() {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cmd := c.commands()[name]
		fmt.Fprintf(w, "  %-12s %s\n", cmd.Name(), cmd.Help())
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'groot service <command> -h' for more information on a command.")
}

type serviceStartCmd struct{}

func (c *serviceStartCmd) Name() string { return "start" }
func (c *serviceStartCmd) Help() string { return "Start a declared service for a workspace" }

type serviceAddCmd struct{}

func (c *serviceAddCmd) Name() string { return "add" }
func (c *serviceAddCmd) Help() string { return "Add or update a declared service for a workspace" }

func (c *serviceAddCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("service add", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	cwd := fs.String("cwd", "", "relative or absolute working directory for the service")
	restart := fs.String("restart", "", "restart policy recorded in the manifest")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot service add <workspace> <name> [--cwd dir] [--restart policy] -- <cmd> [args...]")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}

	workspaceName, serviceName, command, err := parseServiceDeclarationArgs(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	workspaceName, err = requireWorkspaceArg(a, workspaceName)
	if err != nil {
		return err
	}

	if err := a.DeclareService(workspaceName, app.ServiceSpec{
		Name:    serviceName,
		Command: append([]string{}, command...),
		Cwd:     strings.TrimSpace(*cwd),
		Restart: strings.TrimSpace(*restart),
	}); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Declared service %q in workspace %q.\n", serviceName, workspaceName)
	return nil
}

type serviceRemoveCmd struct{}

func (c *serviceRemoveCmd) Name() string { return "remove" }
func (c *serviceRemoveCmd) Help() string { return "Remove a declared service from a workspace" }

func (c *serviceRemoveCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("service remove", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot service remove <workspace> <name>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("workspace name and service name required")
	}

	workspaceName, err := requireWorkspaceArg(a, fs.Arg(0))
	if err != nil {
		return err
	}
	serviceName := strings.TrimSpace(fs.Arg(1))
	if err := a.DeleteService(workspaceName, serviceName); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Removed service %q from workspace %q.\n", serviceName, workspaceName)
	return nil
}

type serviceListDeclaredCmd struct{}

func (c *serviceListDeclaredCmd) Name() string { return "list-declared" }
func (c *serviceListDeclaredCmd) Help() string { return "List declared services for a workspace" }

func (c *serviceListDeclaredCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("service list-declared", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot service list-declared <workspace>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("workspace name required")
	}

	workspaceName, err := requireWorkspaceArg(a, fs.Arg(0))
	if err != nil {
		return err
	}
	services, err := a.DeclaredServices(workspaceName)
	if err != nil {
		return err
	}
	if len(services) == 0 {
		fmt.Fprintln(os.Stdout, "No declared services.")
		return nil
	}
	for _, service := range services {
		fmt.Fprintf(os.Stdout, "%s\t%s\n", service.Name, strings.Join(service.Command, " "))
	}
	return nil
}

func (c *serviceStartCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("service start", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot service start <workspace> <name>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("workspace name and service name required")
	}
	workspaceName, err := requireWorkspaceArg(a, fs.Arg(0))
	if err != nil {
		return err
	}
	if err := enforceWorkspaceOwnership(a, workspaceName); err != nil {
		return err
	}
	service, err := a.StartService(workspaceName, fs.Arg(1))
	if err != nil {
		return err
	}
	writeServiceStatus(service)
	return nil
}

type serviceStatusCmd struct{}

func (c *serviceStatusCmd) Name() string { return "status" }
func (c *serviceStatusCmd) Help() string {
	return "Print service status for a workspace and service name"
}

func (c *serviceStatusCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("service status", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot service status <workspace> <name>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("workspace name and service name required")
	}
	workspaceName, err := requireWorkspaceArg(a, fs.Arg(0))
	if err != nil {
		return err
	}
	service, err := a.ServiceStatus(workspaceName, fs.Arg(1))
	if err != nil {
		return err
	}
	writeServiceStatus(service)
	return nil
}

type serviceListCmd struct{}

func (c *serviceListCmd) Name() string { return "list" }
func (c *serviceListCmd) Help() string { return "List service runtime status for a workspace" }

func (c *serviceListCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("service list", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot service list <workspace>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("workspace name required")
	}
	workspaceName, err := requireWorkspaceArg(a, fs.Arg(0))
	if err != nil {
		return err
	}
	services, err := a.ServiceList(workspaceName)
	if err != nil {
		return err
	}
	if len(services) == 0 {
		fmt.Fprintln(os.Stdout, "No services.")
		return nil
	}
	for _, service := range services {
		fmt.Fprintf(os.Stdout, "%s\t%s\n", service.Name, service.State)
	}
	return nil
}

type serviceLogsCmd struct{}

func (c *serviceLogsCmd) Name() string { return "logs" }
func (c *serviceLogsCmd) Help() string { return "Print captured stdout and stderr for a service" }

func (c *serviceLogsCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("service logs", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot service logs <workspace> <name>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("workspace name and service name required")
	}
	workspaceName, err := requireWorkspaceArg(a, fs.Arg(0))
	if err != nil {
		return err
	}
	logs, err := a.ServiceLogs(workspaceName, fs.Arg(1))
	if err != nil {
		return err
	}
	if logs.Stdout != "" {
		fmt.Fprint(os.Stdout, logs.Stdout)
	}
	if logs.Stderr != "" {
		fmt.Fprint(os.Stderr, logs.Stderr)
	}
	return nil
}

type serviceStopCmd struct{}

func (c *serviceStopCmd) Name() string { return "stop" }
func (c *serviceStopCmd) Help() string { return "Stop a running service for a workspace" }

func (c *serviceStopCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("service stop", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot service stop <workspace> <name>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("workspace name and service name required")
	}
	workspaceName, err := requireWorkspaceArg(a, fs.Arg(0))
	if err != nil {
		return err
	}
	service, err := a.StopService(workspaceName, fs.Arg(1))
	if err != nil {
		return err
	}
	writeServiceStatus(service)
	return nil
}

func writeServiceStatus(service app.ServiceStatus) {
	fmt.Fprintf(os.Stdout, "Service: %s\n", service.Name)
	fmt.Fprintf(os.Stdout, "Workspace: %s\n", service.Workspace)
	fmt.Fprintf(os.Stdout, "State: %s\n", service.State)
	fmt.Fprintf(os.Stdout, "Command: %s\n", service.Command)
	if len(service.Args) > 0 {
		fmt.Fprintf(os.Stdout, "Args: %s\n", strings.Join(service.Args, " "))
	}
	fmt.Fprintf(os.Stdout, "Workdir: %s\n", service.Cwd)
	if service.RestartPolicy != "" {
		fmt.Fprintf(os.Stdout, "Restart Policy: %s\n", service.RestartPolicy)
	}
	if service.PID != 0 {
		fmt.Fprintf(os.Stdout, "PID: %d\n", service.PID)
	}
	if service.ExitCode != nil {
		fmt.Fprintf(os.Stdout, "Exit Code: %d\n", *service.ExitCode)
	}
	if service.StopReason != "" {
		fmt.Fprintf(os.Stdout, "Stop Reason: %s\n", service.StopReason)
	}
}
