package collect

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// ParseQueue lê a saída de `linear-sync --queue --json`:
// {"ok":true,"result":{"queue":[...]}}.
func ParseQueue(b []byte) ([]QueueItem, error) {
	var env struct {
		OK     bool `json:"ok"`
		Result struct {
			Queue []QueueItem `json:"queue"`
		} `json:"result"`
	}
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, fmt.Errorf("queue: %w", err)
	}
	if !env.OK {
		return nil, fmt.Errorf("queue: ok=false")
	}
	return env.Result.Queue, nil
}

// ParseIssues lê a saída de `linear-gql list`: {"issues":[...]}.
func ParseIssues(b []byte) ([]Issue, error) {
	var env struct {
		Issues []Issue `json:"issues"`
	}
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, fmt.Errorf("issues: %w", err)
	}
	return env.Issues, nil
}

// ParseIssueDetail lê a saída de `linear-gql issue KEY` (issue + comments + children).
func ParseIssueDetail(b []byte) (*IssueDetail, error) {
	var d IssueDetail
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("issue: %w", err)
	}
	if d.Identifier == "" {
		return nil, fmt.Errorf("issue: sem identifier na resposta")
	}
	return &d, nil
}

// ParseGates lê a saída de `gates list --json`: {"gates":[...]}.
func ParseGates(b []byte) ([]Gate, error) {
	var env struct {
		Gates []Gate `json:"gates"`
	}
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, fmt.Errorf("gates: %w", err)
	}
	return env.Gates, nil
}

// agentEvent é uma linha do agents.jsonl (hooks do Claude Code).
type agentEvent struct {
	TS        int64  `json:"ts"`
	Event     string `json:"event"`
	Cwd       string `json:"cwd"`
	SessionID string `json:"session_id"`
	PID       int    `json:"pid"`
}

// stateForEvent mapeia evento de hook → estado da sessão. Evento desconhecido
// (PostToolUse etc.) é atividade, logo working.
func stateForEvent(event string) string {
	switch event {
	case "SessionStart", "UserPromptSubmit":
		return StateWorking
	case "Stop":
		return StateWaiting
	case "PermissionRequest":
		return StateBlocked
	default:
		return StateWorking
	}
}

// ParseAgents deriva as sessões vivas do agents.jsonl: último evento
// não-SubagentStop por session_id define estado/cwd/pid; sessões cujo pid não
// responde a alive() são descartadas (processo morto). Linhas inválidas ou sem
// session_id são ignoradas.
func ParseAgents(r io.Reader, alive func(pid int) bool) []Session {
	last := map[string]agentEvent{}
	order := []string{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var ev agentEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if ev.SessionID == "" || ev.Event == "SubagentStop" {
			continue
		}
		if _, seen := last[ev.SessionID]; !seen {
			order = append(order, ev.SessionID)
		}
		last[ev.SessionID] = ev
	}

	var out []Session
	for _, id := range order {
		ev := last[id]
		if !alive(ev.PID) {
			continue
		}
		out = append(out, Session{
			SessionID: id,
			State:     stateForEvent(ev.Event),
			Cwd:       ev.Cwd,
			PID:       ev.PID,
			TS:        ev.TS,
		})
	}
	return out
}

// ParseTmuxPanes lê `tmux list-panes -a -F '#{window_name}\t#{pane_current_path}'`
// e devolve uma janela por nome (primeiro pane vence).
func ParseTmuxPanes(out string) []Window {
	seen := map[string]bool{}
	var ws []Window
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		name, path, ok := strings.Cut(line, "\t")
		if !ok || name == "" || seen[name] {
			continue
		}
		seen[name] = true
		ws = append(ws, Window{Name: name, Path: path})
	}
	return ws
}

// AgentWindowPrefix marca janelas de agente (workmux).
const AgentWindowPrefix = "wm-"

// AgentWindows filtra as janelas de agente (prefixo wm-).
func AgentWindows(ws []Window) []Window {
	var out []Window
	for _, w := range ws {
		if strings.HasPrefix(w.Name, AgentWindowPrefix) {
			out = append(out, w)
		}
	}
	return out
}

// Slug extrai o slug de uma janela de agente (nome sem o prefixo wm-).
func (w Window) Slug() string { return strings.TrimPrefix(w.Name, AgentWindowPrefix) }

// CwdMatches diz se cwd está dentro do worktree: igual ao path ou subdiretório
// dele — nunca o contrário.
func CwdMatches(cwd, wtPath string) bool {
	if cwd == "" || wtPath == "" {
		return false
	}
	cwd = filepath.Clean(cwd)
	wtPath = filepath.Clean(wtPath)
	return cwd == wtPath || strings.HasPrefix(cwd, wtPath+string(filepath.Separator))
}

// MatchSession acha a sessão viva cujo cwd cai dentro do worktree. Empate:
// evento mais recente vence.
func MatchSession(sessions []Session, wtPath string) *Session {
	var best *Session
	for i := range sessions {
		s := &sessions[i]
		if !CwdMatches(s.Cwd, wtPath) {
			continue
		}
		if best == nil || s.TS > best.TS {
			best = s
		}
	}
	return best
}

// ParsePRs lê a saída de `gh pr list --json number,state,mergedAt` e escolhe o
// PR mais relevante: OPEN > MERGED > primeiro (gh ordena por recência).
func ParsePRs(b []byte) (*PR, error) {
	var prs []PR
	if err := json.Unmarshal(b, &prs); err != nil {
		return nil, fmt.Errorf("prs: %w", err)
	}
	if len(prs) == 0 {
		return nil, nil
	}
	for i := range prs {
		if prs[i].State == "OPEN" {
			return &prs[i], nil
		}
	}
	for i := range prs {
		if prs[i].State == "MERGED" {
			return &prs[i], nil
		}
	}
	return &prs[0], nil
}
