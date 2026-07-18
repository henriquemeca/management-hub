package commands

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/henriquemeca/management-hub/internal/collect"
	"github.com/henriquemeca/management-hub/internal/ui"
)

var uiSmoke bool

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Abre a TUI — a mesa única do ciclo (mesmo code path do wos sem args)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUI(uiSmoke)
	},
}

func init() {
	uiCmd.Flags().BoolVar(&uiSmoke, "smoke", false,
		"coleta uma vez, renderiza um frame no stdout e sai (verificação headless)")
}

// runUI roda a TUI; smoke = um frame headless com dados reais e exit 0.
func runUI(smoke bool) error {
	if smoke {
		snap := collect.Collect(context.Background(), collect.Options{Timeout: 8 * time.Second})
		m := ui.NewSmoke(snap, 120, 35)
		fmt.Println(m.View())
		return nil
	}
	p := tea.NewProgram(ui.New(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
