package commands

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
)

// Item é uma linha do feed do hub — o contrato que painéis (fzf, TUI) renderizam.
// Espelha o proto-feed do `wos ps --tsv` (kind/handle/label) e o modelo de dados
// de docs/03-hub.md; campos novos só entram com versão.
type Item struct {
	Kind    string `json:"kind"`    // gate | ask | input | review | integrate | spawn | stuck | orphan | run | queue
	ID      string `json:"id"`      // chave no sistema de origem (gate id, issue key, repo@dev)
	AgeS    int64  `json:"age_s"`   // idade em segundos (0 = desconhecida)
	Urgency string `json:"urgency"` // blocking | ready | capacity | hygiene
	Handle  string `json:"handle"`  // pane/sessão para focar (vazio = sem pane)
	Label   string `json:"label"`   // linha exibível, já com o dado da decisão
	Action  string `json:"action"`  // ação default do enter
}

// Feed é o envelope versionado consumido por qualquer superfície.
type Feed struct {
	Version int    `json:"version"`
	Items   []Item `json:"items"`
}

var feedCmd = &cobra.Command{
	Use:   "feed",
	Short: "Emite o feed do hub em JSON (contrato das superfícies: fzf, status bar, TUI)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// v0: envelope vazio — os coletores (Linear, workmux, gh, telemetria JSONL)
		// entram por graduação, um por vez, cada um com teste.
		feed := Feed{Version: 1, Items: []Item{}}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(feed)
	},
}
