// Package act despacha as ações do ciclo executando os comandos existentes —
// a TUI nunca reimplementa lógica de domínio, só chama o dono do fato.
package act

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/henriquemeca/management-hub/internal/collect"
)

// Result é o desfecho de uma ação: rótulo curto para a status bar + tail da
// saída para inspeção.
type Result struct {
	Label  string // ex.: "ship HEN-42"
	Output string // stdout+stderr (tail)
	Err    error
}

// OK diz se a ação terminou sem erro.
func (r Result) OK() bool { return r.Err == nil }

// Summary comprime o resultado numa linha para a status bar.
func (r Result) Summary() string {
	if r.Err != nil {
		return fmt.Sprintf("✗ %s — %v", r.Label, r.Err)
	}
	if tail := lastLine(r.Output); tail != "" {
		return fmt.Sprintf("✓ %s — %s", r.Label, tail)
	}
	return fmt.Sprintf("✓ %s", r.Label)
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); l != "" {
			return l
		}
	}
	return ""
}

// Tail devolve as últimas n linhas da saída (painel de log de ações longas).
func (r Result) Tail(n int) string {
	lines := strings.Split(strings.TrimRight(r.Output, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// doIn roda um comando (cwd dir, env extra) e captura stdout+stderr juntos —
// ações são interativas por natureza, o humano quer ver o que aconteceu.
func doIn(timeout time.Duration, dir string, extraEnv []string, label, name string, args ...string) Result {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, collect.CommandPath(name), args...)
	cmd.Dir = dir
	cmd.WaitDelay = 5 * time.Second // filhos segurando o pipe não travam o Wait
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("timeout de %s", timeout)
	}
	return Result{Label: label, Output: string(out), Err: err}
}

func do(timeout time.Duration, label, name string, args ...string) Result {
	return doIn(timeout, "", nil, label, name, args...)
}

// ResolveGate responde um gate pendente.
func ResolveGate(id, resolution string) Result {
	return do(30*time.Second, "gate "+shortID(id),
		"gates", "resolve", "--id", id, "--resolution", resolution)
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// Spawn dispara `linear-sync --only KEY`; model/effort não-vazios viram
// overrides de env (perfil não-default escolhido no picker).
func Spawn(key, model, effort string) Result {
	var env []string
	if model != "" {
		env = append(env, "LINEAR_SYNC_MODEL="+model)
	}
	if effort != "" {
		env = append(env, "LINEAR_SYNC_EFFORT="+effort)
	}
	return doIn(2*time.Minute, "", env, "spawn "+key, "linear-sync", "--only", key)
}

// Ship fecha a issue: Done + remoção da worktree (guard-rails vivem no ship).
func Ship(key string) Result {
	return do(2*time.Minute, "ship "+key, "ship", key)
}

// Capture registra uma ideia crua via work. dir = repo do projeto selecionado
// quando houver (vínculo repo→issue nasce do cwd), senão vazio (cwd atual).
func Capture(title, desc, dir string) Result {
	args := []string{title}
	if strings.TrimSpace(desc) != "" {
		args = append(args, desc)
	}
	return doIn(90*time.Second, dir, nil, "capturar", "work", args...)
}

// Triage roda a triagem Backlog→Todo — de uma issue (only) ou de todas.
// Saída longa: o chamador mostra o tail.
func Triage(only string) Result {
	if only != "" {
		return do(10*time.Minute, "triage "+only, "triage", "--only", only)
	}
	return do(10*time.Minute, "triage", "triage")
}

// repoTrailerRe casa o trailer `repo: `host/owner/repo“ em linha própria;
// a última ocorrência vence (specs podem citar o formato como exemplo, HEN-37).
var repoTrailerRe = regexp.MustCompile("(?m)^\\s*repo:\\s*`([^`]+)`\\s*$")

// RepoDir resolve o diretório local do repo apontado pelo trailer `repo:` da
// description, via registro ~/.local/state/work-os/repos.json. Vazio = sem
// vínculo (o capture usa o cwd atual).
func RepoDir(description string) string {
	ms := repoTrailerRe.FindAllStringSubmatch(description, -1)
	if len(ms) == 0 {
		return ""
	}
	key := ms[len(ms)-1][1]
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(collect.StateDir(home), "repos.json"))
	if err != nil {
		return ""
	}
	var repos map[string]string
	if err := json.Unmarshal(b, &repos); err != nil {
		return ""
	}
	dir := repos[key]
	if dir == "" {
		return ""
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return ""
	}
	return dir
}

// FocusWindow foca a janela tmux do agente (a TUI segue na janela dela).
func FocusWindow(window string) Result {
	return do(10*time.Second, "focar "+window, "tmux", "select-window", "-t", window)
}

// OpenURL abre a issue no browser.
func OpenURL(url string) Result {
	return do(15*time.Second, "abrir "+url, "open", url)
}
