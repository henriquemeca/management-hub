package collect

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
)

func TestParseQueue(t *testing.T) {
	fixture := `{"ok": true, "result": {"queue": [
		{"key": "HEN-58", "priority": 3, "priorityLabel": "P3", "title": "captura à prova de falha"},
		{"key": "HEN-60", "priority": 0, "priorityLabel": "—", "title": "sem prioridade"}
	]}}`
	q, err := ParseQueue([]byte(fixture))
	if err != nil {
		t.Fatalf("ParseQueue: %v", err)
	}
	if len(q) != 2 {
		t.Fatalf("len = %d, esperado 2", len(q))
	}
	if q[0].Key != "HEN-58" || q[0].Priority != 3 || q[0].PriorityLabel != "P3" {
		t.Errorf("primeiro item errado: %+v", q[0])
	}
	if q[1].Title != "sem prioridade" {
		t.Errorf("título do segundo item: %q", q[1].Title)
	}
}

func TestParseQueueNotOK(t *testing.T) {
	if _, err := ParseQueue([]byte(`{"ok": false, "error": "x"}`)); err == nil {
		t.Error("ok=false deveria ser erro")
	}
	if _, err := ParseQueue([]byte(`não é json`)); err == nil {
		t.Error("JSON inválido deveria ser erro")
	}
}

func TestParseIssues(t *testing.T) {
	fixture := `{"issues": [
		{"identifier": "HEN-84", "title": "plan-store local", "description": "d",
		 "priority": 3, "url": "https://linear.app/x/HEN-84",
		 "state": {"name": "Backlog"}, "team": {"key": "HEN"},
		 "project": {"name": "management-hub"}, "parent": null,
		 "branchName": "u/hen-84-plan-store", "updatedAt": "2026-07-18T13:56:14.839Z"},
		{"identifier": "HEN-80", "title": "escalação", "description": "",
		 "priority": 0, "url": "u", "state": {"name": "Todo"}, "team": {"key": "HEN"},
		 "project": null, "parent": {"identifier": "HEN-59"},
		 "branchName": "u/hen-80", "updatedAt": "2026-07-18T13:21:35.765Z"}
	]}`
	issues, err := ParseIssues([]byte(fixture))
	if err != nil {
		t.Fatalf("ParseIssues: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("len = %d, esperado 2", len(issues))
	}
	if issues[0].ProjectName() != "management-hub" || issues[1].ProjectName() != "" {
		t.Errorf("ProjectName: %q / %q", issues[0].ProjectName(), issues[1].ProjectName())
	}
	if issues[1].Parent == nil || issues[1].Parent.Identifier != "HEN-59" {
		t.Errorf("parent do segundo: %+v", issues[1].Parent)
	}
	if issues[0].State.Name != "Backlog" {
		t.Errorf("state: %q", issues[0].State.Name)
	}
}

func TestParseIssueDetail(t *testing.T) {
	fixture := `{"identifier": "HEN-58", "title": "captura", "state": {"name": "Todo"},
		"comments": {"nodes": [{"body": "## Triagem", "createdAt": "2026-07-18T13:20:30Z"}]},
		"children": {"nodes": []}}`
	d, err := ParseIssueDetail([]byte(fixture))
	if err != nil {
		t.Fatalf("ParseIssueDetail: %v", err)
	}
	if d.Identifier != "HEN-58" || len(d.Comments.Nodes) != 1 {
		t.Errorf("detalhe errado: %+v", d)
	}
	if _, err := ParseIssueDetail([]byte(`{}`)); err == nil {
		t.Error("resposta sem identifier deveria ser erro")
	}
}

func TestParseGates(t *testing.T) {
	fixture := `{"gates": [{"id": "a391ed9379bd", "task": "triagem HEN-17",
		"question": "[HEN-17] repo alvo?", "status": "pending",
		"created_at": 1784398468, "resolution": null, "resolved_at": null}]}`
	gs, err := ParseGates([]byte(fixture))
	if err != nil {
		t.Fatalf("ParseGates: %v", err)
	}
	if len(gs) != 1 || gs[0].ID != "a391ed9379bd" || gs[0].CreatedAt != 1784398468 {
		t.Errorf("gate errado: %+v", gs)
	}
}

// jl monta uma linha do agents.jsonl.
func jl(ts int64, event, cwd, sid string, pid int) string {
	return fmt.Sprintf(`{"ts": %d, "event": %q, "cwd": %q, "session_id": %q, "pid": %d}`,
		ts, event, cwd, sid, pid)
}

func TestParseAgentsDerivation(t *testing.T) {
	alivePID := os.Getpid() // vivo de verdade
	deadPID := 4194300      // perto do máximo do kernel; quase certamente morto
	fixture := strings.Join([]string{
		jl(10, "SessionStart", "/wt/a", "sess-a", alivePID),
		jl(11, "UserPromptSubmit", "/wt/a", "sess-a", alivePID),
		jl(12, "Stop", "/wt/a", "sess-a", alivePID),
		jl(13, "SubagentStop", "/wt/a", "sess-a", alivePID), // não conta como último
		jl(20, "SessionStart", "/wt/b", "sess-b", alivePID),
		jl(21, "PermissionRequest", "/wt/b", "sess-b", alivePID),
		jl(30, "SessionStart", "/wt/c", "sess-c", alivePID),
		jl(40, "Stop", "/wt/dead", "sess-dead", deadPID),                         // pid morto → ignorar
		`{"ts": 50, "event": "--help", "cwd": "/x", "session_id": "", "pid": 1}`, // sem sessão
		"linha inválida{",
		"",
	}, "\n")

	real := func(pid int) bool { return syscall.Kill(pid, 0) == nil }
	sessions := ParseAgents(strings.NewReader(fixture), real)

	byID := map[string]Session{}
	for _, s := range sessions {
		byID[s.SessionID] = s
	}
	if len(sessions) != 3 {
		t.Fatalf("sessões vivas = %d (%v), esperado 3", len(sessions), byID)
	}
	// SubagentStop ignorado → último evento de sess-a é Stop (ts 12) → waiting
	if s := byID["sess-a"]; s.State != StateWaiting || s.TS != 12 {
		t.Errorf("sess-a: %+v, esperado waiting ts=12", s)
	}
	if s := byID["sess-b"]; s.State != StateBlocked {
		t.Errorf("sess-b: %+v, esperado blocked", s)
	}
	if s := byID["sess-c"]; s.State != StateWorking {
		t.Errorf("sess-c: %+v, esperado working", s)
	}
	if _, ok := byID["sess-dead"]; ok {
		t.Error("sess-dead (pid morto) não deveria aparecer")
	}
}

func TestParseAgentsAliveFuncIsHonored(t *testing.T) {
	fixture := jl(1, "Stop", "/wt/x", "sess-x", 123)
	if got := ParseAgents(strings.NewReader(fixture), func(int) bool { return false }); len(got) != 0 {
		t.Errorf("alive=false deveria descartar tudo, veio %+v", got)
	}
	if got := ParseAgents(strings.NewReader(fixture), func(int) bool { return true }); len(got) != 1 {
		t.Errorf("alive=true deveria manter a sessão, veio %+v", got)
	}
}

func TestParseTmuxPanes(t *testing.T) {
	out := "nvim\t/Users/h/.dotfiles\n" +
		"wm-hen-85-doctor\t/Users/h/worktrees/management-hub/hen-85-doctor\n" +
		"wm-hen-85-doctor\t/Users/h/worktrees/management-hub/hen-85-doctor/sub\n" + // 2º pane
		"zsh\t/Users/h\n" +
		"wm-hen-90-x\t/Users/h/worktrees/dotfiles/hen-90-x\n"
	ws := ParseTmuxPanes(out)
	if len(ws) != 4 {
		t.Fatalf("janelas = %d, esperado 4 (dedup por nome)", len(ws))
	}
	ag := AgentWindows(ws)
	if len(ag) != 2 {
		t.Fatalf("janelas de agente = %d, esperado 2", len(ag))
	}
	if ag[0].Slug() != "hen-85-doctor" || ag[1].Slug() != "hen-90-x" {
		t.Errorf("slugs: %q, %q", ag[0].Slug(), ag[1].Slug())
	}
	// dedup mantém o primeiro pane da janela
	if ag[0].Path != "/Users/h/worktrees/management-hub/hen-85-doctor" {
		t.Errorf("path do primeiro pane: %q", ag[0].Path)
	}
}

func TestCwdMatches(t *testing.T) {
	wt := "/Users/h/worktrees/management-hub/hen-85"
	cases := []struct {
		cwd  string
		want bool
	}{
		{wt, true},                    // igual
		{wt + "/internal/ui", true},   // subdiretório
		{wt + "-outro", false},        // prefixo de string ≠ subdiretório
		{"/Users/h/worktrees", false}, // pai do worktree — nunca o contrário
		{"/Users/h/.dotfiles", false}, // sem relação
		{wt + "/./internal", true},    // path não normalizado
		{"", false},                   // vazio
	}
	for _, c := range cases {
		if got := CwdMatches(c.cwd, wt); got != c.want {
			t.Errorf("CwdMatches(%q) = %v, esperado %v", c.cwd, got, c.want)
		}
	}
}

func TestMatchSession(t *testing.T) {
	wt := "/wt/a"
	sessions := []Session{
		{SessionID: "velha", Cwd: "/wt/a", TS: 10},
		{SessionID: "nova", Cwd: "/wt/a/sub", TS: 20},
		{SessionID: "fora", Cwd: "/wt/b", TS: 99},
	}
	got := MatchSession(sessions, wt)
	if got == nil || got.SessionID != "nova" {
		t.Errorf("MatchSession = %+v, esperado a mais recente (nova)", got)
	}
	if MatchSession(sessions, "/wt/zzz") != nil {
		t.Error("worktree sem sessão deveria dar nil")
	}
}

func TestParsePRs(t *testing.T) {
	merged := `[{"number": 11, "state": "MERGED", "mergedAt": "2026-07-17T00:00:00Z"}]`
	pr, err := ParsePRs([]byte(merged))
	if err != nil || pr == nil || pr.State != "MERGED" || pr.Number != 11 {
		t.Errorf("merged: %+v, %v", pr, err)
	}
	mixed := `[{"number": 3, "state": "CLOSED", "mergedAt": null},
		{"number": 5, "state": "OPEN", "mergedAt": null},
		{"number": 4, "state": "MERGED", "mergedAt": "x"}]`
	pr, err = ParsePRs([]byte(mixed))
	if err != nil || pr == nil || pr.Number != 5 {
		t.Errorf("OPEN deveria vencer: %+v, %v", pr, err)
	}
	pr, err = ParsePRs([]byte(`[]`))
	if err != nil || pr != nil {
		t.Errorf("lista vazia deveria dar nil sem erro: %+v, %v", pr, err)
	}
}
