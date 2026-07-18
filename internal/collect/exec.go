package collect

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Options parametriza uma coleta.
type Options struct {
	Timeout time.Duration // timeout por exec (0 → 15s)
	Home    string        // home do usuário (0 → os.UserHomeDir)
}

func (o Options) timeout() time.Duration {
	if o.Timeout <= 0 {
		return 15 * time.Second
	}
	return o.Timeout
}

func (o Options) home() string {
	if o.Home != "" {
		return o.Home
	}
	h, _ := os.UserHomeDir()
	return h
}

// CommandPath resolve um comando do work-os: PATH, senão bin/ ao lado do
// binário (dist/wos → ../bin), senão ~/.local/bin.
func CommandPath(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	if self, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(self); err == nil {
			self = resolved
		}
		p := filepath.Join(filepath.Dir(self), "..", "bin", name)
		if info, err := os.Stat(p); err == nil && info.Mode()&0o111 != 0 {
			return p
		}
	}
	if h, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(h, ".local", "bin", name)
		if info, err := os.Stat(p); err == nil && info.Mode()&0o111 != 0 {
			return p
		}
	}
	return name // deixa o exec falhar com a mensagem natural
}

// run executa um comando com timeout e devolve o stdout. stderr só entra no
// erro (as fontes degradam para vazio + aviso; nunca travam a TUI).
func run(ctx context.Context, timeout time.Duration, dir, name string, args ...string) ([]byte, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, CommandPath(name), args...)
	cmd.Dir = dir
	cmd.WaitDelay = 2 * time.Second // filhos segurando o pipe não travam o Wait
	var errb strings.Builder
	cmd.Stderr = &errb
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(errb.String())
		if cctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("%s: timeout de %s", name, timeout)
		}
		if msg != "" {
			if len(msg) > 200 {
				msg = msg[:200] + "…"
			}
			return nil, fmt.Errorf("%s: %s", name, msg)
		}
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

// PIDAlive testa o processo com kill(pid, 0). EPERM = vivo (só não é nosso).
func PIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// StateDir devolve o diretório de estado do work-os (WORK_OS_STATE sobrepõe).
func StateDir(home string) string {
	if dir := os.Getenv("WORK_OS_STATE"); dir != "" {
		return dir
	}
	return filepath.Join(home, ".local", "state", "work-os")
}

// AgentsPath devolve o caminho do agents.jsonl.
func AgentsPath(home string) string {
	return filepath.Join(StateDir(home), "agents.jsonl")
}

// capFromEnv lê LINEAR_SYNC_CAP (default 2).
func capFromEnv() int {
	if v := os.Getenv("LINEAR_SYNC_CAP"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 2
}

// worktreePath resolve o worktree de um slug: ~/worktrees/*/<slug> (glob);
// fallback: path do pane.
func worktreePath(home, slug, panePath string) string {
	matches, _ := filepath.Glob(filepath.Join(home, "worktrees", "*", slug))
	for _, m := range matches {
		if info, err := os.Stat(m); err == nil && info.IsDir() {
			return m
		}
	}
	return panePath
}

// listWorktreeDirs enumera ~/worktrees/*/* (diretórios).
func listWorktreeDirs(home string) []string {
	matches, _ := filepath.Glob(filepath.Join(home, "worktrees", "*", "*"))
	var dirs []string
	for _, m := range matches {
		if info, err := os.Stat(m); err == nil && info.IsDir() {
			dirs = append(dirs, m)
		}
	}
	return dirs
}

// FetchIssueDetail busca a issue completa (comments + children) sob demanda.
func FetchIssueDetail(ctx context.Context, key string) (*IssueDetail, error) {
	out, err := run(ctx, 15*time.Second, "", "linear-gql", "issue", key)
	if err != nil {
		return nil, err
	}
	return ParseIssueDetail(out)
}

// Collect monta o Snapshot completo: fontes independentes em paralelo, cada
// falha degrada para seção vazia + aviso — nunca erro fatal.
func Collect(ctx context.Context, o Options) Snapshot {
	timeout := o.timeout()
	home := o.home()

	snap := Snapshot{Cap: capFromEnv(), CollectedAt: time.Now()}
	var mu sync.Mutex
	warn := func(format string, a ...any) {
		mu.Lock()
		snap.Warnings = append(snap.Warnings, fmt.Sprintf(format, a...))
		mu.Unlock()
	}

	var windows []Window
	var wg sync.WaitGroup
	launch := func(fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	launch(func() {
		out, err := run(ctx, timeout, "", "linear-sync", "--queue", "--json")
		if err != nil {
			warn("fila: %v", err)
			return
		}
		q, err := ParseQueue(out)
		if err != nil {
			warn("fila: %v", err)
			return
		}
		mu.Lock()
		snap.Queue = q
		mu.Unlock()
	})

	launch(func() {
		team := os.Getenv("WORK_LINEAR_TEAM")
		if team == "" {
			team = "HEN"
		}
		out, err := run(ctx, timeout, "", "linear-gql", "list", "--team", team, "--limit", "100")
		if err != nil {
			warn("plano: %v", err)
			return
		}
		issues, err := ParseIssues(out)
		if err != nil {
			warn("plano: %v", err)
			return
		}
		mu.Lock()
		snap.Issues = issues
		mu.Unlock()
	})

	launch(func() {
		out, err := run(ctx, timeout, "", "gates", "list", "--status", "pending", "--json")
		if err != nil {
			warn("gates: %v", err)
			return
		}
		gs, err := ParseGates(out)
		if err != nil {
			warn("gates: %v", err)
			return
		}
		mu.Lock()
		snap.Gates = gs
		mu.Unlock()
	})

	launch(func() {
		f, err := os.Open(AgentsPath(home))
		if err != nil {
			if !os.IsNotExist(err) {
				warn("sessões: %v", err)
			}
			return
		}
		defer f.Close()
		sessions := ParseAgents(f, PIDAlive)
		mu.Lock()
		snap.Sessions = sessions
		mu.Unlock()
	})

	launch(func() {
		out, err := run(ctx, timeout, "", "tmux", "list-panes", "-a", "-F",
			"#{window_name}\t#{pane_current_path}")
		if err != nil {
			warn("tmux: %v", err)
			return
		}
		mu.Lock()
		windows = ParseTmuxPanes(string(out))
		mu.Unlock()
	})

	wg.Wait()

	// Worktrees: união (janelas wm-*) ∪ (diretórios em ~/worktrees) —
	// worktree sem janela é órfã, janela wm- sem sessão continua execução.
	agents := AgentWindows(windows)
	snap.Occupied = len(agents)

	byPath := map[string]*Worktree{}
	var order []string
	for _, w := range agents {
		path := worktreePath(home, w.Slug(), w.Path)
		if _, ok := byPath[path]; !ok {
			byPath[path] = &Worktree{Slug: w.Slug(), Path: path, Window: w.Name}
			order = append(order, path)
		}
	}
	for _, dir := range listWorktreeDirs(home) {
		if _, ok := byPath[dir]; !ok {
			byPath[dir] = &Worktree{Slug: filepath.Base(dir), Path: dir}
			order = append(order, dir)
		}
	}
	sort.Strings(order)

	// Branch + PR por worktree em paralelo (gh falhando = sem info, não erro).
	var prwg sync.WaitGroup
	for _, path := range order {
		wt := byPath[path]
		wt.Session = MatchSession(snap.Sessions, wt.Path)
		prwg.Add(1)
		go func(wt *Worktree) {
			defer prwg.Done()
			bout, err := run(ctx, timeout, "", "git", "-C", wt.Path, "rev-parse", "--abbrev-ref", "HEAD")
			if err != nil {
				return
			}
			wt.Branch = strings.TrimSpace(string(bout))
			if wt.Branch == "" {
				return
			}
			pout, err := run(ctx, timeout, wt.Path, "gh", "pr", "list", "--head", wt.Branch,
				"--state", "all", "--json", "number,state,mergedAt")
			if err != nil {
				return // sem info de PR — não é erro
			}
			if pr, err := ParsePRs(pout); err == nil {
				wt.PR = pr
			}
		}(wt)
	}
	prwg.Wait()

	for _, path := range order {
		snap.Worktrees = append(snap.Worktrees, *byPath[path])
	}
	return snap
}
