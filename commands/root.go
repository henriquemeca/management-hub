package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0-dev"

var rootCmd = &cobra.Command{
	Use:   "wos",
	Short: "Hub do work-os: plano (Linear), execução (workmux) e atenção numa superfície só",
	Long: `wos é o hub do work-os: projeta o plano (Linear), a execução (workmux/tmux)
e a integração (GitHub) numa superfície única, e despacha ações na fonte.

Os scripts zsh em bin/ são a implementação vigente; os comandos migram para cá
por graduação, sempre nascendo com teste (docs/05-roadmap.md).`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Imprime a versão",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

func Execute() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(feedCmd)
	rootCmd.AddCommand(uiCmd)
	// wos sem args = a TUI (mesmo code path do `wos ui`) — a mesa única.
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runUI(false)
	}
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
