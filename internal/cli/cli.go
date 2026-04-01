package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

const defaultConfigPath = "config.json"

// Execute runs the HomeStock CLI and returns an exit code.
func Execute(args []string, stdout, stderr io.Writer) int {
	runner := runner{
		stdout: stdout,
		stderr: stderr,
	}

	return runner.execute(args)
}

type runner struct {
	stdout io.Writer
	stderr io.Writer
}

func (r runner) execute(args []string) int {
	if len(args) == 0 {
		r.printUsage()
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		r.printUsage()
		return 0
	case "config":
		return r.runConfig(args[1:])
	default:
		fmt.Fprintf(r.stderr, "unknown command %q\n\n", args[0])
		r.printUsage()
		return 2
	}
}

func (r runner) runConfig(args []string) int {
	if len(args) == 0 {
		r.printConfigUsage()
		return 0
	}

	switch args[0] {
	case "show":
		return r.runConfigShow(args[1:])
	case "help", "-h", "--help":
		r.printConfigUsage()
		return 0
	default:
		fmt.Fprintf(r.stderr, "unknown config command %q\n\n", args[0])
		r.printConfigUsage()
		return 2
	}
}

func (r runner) runConfigShow(args []string) int {
	fs := flag.NewFlagSet("config show", flag.ContinueOnError)
	fs.SetOutput(r.stderr)

	configPath := fs.String("config", defaultConfigPath, "Path to config JSON file")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if fs.NArg() != 0 {
		fmt.Fprintf(r.stderr, "config show does not accept positional arguments: %v\n", fs.Args())
		return 2
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(r.stderr, "load config: %v\n", err)
		return 1
	}

	encoder := json.NewEncoder(r.stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(cfg); err != nil {
		fmt.Fprintf(r.stderr, "write config: %v\n", err)
		return 1
	}

	return 0
}

func (r runner) printUsage() {
	fmt.Fprintln(r.stdout, "HomeStock CLI")
	fmt.Fprintln(r.stdout)
	fmt.Fprintln(r.stdout, "Usage:")
	fmt.Fprintln(r.stdout, "  homestock <command> [flags]")
	fmt.Fprintln(r.stdout)
	fmt.Fprintln(r.stdout, "Available Commands:")
	fmt.Fprintln(r.stdout, "  config show   Print the effective configuration as JSON")
}

func (r runner) printConfigUsage() {
	fmt.Fprintln(r.stdout, "Usage:")
	fmt.Fprintln(r.stdout, "  homestock config show [--config path]")
}
