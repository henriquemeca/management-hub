package ui

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/henriquemeca/management-hub/internal/collect"
)

// rowKind classifica uma linha das listas AGORA/PLANO.
type rowKind int

const (
	rowHeader  rowKind = iota // cabeçalho de seção/projeto (não selecionável)
	rowInfo                   // linha informativa (não selecionável)
	rowGate                   // ⛔ decisão pendente
	rowSession                // ⏸ sessão blocked/waiting
	rowShip                   // ⇧ PR merged, falta ship
	rowQueue                  // ○ fila pronta para spawn
	rowOrphan                 // ⚠ worktree sem janela
	rowIssue                  // issue do PLANO
)

// row é uma linha renderizável + o payload da ação primária.
type row struct {
	kind       rowKind
	selectable bool
	icon       string
	id         string // identificador curto exibido (KEY, gate id…)
	text       string
	meta       string // idade/badge, alinhado à direita

	key      string // issue key associada (quando derivável)
	url      string
	window   string // janela tmux para focar
	gate     *collect.Gate
	issue    *collect.Issue
	wt       *collect.Worktree
	queuePos int
}

// ---------- helpers de derivação ----------

// keyFromSlug deriva "HEN-85" de "hen-85-wos-doctor-…" (melhor esforço).
func keyFromSlug(slug string) string {
	parts := strings.SplitN(slug, "-", 3)
	if len(parts) >= 2 {
		if _, err := strconv.Atoi(parts[1]); err == nil {
			return strings.ToUpper(parts[0]) + "-" + parts[1]
		}
	}
	return ""
}

// slugForIssue segue a derivação do linear-sync/ship: última parte do
// branchName; fallback identifier minúsculo.
func slugForIssue(i collect.Issue) string {
	if i.BranchName != "" {
		parts := strings.Split(i.BranchName, "/")
		return parts[len(parts)-1]
	}
	return strings.ToLower(i.Identifier)
}

// wtForIssue acha o worktree da issue: slug igual ao derivado da branch, ou
// que começa com a key minúscula (workmux pode truncar o slug).
func wtForIssue(i collect.Issue, wts []collect.Worktree) *collect.Worktree {
	slug := slugForIssue(i)
	lower := strings.ToLower(i.Identifier)
	for idx := range wts {
		w := &wts[idx]
		if w.Slug == slug || w.Slug == lower || strings.HasPrefix(w.Slug, lower+"-") {
			return w
		}
	}
	return nil
}

func issueByKey(issues []collect.Issue, key string) *collect.Issue {
	for i := range issues {
		if issues[i].Identifier == key {
			return &issues[i]
		}
	}
	return nil
}

// gateMatchesKey: a task do gate contém a KEY como token (contrato da triage:
// task "triagem KEY"); gates legados têm task vazia e a KEY na pergunta —
// aceito como fallback, sempre com fronteira de palavra (HEN-1 ≠ HEN-17).
func gateMatchesKey(g collect.Gate, key string) bool {
	if key == "" {
		return false
	}
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(key) + `\b`)
	return re.MatchString(g.Task) || re.MatchString(g.Question)
}

func queuePos(q []collect.QueueItem, key string) int {
	for i := range q {
		if q[i].Key == key {
			return i + 1
		}
	}
	return 0
}

// prioRank ordena como a fila: urgent(1)→low(4), sem prioridade(0) por último.
func prioRank(p int) int {
	if p == 0 {
		return 5
	}
	return p
}

func prioLabel(p int) string {
	if p == 0 {
		return "—"
	}
	return "P" + strconv.Itoa(p)
}

func keyNumber(key string) int {
	if _, num, ok := strings.Cut(key, "-"); ok {
		if n, err := strconv.Atoi(num); err == nil {
			return n
		}
	}
	return 0
}

func ageStr(now time.Time, t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := now.Sub(t)
	switch {
	case d < 0:
		return "0s"
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func ageUnix(now time.Time, ts int64) string {
	if ts <= 0 {
		return ""
	}
	return ageStr(now, time.Unix(ts, 0))
}

func ageISO(now time.Time, iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return ""
	}
	return ageStr(now, t)
}

// gateKey extrai a issue key referenciada por um gate ("triagem HEN-17" ou
// "[HEN-17] …"), para triage --only e navegação.
var keyTokenRe = regexp.MustCompile(`\b[A-Z][A-Z0-9]*-\d+\b`)

func gateKey(g collect.Gate) string {
	if k := keyTokenRe.FindString(g.Task); k != "" {
		return k
	}
	return keyTokenRe.FindString(g.Question)
}

// ---------- AGORA ----------

// counts para o header: ⛔ ⏸ ⇧ ○.
type counts struct{ gates, paused, ship, queue int }

func deriveCounts(s collect.Snapshot) counts {
	c := counts{gates: len(s.Gates), queue: len(s.Queue)}
	for i := range s.Worktrees {
		wt := &s.Worktrees[i]
		if wt.Merged() {
			c.ship++
			continue
		}
		if wt.Session != nil && wt.Session.State != collect.StateWorking {
			c.paused++
		}
	}
	return c
}

// agoraRows monta a lista AGORA na ordem de atenção: ⛔ gates → ⏸ sessões
// paradas → ⇧ falta-ship → ○ fila → ⚠ órfãos. Categorias disjuntas: PR merged
// vence sessão parada; órfão é worktree sem janela e sem PR merged.
func agoraRows(s collect.Snapshot, now time.Time) []row {
	var rows []row
	section := func(label string) { rows = append(rows, row{kind: rowHeader, text: label}) }

	if len(s.Gates) > 0 {
		section(fmt.Sprintf("⛔ decisões pendentes (%d)", len(s.Gates)))
		for i := range s.Gates {
			g := &s.Gates[i]
			key := gateKey(*g)
			id := key
			if id == "" {
				id = g.ID
			}
			var issue *collect.Issue
			var url string
			if key != "" {
				if is := issueByKey(s.Issues, key); is != nil {
					issue = is
					url = is.URL
				}
			}
			rows = append(rows, row{
				kind: rowGate, selectable: true, icon: "⛔", id: id,
				text: strings.TrimSpace(g.Question), meta: ageUnix(now, g.CreatedAt),
				key: key, url: url, gate: g, issue: issue,
			})
		}
	}

	var paused, ship, orphans []*collect.Worktree
	for i := range s.Worktrees {
		wt := &s.Worktrees[i]
		switch {
		case wt.Merged():
			ship = append(ship, wt)
		case wt.Session != nil && wt.Session.State != collect.StateWorking:
			paused = append(paused, wt)
		case wt.Window == "":
			orphans = append(orphans, wt)
		}
	}

	if len(paused) > 0 {
		// blocked antes de waiting; mais antigo primeiro dentro do estado
		sort.SliceStable(paused, func(a, b int) bool {
			sa, sb := paused[a].Session, paused[b].Session
			if sa.State != sb.State {
				return sa.State == collect.StateBlocked
			}
			return sa.TS < sb.TS
		})
		section(fmt.Sprintf("⏸ sessões paradas (%d)", len(paused)))
		for _, wt := range paused {
			icon := "⏸"
			if wt.Session.State == collect.StateBlocked {
				icon = "⛔"
			}
			rows = append(rows, row{
				kind: rowSession, selectable: true, icon: icon, id: wtID(wt),
				text: fmt.Sprintf("%s · %s", wt.Session.State, wt.Slug),
				meta: ageUnix(now, wt.Session.TS),
				key:  keyFromSlug(wt.Slug), window: wt.Window, wt: wt,
				url: urlForSlug(s.Issues, wt.Slug),
			})
		}
	}

	if len(ship) > 0 {
		section(fmt.Sprintf("⇧ falta ship (%d)", len(ship)))
		for _, wt := range ship {
			pr := ""
			age := ""
			if wt.PR != nil {
				pr = fmt.Sprintf("PR #%d merged", wt.PR.Number)
				age = ageISO(now, wt.PR.MergedAt)
			}
			rows = append(rows, row{
				kind: rowShip, selectable: true, icon: "⇧", id: wtID(wt),
				text: strings.TrimSpace(pr + " · " + wt.Slug), meta: age,
				key: keyFromSlug(wt.Slug), window: wt.Window, wt: wt,
				url: urlForSlug(s.Issues, wt.Slug),
			})
		}
	}

	free := s.Cap - s.Occupied
	if free < 0 {
		free = 0
	}
	if len(s.Queue) > 0 {
		section(fmt.Sprintf("○ fila (%d · %d slot(s) livre(s))", len(s.Queue), free))
		for i := range s.Queue {
			q := &s.Queue[i]
			is := issueByKey(s.Issues, q.Key)
			var url string
			if is != nil {
				url = is.URL
			}
			rows = append(rows, row{
				kind: rowQueue, selectable: true, icon: "○", id: q.Key,
				text: q.Title, meta: fmt.Sprintf("#%d %s", i+1, q.PriorityLabel),
				key: q.Key, url: url, issue: is, queuePos: i + 1,
			})
		}
	}

	if len(orphans) > 0 {
		section(fmt.Sprintf("⚠ worktrees órfãos (%d)", len(orphans)))
		for _, wt := range orphans {
			rows = append(rows, row{
				kind: rowOrphan, selectable: true, icon: "⚠", id: wtID(wt),
				text: "sem janela tmux · " + wt.Path, meta: "",
				key: keyFromSlug(wt.Slug), wt: wt, url: urlForSlug(s.Issues, wt.Slug),
			})
		}
	}

	if len(rows) == 0 {
		rows = append(rows, row{kind: rowInfo, text: "tudo limpo — nada pedindo atenção agora"})
	}
	return rows
}

func wtID(wt *collect.Worktree) string {
	if k := keyFromSlug(wt.Slug); k != "" {
		return k
	}
	return wt.Slug
}

func urlForSlug(issues []collect.Issue, slug string) string {
	key := keyFromSlug(slug)
	if key == "" {
		return ""
	}
	if is := issueByKey(issues, key); is != nil {
		return is.URL
	}
	return ""
}

// ---------- PLANO ----------

// badgeForIssue deriva o badge de estágio do ciclo (ordem de precedência:
// falta-ship > rodando > pronta #N > em-interview > crua > estado literal).
func badgeForIssue(i collect.Issue, s collect.Snapshot) string {
	if wt := wtForIssue(i, s.Worktrees); wt != nil {
		if wt.Merged() {
			return "falta-ship"
		}
		if wt.Window != "" {
			state := "sem sessão"
			if wt.Session != nil {
				state = wt.Session.State
			}
			return "rodando·" + state
		}
	}
	if pos := queuePos(s.Queue, i.Identifier); pos > 0 {
		return fmt.Sprintf("pronta #%d", pos)
	}
	if i.State.Name == "Backlog" {
		for _, g := range s.Gates {
			if gateMatchesKey(g, i.Identifier) {
				return "em-interview"
			}
		}
		return "crua"
	}
	return strings.ReplaceAll(strings.ToLower(i.State.Name), " ", "-")
}

// planoRows agrupa o plano por projeto (sem projeto por último), ordenado por
// prioridade (urgent→low, sem prioridade por último) e número da key.
func planoRows(s collect.Snapshot, now time.Time) []row {
	groups := map[string][]collect.Issue{}
	var names []string
	for _, i := range s.Issues {
		name := i.ProjectName()
		if _, ok := groups[name]; !ok {
			names = append(names, name)
		}
		groups[name] = append(groups[name], i)
	}
	sort.Slice(names, func(a, b int) bool {
		if (names[a] == "") != (names[b] == "") {
			return names[b] == "" // sem projeto por último
		}
		return names[a] < names[b]
	})

	var rows []row
	for _, name := range names {
		issues := groups[name]
		sort.SliceStable(issues, func(a, b int) bool {
			ra, rb := prioRank(issues[a].Priority), prioRank(issues[b].Priority)
			if ra != rb {
				return ra < rb
			}
			return keyNumber(issues[a].Identifier) < keyNumber(issues[b].Identifier)
		})
		label := name
		if label == "" {
			label = "sem projeto"
		}
		rows = append(rows, row{kind: rowHeader, text: fmt.Sprintf("▸ %s (%d)", label, len(issues))})
		for idx := range issues {
			i := issues[idx]
			badge := badgeForIssue(i, s)
			icon := "·"
			switch {
			case badge == "falta-ship":
				icon = "⇧"
			case strings.HasPrefix(badge, "rodando"):
				icon = "●"
			case strings.HasPrefix(badge, "pronta"):
				icon = "○"
			case badge == "em-interview":
				icon = "⛔"
			}
			title := i.Title
			if i.Parent != nil {
				title = "↳ " + title
			}
			issueCopy := i
			r := row{
				kind: rowIssue, selectable: true, icon: icon, id: i.Identifier,
				text: fmt.Sprintf("%s %s", prioLabel(i.Priority), title),
				meta: badge + " " + ageISO(now, i.UpdatedAt),
				key:  i.Identifier, url: i.URL, issue: &issueCopy,
				queuePos: queuePos(s.Queue, i.Identifier),
			}
			if wt := wtForIssue(i, s.Worktrees); wt != nil {
				r.wt = wt
				r.window = wt.Window
			}
			rows = append(rows, r)
		}
	}
	if len(rows) == 0 {
		rows = append(rows, row{kind: rowInfo, text: "plano vazio (ou fonte fora do ar)"})
	}
	return rows
}
