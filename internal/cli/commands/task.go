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

type TaskCmd struct {
	subcmds map[string]interfaces.Cmd
}

func NewTaskCmd(cmds ...interfaces.Cmd) *TaskCmd {
	if len(cmds) == 0 {
		cmds = defaultTaskCommands()
	}
	return &TaskCmd{subcmds: taskCommandMap(cmds...)}
}

func taskCommandMap(cmds ...interfaces.Cmd) map[string]interfaces.Cmd {
	m := make(map[string]interfaces.Cmd, len(cmds))
	for _, c := range cmds {
		m[c.Name()] = c
	}
	return m
}

func defaultTaskCommands() []interfaces.Cmd {
	return []interfaces.Cmd{
		&taskAddCmd{},
		&taskRemoveCmd{},
		&taskListDeclaredCmd{},
		&taskStartCmd{},
		&taskStatusCmd{},
		&taskListCmd{},
		&taskLogsCmd{},
		&taskStopCmd{},
	}
}

func (c *TaskCmd) commands() map[string]interfaces.Cmd {
	if c.subcmds == nil {
		c.subcmds = taskCommandMap(defaultTaskCommands()...)
	}
	return c.subcmds
}

func (c *TaskCmd) Name() string { return "task" }

func (c *TaskCmd) Help() string {
	return "Run and inspect workspace-owned tasks for a workspace"
}

func (c *TaskCmd) Run(a *app.App, args []string) error {
	if cliutil.IsHelpRequest(args) {
		c.PrintHelp(os.Stdout)
		return nil
	}
	if len(args) == 0 {
		c.PrintHelp(os.Stdout)
		return fmt.Errorf("task command required")
	}

	subcmd, ok := c.commands()[args[0]]
	if !ok {
		return fmt.Errorf("unknown task command %q (try: groot task -h)", args[0])
	}
	return subcmd.Run(a, args[1:])
}

func (c *TaskCmd) PrintHelp(w io.Writer) {
	fmt.Fprintln(w, "usage: groot task <command> [args]")
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
	fmt.Fprintln(w, "Run 'groot task <command> -h' for more information on a command.")
}

type taskStartCmd struct{}

func (c *taskStartCmd) Name() string { return "start" }
func (c *taskStartCmd) Help() string { return "Start an ad hoc or declared task for a workspace" }

type taskAddCmd struct{}

func (c *taskAddCmd) Name() string { return "add" }
func (c *taskAddCmd) Help() string { return "Add or update a declared task for a workspace" }

func (c *taskAddCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("task add", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	cwd := fs.String("cwd", "", "relative or absolute working directory for the task")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot task add <workspace> <name> [--cwd dir] -- <cmd> [args...]")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}

	workspaceName, taskName, command, err := parseTaskDeclarationArgs(fs, args)
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

	if err := a.DeclareTask(workspaceName, app.TaskSpec{
		Name:    taskName,
		Command: append([]string{}, command...),
		Cwd:     strings.TrimSpace(*cwd),
	}); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Declared task %q in workspace %q.\n", taskName, workspaceName)
	return nil
}

type taskRemoveCmd struct{}

func (c *taskRemoveCmd) Name() string { return "remove" }
func (c *taskRemoveCmd) Help() string { return "Remove a declared task from a workspace" }

func (c *taskRemoveCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("task remove", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot task remove <workspace> <name>")
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
		return fmt.Errorf("workspace name and task name required")
	}

	workspaceName, err := requireWorkspaceArg(a, fs.Arg(0))
	if err != nil {
		return err
	}
	taskName := strings.TrimSpace(fs.Arg(1))
	if err := a.DeleteTask(workspaceName, taskName); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Removed task %q from workspace %q.\n", taskName, workspaceName)
	return nil
}

type taskListDeclaredCmd struct{}

func (c *taskListDeclaredCmd) Name() string { return "list-declared" }
func (c *taskListDeclaredCmd) Help() string { return "List declared tasks for a workspace" }

func (c *taskListDeclaredCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("task list-declared", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot task list-declared <workspace>")
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
	tasks, err := a.DeclaredTasks(workspaceName)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		fmt.Fprintln(os.Stdout, "No declared tasks.")
		return nil
	}
	for _, task := range tasks {
		fmt.Fprintf(os.Stdout, "%s\t%s\n", task.Name, strings.Join(task.Command, " "))
	}
	return nil
}

func (c *taskStartCmd) Run(a *app.App, args []string) error {
	if len(args) == 0 {
		fs := flag.NewFlagSet("task start", flag.ContinueOnError)
		fs.SetOutput(os.Stdout)
		fs.Usage = func() {
			fmt.Fprintln(fs.Output(), "usage: groot task start <workspace> [--name task-name] [--cwd dir] <cmd> [args...]")
			fmt.Fprintln(fs.Output(), "   or: groot task start <workspace> --task <declared-task-name>")
			fmt.Fprintln(fs.Output())
			fmt.Fprintln(fs.Output(), c.Help())
		}
		fs.Usage()
		return fmt.Errorf("workspace name required")
	}
	workspaceName := args[0]
	fs := flag.NewFlagSet("task start", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	displayName := fs.String("name", "", "display name for the ad hoc task")
	cwd := fs.String("cwd", "", "relative or absolute working directory for the task")
	declaredTask := fs.String("task", "", "start a declared task from the manifest by name")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot task start <workspace> [--name task-name] [--cwd dir] <cmd> [args...]")
		fmt.Fprintln(fs.Output(), "   or: groot task start <workspace> --task <declared-task-name>")
		fmt.Fprintln(fs.Output())
		fmt.Fprintln(fs.Output(), c.Help())
	}
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	workspaceName, err := requireWorkspaceArg(a, workspaceName)
	if err != nil {
		return err
	}
	if err := enforceWorkspaceOwnership(a, workspaceName); err != nil {
		return err
	}

	var task app.TaskRun
	if strings.TrimSpace(*declaredTask) != "" {
		if fs.NArg() != 0 {
			fs.Usage()
			return fmt.Errorf("declared task start does not accept an ad hoc command")
		}
		task, err = a.StartDeclaredTask(workspaceName, strings.TrimSpace(*declaredTask))
	} else {
		if fs.NArg() < 1 {
			fs.Usage()
			return fmt.Errorf("ad hoc task start requires a command")
		}
		task, err = a.StartTask(workspaceName, app.TaskStartSpec{
			Name:    strings.TrimSpace(*displayName),
			Command: fs.Arg(0),
			Args:    fs.Args()[1:],
			Cwd:     strings.TrimSpace(*cwd),
		})
	}
	if err != nil {
		return err
	}

	writeTaskRun(task)
	return nil
}

type taskStatusCmd struct{}

func (c *taskStatusCmd) Name() string { return "status" }
func (c *taskStatusCmd) Help() string { return "Print task status for a workspace and task id" }

func (c *taskStatusCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("task status", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot task status <workspace> <task-id>")
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
		return fmt.Errorf("workspace name and task id required")
	}

	workspaceName, err := requireWorkspaceArg(a, fs.Arg(0))
	if err != nil {
		return err
	}
	task, err := a.TaskStatus(workspaceName, fs.Arg(1))
	if err != nil {
		return err
	}
	writeTaskRun(task)
	return nil
}

type taskListCmd struct{}

func (c *taskListCmd) Name() string { return "list" }
func (c *taskListCmd) Help() string { return "List task runs for a workspace" }

func (c *taskListCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("task list", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot task list <workspace>")
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
	tasks, err := a.TaskList(workspaceName)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		fmt.Fprintln(os.Stdout, "No task runs.")
		return nil
	}
	for _, task := range tasks {
		fmt.Fprintf(os.Stdout, "%s\t%s\t%s\n", task.ID, task.State, task.Name)
	}
	return nil
}

type taskLogsCmd struct{}

func (c *taskLogsCmd) Name() string { return "logs" }
func (c *taskLogsCmd) Help() string { return "Print captured stdout and stderr for a task" }

func (c *taskLogsCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("task logs", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot task logs <workspace> <task-id>")
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
		return fmt.Errorf("workspace name and task id required")
	}

	workspaceName, err := requireWorkspaceArg(a, fs.Arg(0))
	if err != nil {
		return err
	}
	logs, err := a.TaskLogs(workspaceName, fs.Arg(1))
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

type taskStopCmd struct{}

func (c *taskStopCmd) Name() string { return "stop" }
func (c *taskStopCmd) Help() string { return "Stop a running task for a workspace" }

func (c *taskStopCmd) Run(a *app.App, args []string) error {
	fs := flag.NewFlagSet("task stop", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: groot task stop <workspace> <task-id>")
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
		return fmt.Errorf("workspace name and task id required")
	}

	workspaceName, err := requireWorkspaceArg(a, fs.Arg(0))
	if err != nil {
		return err
	}
	task, err := a.StopTask(workspaceName, fs.Arg(1))
	if err != nil {
		return err
	}
	writeTaskRun(task)
	return nil
}

func writeTaskRun(task app.TaskRun) {
	fmt.Fprintf(os.Stdout, "Task: %s\n", task.ID)
	fmt.Fprintf(os.Stdout, "Name: %s\n", task.Name)
	fmt.Fprintf(os.Stdout, "Workspace: %s\n", task.Workspace)
	fmt.Fprintf(os.Stdout, "State: %s\n", task.State)
	fmt.Fprintf(os.Stdout, "Command: %s\n", task.Command)
	if len(task.Args) > 0 {
		fmt.Fprintf(os.Stdout, "Args: %s\n", strings.Join(task.Args, " "))
	}
	fmt.Fprintf(os.Stdout, "Workdir: %s\n", task.Cwd)
	if task.Declared {
		fmt.Fprintln(os.Stdout, "Declared: yes")
	}
	if task.ExitCode != nil {
		fmt.Fprintf(os.Stdout, "Exit Code: %d\n", *task.ExitCode)
	}
	if task.CancelReason != "" {
		fmt.Fprintf(os.Stdout, "Cancel Reason: %s\n", task.CancelReason)
	}
}
