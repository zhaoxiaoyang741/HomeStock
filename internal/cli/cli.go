package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

// Version information — set via -ldflags at build time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

const defaultConfigPath = "config.json"

// runtimeError marks an error as a runtime failure (exit code 1) as opposed to
// a CLI usage error (exit code 2).
type runtimeError struct{ error }

func (e *runtimeError) Unwrap() error { return e.error }

func asRuntimeError(err error) error {
	if err == nil {
		return nil
	}
	return &runtimeError{err}
}

func isRuntimeError(err error) bool {
	if err == nil {
		return false
	}
	var e *runtimeError
	return errors.As(err, &e)
}

// Execute is the single entry point. It returns an exit code suitable for
// os.Exit: 0 on success, 1 for runtime errors, 2 for CLI usage errors.
func Execute(args []string, stdout, stderr io.Writer) int {
	rootCmd := newRootCmd()
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	if args == nil {
		args = []string{}
	}
	rootCmd.SetArgs(args)

	_, err := rootCmd.ExecuteC()
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		if isRuntimeError(err) {
			return 1
		}
		return 2
	}
	return 0
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "homestock",
		Short: "HomeStock inventory management bot",
		Long: "HomeStock is an inventory management bot that integrates with " +
			"Feishu and WeChat for tracking material stock, inbound/outbound " +
			"movements, and expiry notifications.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().String("config", "", "Path to config JSON file")

	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newServerCmd())
	cmd.AddCommand(newGatewayCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newVersionCmd())

	return cmd
}

// ---------------------------------------------------------------------------
// Config commands
// ---------------------------------------------------------------------------

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newConfigShowCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the effective configuration as JSON",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("config show does not accept positional arguments: %v", args)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Backward compatible config path resolution:
			//   — if --config was explicitly passed (even ""), use it verbatim
			//   — otherwise default to "config.json"
			f := cmd.Root().PersistentFlags().Lookup("config")
			cfgPath := defaultConfigPath
			if f.Changed {
				cfgPath = f.Value.String()
			}

			cfg, err := config.Load(cfgPath)
			if err != nil {
				return asRuntimeError(fmt.Errorf("load config: %w", err))
			}

			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(cfg); err != nil {
				return asRuntimeError(fmt.Errorf("write config: %w", err))
			}
			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// resolveConfigPath resolves the config path with the fallback chain:
//  1. --config flag value (non-empty)
//  2. HOMESTOCK_CONFIG_PATH environment variable
//  3. default "config.json"
func resolveConfigPath(cmd *cobra.Command) string {
	path, _ := cmd.Root().PersistentFlags().GetString("config")
	if path == "" {
		path = os.Getenv("HOMESTOCK_CONFIG_PATH")
	}
	if path == "" {
		path = defaultConfigPath
	}
	return path
}
