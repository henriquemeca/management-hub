// Package collect lê o estado do ciclo nas fontes donas de cada fato
// (linear-sync, linear-gql, gates, agents.jsonl, tmux, gh) e o projeta num
// Snapshot imutável. Nenhum dado nasce aqui: parsers puros sobre a saída dos
// comandos existentes — o contrato único do work-os.
package collect

import "time"

// QueueItem é uma linha de `linear-sync --queue --json` (ordem já computada lá).
type QueueItem struct {
	Key           string `json:"key"`
	Priority      int    `json:"priority"`
	PriorityLabel string `json:"priorityLabel"`
	Title         string `json:"title"`
}

// Issue é uma issue aberta de `linear-gql list` (Done não vem).
type Issue struct {
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	URL         string `json:"url"`
	State       struct {
		Name string `json:"name"`
	} `json:"state"`
	Team struct {
		Key string `json:"key"`
	} `json:"team"`
	Project *struct {
		Name string `json:"name"`
	} `json:"project"`
	Parent *struct {
		Identifier string `json:"identifier"`
	} `json:"parent"`
	BranchName string `json:"branchName"`
	UpdatedAt  string `json:"updatedAt"`
}

// ProjectName devolve o nome do projeto ou vazio.
func (i Issue) ProjectName() string {
	if i.Project == nil {
		return ""
	}
	return i.Project.Name
}

// Comment é um comentário de `linear-gql issue KEY`.
type Comment struct {
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

// IssueDetail é a issue completa de `linear-gql issue KEY`.
type IssueDetail struct {
	Issue
	Comments struct {
		Nodes []Comment `json:"nodes"`
	} `json:"comments"`
	Children struct {
		Nodes []Issue `json:"nodes"`
	} `json:"children"`
}

// Gate é uma decisão pendente de `gates list --status pending --json`.
type Gate struct {
	ID        string `json:"id"`
	Task      string `json:"task"`
	Question  string `json:"question"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

// Estados derivados de sessão (agents.jsonl).
const (
	StateWorking = "working"
	StateWaiting = "waiting"
	StateBlocked = "blocked"
)

// Session é o estado derivado de uma sessão viva do Claude Code.
type Session struct {
	SessionID string
	State     string // working | waiting | blocked
	Cwd       string
	PID       int
	TS        int64 // ts do último evento considerado
}

// Window é uma janela tmux (deduplicada) com o path do primeiro pane.
type Window struct {
	Name string
	Path string
}

// PR é o pull request mais relevante de uma branch (via gh).
type PR struct {
	Number   int    `json:"number"`
	State    string `json:"state"` // OPEN | CLOSED | MERGED
	MergedAt string `json:"mergedAt"`
}

// Worktree junta worktree, janela tmux, branch, PR e sessão — a unidade de
// execução que a TUI projeta.
type Worktree struct {
	Slug    string
	Path    string
	Window  string // janela tmux wm-<slug>; vazio = órfã
	Branch  string
	PR      *PR
	Session *Session // sessão viva com cwd dentro do worktree; nil = sem sessão
}

// Merged diz se o PR do worktree está merged.
func (w Worktree) Merged() bool { return w.PR != nil && w.PR.State == "MERGED" }

// Snapshot é a foto completa que a TUI renderiza.
type Snapshot struct {
	Queue       []QueueItem
	Issues      []Issue
	Gates       []Gate
	Sessions    []Session
	Worktrees   []Worktree
	Cap         int
	Occupied    int // nº de janelas wm-*
	Warnings    []string
	CollectedAt time.Time
}
