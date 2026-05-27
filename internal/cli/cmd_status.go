package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show runtime status from configuration",
		Long: "Displays the current HomeStock configuration status including " +
			"server info, outbound endpoints, and CLI version. " +
			"Does not require a running server.",
		RunE: func(cmd *cobra.Command, args []string) error {
			f := cmd.Root().PersistentFlags().Lookup("config")
			cfgPath := defaultConfigPath
			if f.Changed {
				cfgPath = f.Value.String()
			}

			cfg, err := config.Load(cfgPath)
			if err != nil {
				return asRuntimeError(fmt.Errorf("load config: %w", err))
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "HomeStock Status")
			fmt.Fprintln(out, "=================")
			fmt.Fprintf(out, "  CLI Version:  %s\n", Version)
			fmt.Fprintf(out, "  Commit:       %s\n", Commit)
			fmt.Fprintf(out, "  Build Date:   %s\n", Date)
			fmt.Fprintf(out, "  Server Port:  %s\n", cfg.Server.Port)
			fmt.Fprintf(out, "  Database:     %s (%s)\n", cfg.Database.Driver, cfg.Database.DSN)
			fmt.Fprintf(out, "  Outbound Endpoints: %d configured\n", len(cfg.Outbound.Endpoints))
			for _, ep := range cfg.Outbound.Endpoints {
				fmt.Fprintf(out, "    - %s (%s, enabled: %t)\n", ep.Name, ep.URL, ep.Enabled)
			}

			return nil
		},
	}
}
