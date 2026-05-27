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
			"the active model, configured channels, and CLI version. " +
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

			active, err := cfg.ActiveModelConfig()
			if err == nil {
				fmt.Fprintf(out, "  Active Model: %s (%s, %s)\n",
					active.ModelName, active.Model, active.Provider)
			} else {
				fmt.Fprintf(out, "  Active Model: none (%v)\n", err)
			}

			channelCount := len(cfg.Channels)
			fmt.Fprintf(out, "  Channels:     %d configured\n", channelCount)
			for name := range cfg.Channels {
				fmt.Fprintf(out, "    - %s\n", name)
			}

			return nil
		},
	}
}
