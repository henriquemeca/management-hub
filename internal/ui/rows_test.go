package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/henriquemeca/management-hub/internal/collect"
)

func issue(key, state, project string, prio int, branch string) collect.Issue {
	var i collect.Issue
	i.Identifier = key
	i.Title = "título " + key
	i.State.Name = state
	i.Priority = prio
	i.BranchName = branch
	i.URL = "https://linear.app/x/" + key
	if project != "" {
		i.Project = &struct {
			Name string `json:"name"`
		}{Name: project}
	}
	return i
}

func snapFixture() collect.Snapshot {
	sess := collect.Session{SessionID: "s1", State: collect.StateWaiting, Cwd: "/wt/mh/hen-85-doctor", TS: 100}
	return collect.Snapshot{
		Queue: []collect.QueueItem{{Key: "HEN-58", Priority: 3, PriorityLabel: "P3", Title: "captura"}},
		Issues: []collect.Issue{
			issue("HEN-58", "Todo", "management-hub", 3, "u/hen-58-captura"),
			issue("HEN-17", "Backlog", "dotfiles", 2, "u/hen-17-matriz"),
			issue("HEN-84", "Backlog", "management-hub", 3, "u/hen-84-plan-store"),
			issue("HEN-85", "In Progress", "management-hub", 1, "u/hen-85-doctor"),
			issue("HEN-90", "In Progress", "management-hub", 2, "u/hen-90-merged"),
		},
		Gates: []collect.Gate{
			{ID: "g1", Task: "", Question: "[HEN-17] repo alvo?", Status: "pending", CreatedAt: 50},
		},
		Worktrees: []collect.Worktree{
			{Slug: "hen-85-doctor", Path: "/wt/mh/hen-85-doctor", Window: "wm-hen-85-doctor", Session: &sess},
			{Slug: "hen-90-merged", Path: "/wt/mh/hen-90-merged", Window: "wm-hen-90-merged",
				PR: &collect.PR{Number: 12, State: "MERGED", MergedAt: "2026-07-17T00:00:00Z"}},
			{Slug: "hen-99-orfao", Path: "/wt/mh/hen-99-orfao"}, // sem janela
		},
		Cap: 2, Occupied: 2,
	}
}

func kindsOf(rows []row) []rowKind {
	var ks []rowKind
	for _, r := range rows {
		if r.selectable {
			ks = append(ks, r.kind)
		}
	}
	return ks
}

func TestAgoraRowsOrderAndCategories(t *testing.T) {
	rows := agoraRows(snapFixture(), time.Unix(200, 0))
	got := kindsOf(rows)
	want := []rowKind{rowGate, rowSession, rowShip, rowQueue, rowOrphan}
	if len(got) != len(want) {
		t.Fatalf("linhas selecionáveis = %d (%v), esperado %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("posição %d: kind %v, esperado %v", i, got[i], want[i])
		}
	}
	// gate herda a issue key e a fila traz posição
	for _, r := range rows {
		switch r.kind {
		case rowGate:
			if r.key != "HEN-17" {
				t.Errorf("gate key = %q, esperado HEN-17", r.key)
			}
		case rowQueue:
			if r.queuePos != 1 || r.key != "HEN-58" {
				t.Errorf("fila: %+v", r)
			}
		case rowShip:
			if r.key != "HEN-90" {
				t.Errorf("ship key = %q, esperado HEN-90", r.key)
			}
		}
	}
}

func TestBadgeDerivation(t *testing.T) {
	s := snapFixture()
	cases := map[string]string{
		"HEN-58": "pronta #1",       // Todo na fila
		"HEN-17": "em-interview",    // Backlog + gate pendente com a KEY
		"HEN-84": "crua",            // Backlog sem gate
		"HEN-85": "rodando·waiting", // janela wm- + sessão waiting
		"HEN-90": "falta-ship",      // PR merged vence rodando
	}
	for key, want := range cases {
		is := issueByKey(s.Issues, key)
		if is == nil {
			t.Fatalf("fixture sem %s", key)
		}
		if got := badgeForIssue(*is, s); got != want {
			t.Errorf("badge de %s = %q, esperado %q", key, got, want)
		}
	}
}

func TestBadgeKeyBoundary(t *testing.T) {
	// HEN-1 não pode casar com o gate de HEN-17
	s := snapFixture()
	i := issue("HEN-1", "Backlog", "", 0, "u/hen-1-x")
	if got := badgeForIssue(i, s); got != "crua" {
		t.Errorf("HEN-1 = %q, esperado crua (fronteira de palavra)", got)
	}
}

func TestPlanoRowsGrouping(t *testing.T) {
	rows := planoRows(snapFixture(), time.Unix(200, 0))
	var headers []string
	for _, r := range rows {
		if r.kind == rowHeader {
			headers = append(headers, r.text)
		}
	}
	if len(headers) != 2 {
		t.Fatalf("projetos = %v, esperado 2 grupos", headers)
	}
	if !strings.Contains(headers[0], "dotfiles") || !strings.Contains(headers[1], "management-hub") {
		t.Errorf("ordem dos grupos: %v", headers)
	}
	// dentro de management-hub: prioridade urgent(1) primeiro
	var mh []string
	inMH := false
	for _, r := range rows {
		if r.kind == rowHeader {
			inMH = strings.Contains(r.text, "management-hub")
			continue
		}
		if inMH && r.selectable {
			mh = append(mh, r.id)
		}
	}
	want := []string{"HEN-85", "HEN-90", "HEN-58", "HEN-84"}
	if strings.Join(mh, " ") != strings.Join(want, " ") {
		t.Errorf("ordem em management-hub: %v, esperado %v", mh, want)
	}
}

func TestKeyFromSlug(t *testing.T) {
	cases := map[string]string{
		"hen-85-wos-doctor": "HEN-85",
		"hen-85":            "HEN-85",
		"sem-numero-aqui":   "",
		"solto":             "",
	}
	for slug, want := range cases {
		if got := keyFromSlug(slug); got != want {
			t.Errorf("keyFromSlug(%q) = %q, esperado %q", slug, got, want)
		}
	}
}
