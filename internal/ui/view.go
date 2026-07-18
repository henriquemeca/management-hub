package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/henriquemeca/management-hub/internal/collect"
)

// chrome = header + tabs + keybar + status (linhas fora do corpo).
const chromeLines = 4

func (m Model) bodyHeight() int {
	h := m.height - chromeLines
	if h < 3 {
		h = 3
	}
	return h
}

func (m Model) narrow() bool { return m.width < narrowWidth }

func (m Model) listPaneWidth() int {
	if m.narrow() {
		return m.width
	}
	w := m.width * 5 / 11
	if w < 40 {
		w = 40
	}
	if w > 72 {
		w = 72
	}
	return w
}

// sizeViewport ajusta o viewport de detalhe ao layout corrente.
func (m *Model) sizeViewport() {
	var w int
	if m.narrow() {
		w = m.width - 4
	} else {
		w = m.width - m.listPaneWidth() - 4
	}
	if w < 10 {
		w = 10
	}
	h := m.bodyHeight() - 2
	if h < 1 {
		h = 1
	}
	m.detailVP.Width = w
	m.detailVP.Height = h
	if m.mode == modeDetail {
		m.detailVP.SetContent(m.detailContent())
	}
}

func (m Model) View() string {
	if m.width <= 0 {
		return "carregando…"
	}
	sections := []string{
		m.headerView(),
		m.tabsView(),
		m.bodyView(),
		m.keybarView(),
		m.statusView(),
	}
	return strings.Join(sections, "\n")
}

// ---------- header / tabs ----------

func (m Model) headerView() string {
	c := deriveCounts(m.snap)
	sep := stDim.Render(" · ")
	parts := []string{
		stTitle.Render("WOS"),
		sep,
		stRed.Render(fmt.Sprintf("⛔%d", c.gates)),
		" ",
		stOrange.Render(fmt.Sprintf("⏸%d", c.paused)),
		" ",
		stGreen.Render(fmt.Sprintf("⇧%d", c.ship)),
		" ",
		stBlue.Render(fmt.Sprintf("○%d", c.queue)),
		sep,
		fmt.Sprintf("slots %d/%d", m.snap.Occupied, m.snap.Cap),
		sep,
	}
	if m.lastRefresh.IsZero() {
		parts = append(parts, stDim.Render("coletando…"))
	} else {
		parts = append(parts, stDim.Render(m.lastRefresh.Format("15:04:05")))
	}
	if m.collecting {
		parts = append(parts, " ", m.spin.View())
	}
	return truncateANSI(strings.Join(parts, ""), m.width)
}

func (m Model) tabsView() string {
	agora, plano := stTab, stTab
	if m.tab == 0 {
		agora = stTabActive
	} else {
		plano = stTabActive
	}
	return agora.Render("AGORA") + " " + plano.Render("PLANO")
}

// ---------- corpo ----------

func (m Model) bodyView() string {
	bodyH := m.bodyHeight()

	switch m.mode {
	case modeGateInput, modeShipConfirm, modeSpawnPick, modeCapture:
		return lipgloss.Place(m.width, bodyH, lipgloss.Center, lipgloss.Center, m.modalView())
	case modeDetail:
		detail := stPane.Width(m.detailVP.Width + 2).Height(bodyH - 2).Render(m.detailVP.View())
		if m.narrow() {
			return detail
		}
		return lipgloss.JoinHorizontal(lipgloss.Top, m.listPane(m.listPaneWidth(), bodyH), detail)
	}

	// modeList
	if m.narrow() {
		return m.listPane(m.width, bodyH)
	}
	listW := m.listPaneWidth()
	detW := m.width - listW
	detail := stPane.Width(detW - 2).Height(bodyH - 2).
		Render(m.passiveDetail(detW-4, bodyH-2))
	return lipgloss.JoinHorizontal(lipgloss.Top, m.listPane(listW, bodyH), detail)
}

// listPane renderiza a lista com janela de rolagem em torno do cursor.
func (m Model) listPane(w, h int) string {
	contentW := w - 4
	contentH := h - 2
	if contentW < 10 {
		contentW = 10
	}
	if contentH < 1 {
		contentH = 1
	}

	offset := 0
	if len(m.rows) > contentH {
		offset = m.cursor - contentH/2
		if offset < 0 {
			offset = 0
		}
		if offset > len(m.rows)-contentH {
			offset = len(m.rows) - contentH
		}
	}

	var lines []string
	for i := offset; i < len(m.rows) && len(lines) < contentH; i++ {
		lines = append(lines, m.renderRow(m.rows[i], i == m.cursor, contentW))
	}
	return stPane.Width(w - 2).Height(contentH).Render(strings.Join(lines, "\n"))
}

func (m Model) renderRow(r row, selected bool, w int) string {
	switch r.kind {
	case rowHeader:
		return stHeaderRow.Render(truncate(r.text, w))
	case rowInfo:
		return stDim.Render(truncate(r.text, w))
	}

	innerW := w - 2 // gutter do cursor à esquerda
	if innerW < 8 {
		innerW = 8
	}
	meta := r.meta
	left := strings.TrimSpace(r.icon + " " + r.id)
	if r.text != "" {
		left += "  " + r.text
	}
	avail := innerW
	if meta != "" {
		avail = innerW - lipgloss.Width(meta) - 1
	}
	left = truncate(left, avail)
	pad := innerW - lipgloss.Width(left) - lipgloss.Width(meta)
	if pad < 1 {
		pad = 1
	}
	raw := left
	if meta != "" {
		raw += strings.Repeat(" ", pad) + meta
	}

	if selected {
		return stSelected.Render(truncate("▸ "+raw, w))
	}
	// colore só o ícone para manter a linha discreta
	icon := colorFor(r.kind).Render(r.icon)
	rest := strings.TrimPrefix(raw, r.icon)
	if meta != "" {
		rest = strings.TrimSuffix(rest, meta) + stDim.Render(meta)
	}
	return "  " + icon + rest
}

// passiveDetail é o painel direito no modo lista: resumo do item selecionado.
func (m Model) passiveDetail(w, h int) string {
	r := m.selected()
	if r == nil {
		return stDim.Render("nada selecionado")
	}
	content := detailFor(*r, nil, w)
	lines := strings.Split(content, "\n")
	if len(lines) > h {
		lines = append(lines[:h-1], stDim.Render("… (enter para o detalhe completo)"))
	}
	return strings.Join(lines, "\n")
}

// detailContent monta o conteúdo do viewport no modo detalhe.
func (m Model) detailContent() string {
	return detailFor(m.detRow, m.detail, m.detailVP.Width)
}

// detailFor renderiza o detalhe de uma linha; det ≠ nil = detalhe completo
// buscado via linear-gql issue (comments/children).
func detailFor(r row, det *collect.IssueDetail, w int) string {
	wrap := lipgloss.NewStyle().Width(w)
	var b strings.Builder
	line := func(s string) { b.WriteString(s + "\n") }

	switch r.kind {
	case rowGate:
		g := r.gate
		line(stTitle.Render("⛔ decisão pendente") + stDim.Render("  gate "+g.ID))
		if g.Task != "" {
			line(stDim.Render("task: " + g.Task))
		}
		line("")
		line(wrap.Render(g.Question))
		line("")
		line(stDim.Render("enter responde — a resposta vira comentário na issue; depois rode t para a próxima rodada"))
		return b.String()

	case rowSession, rowShip, rowOrphan:
		wt := r.wt
		title := map[rowKind]string{
			rowSession: "⏸ sessão parada",
			rowShip:    "⇧ falta ship",
			rowOrphan:  "⚠ worktree órfão",
		}[r.kind]
		line(stTitle.Render(title) + "  " + r.id)
		line("")
		if wt != nil {
			line(stDim.Render("worktree: ") + wt.Path)
			if wt.Window != "" {
				line(stDim.Render("janela:   ") + wt.Window)
			}
			if wt.Branch != "" {
				line(stDim.Render("branch:   ") + wt.Branch)
			}
			if wt.PR != nil {
				line(stDim.Render("PR:       ") + fmt.Sprintf("#%d %s", wt.PR.Number, wt.PR.State))
			}
			if wt.Session != nil {
				line(stDim.Render("sessão:   ") + wt.Session.State +
					stDim.Render("  (pid "+fmt.Sprint(wt.Session.PID)+")"))
			}
		}
		switch r.kind {
		case rowShip:
			line("")
			line(stDim.Render("enter confirma o ship (Done + remoção da worktree)"))
		case rowSession:
			line("")
			line(stDim.Render("enter foca a janela tmux do agente"))
		}
		return b.String()
	}

	// issue (PLANO, fila) — do snapshot ou do detalhe buscado
	var is *collect.Issue
	if det != nil {
		is = &det.Issue
	} else if r.issue != nil {
		is = r.issue
	}
	if is == nil {
		line(stTitle.Render(r.id) + "  " + wrap.Render(r.text))
		line("")
		line(stDim.Render("sem detalhe local — issue fora do plano aberto (Done?)"))
		return b.String()
	}

	line(stTitle.Render(is.Identifier) + "  " + wrap.Render(is.Title))
	meta := []string{is.State.Name, prioLabel(is.Priority)}
	if p := is.ProjectName(); p != "" {
		meta = append(meta, p)
	}
	if is.Parent != nil {
		meta = append(meta, "filha de "+is.Parent.Identifier)
	}
	line(stDim.Render(strings.Join(meta, " · ")))
	if r.wt != nil {
		wt := r.wt
		info := "worktree " + wt.Slug
		if wt.Branch != "" {
			info += " · " + wt.Branch
		}
		if wt.PR != nil {
			info += fmt.Sprintf(" · PR #%d %s", wt.PR.Number, wt.PR.State)
		}
		line(stDim.Render(info))
	} else if is.BranchName != "" {
		line(stDim.Render("branch " + is.BranchName))
	}
	line("")
	if strings.TrimSpace(is.Description) != "" {
		line(wrap.Render(strings.TrimSpace(is.Description)))
	} else {
		line(stDim.Render("(sem descrição)"))
	}

	if det != nil {
		comments := det.Comments.Nodes
		if len(comments) > 5 {
			comments = comments[len(comments)-5:]
		}
		if len(comments) > 0 {
			line("")
			line(stHeaderRow.Render(fmt.Sprintf("— últimos comentários (%d) —", len(comments))))
			for _, c := range comments {
				when := c.CreatedAt
				if t, err := time.Parse(time.RFC3339, c.CreatedAt); err == nil {
					when = t.Format("2006-01-02 15:04")
				}
				line("")
				line(stDim.Render("· " + when))
				line(wrap.Render(strings.TrimSpace(c.Body)))
			}
		}
		if n := len(det.Children.Nodes); n > 0 {
			line("")
			line(stHeaderRow.Render(fmt.Sprintf("— sub-issues (%d) —", n)))
			for _, ch := range det.Children.Nodes {
				line("  " + ch.Identifier + " " + stDim.Render("("+ch.State.Name+")") + " " + truncate(ch.Title, w-20))
			}
		}
	} else if r.kind == rowIssue {
		line("")
		line(stDim.Render("enter busca comentários e sub-issues"))
	}
	return b.String()
}

// ---------- modais ----------

// modalWidth/modalInnerWidth: largura do modal e da área útil interna.
func (m Model) modalWidth() int {
	w := m.width - 8
	if w > 80 {
		w = 80
	}
	if w < 30 {
		w = 30
	}
	return w
}

func (m Model) modalInnerWidth() int { return m.modalWidth() - 6 }

func (m Model) modalView() string {
	w := m.modalWidth()
	inner := w - 4
	wrap := lipgloss.NewStyle().Width(inner)
	var b strings.Builder

	switch m.mode {
	case modeGateInput:
		b.WriteString(stTitle.Render("responder gate "+m.pickRow.id) + "\n\n")
		if m.pickRow.gate != nil {
			b.WriteString(wrap.Render(m.pickRow.gate.Question) + "\n\n")
		}
		b.WriteString(m.input.View())
	case modeShipConfirm:
		b.WriteString(stTitle.Render("ship "+m.pickRow.key+"?") + "\n\n")
		if wt := m.pickRow.wt; wt != nil {
			b.WriteString(wrap.Render("worktree "+wt.Slug) + "\n")
			if wt.PR != nil {
				b.WriteString(wrap.Render(fmt.Sprintf("PR #%d %s", wt.PR.Number, wt.PR.State)) + "\n")
			}
			b.WriteString("\n")
		}
		b.WriteString(wrap.Render("marca Done e remove a worktree (guard-rails do ship valem)") + "\n\n")
		b.WriteString(stGreen.Render("enter/y confirma") + stDim.Render(" · esc cancela"))
	case modeSpawnPick:
		b.WriteString(stTitle.Render("spawn "+m.pickRow.key) + stDim.Render("  modelo · effort") + "\n\n")
		for i, p := range spawnProfiles {
			cursor := "  "
			label := p.label
			if i == m.pickIdx {
				cursor = stBlue.Render("▸ ")
				label = stTitle.Render(label)
			}
			b.WriteString(cursor + label + "\n")
		}
		b.WriteString("\n" + stDim.Render("enter spawna · esc cancela"))
	case modeCapture:
		b.WriteString(stTitle.Render("capturar ideia") + stDim.Render("  → work (Backlog)") + "\n\n")
		b.WriteString(m.capTitle.View() + "\n")
		b.WriteString(m.capDesc.View() + "\n\n")
		if m.capDir != "" {
			b.WriteString(stDim.Render("repo: "+m.capDir) + "\n")
		}
		b.WriteString(stDim.Render("enter avança/salva · tab campo · esc cancela"))
	}
	return stPane.Width(w - 2).Render(b.String())
}

// ---------- keybar / status ----------

func (m Model) keybarView() string {
	var keys string
	switch m.mode {
	case modeGateInput:
		keys = "enter responder · esc cancelar"
	case modeShipConfirm:
		keys = "enter/y confirmar · esc cancelar"
	case modeSpawnPick:
		keys = "↑↓ perfil · enter spawnar · esc cancelar"
	case modeCapture:
		keys = "enter avançar/salvar · tab campo · esc cancelar"
	case modeDetail:
		keys = "esc voltar · ↑↓ rolar · o abrir · t triage · s spawn · S ship"
	default:
		primary := "detalhe"
		if r := m.selected(); r != nil {
			switch r.kind {
			case rowGate:
				primary = "responder"
			case rowShip:
				primary = "ship"
			case rowQueue:
				primary = "spawn (perfil)"
			case rowSession:
				primary = "focar"
			}
		}
		other := "PLANO"
		if m.tab == 1 {
			other = "AGORA"
		}
		keys = fmt.Sprintf("enter %s · tab %s · n capturar · t triage · s spawn · S ship · g focar · o abrir · r atualizar · q sair",
			primary, other)
	}
	return truncateANSI(stDim.Render(keys), m.width)
}

func (m Model) statusView() string {
	var parts []string
	if m.busy != "" {
		parts = append(parts, m.spin.View()+" "+m.busy+"…")
	}
	if m.status != "" {
		if m.statusErr {
			parts = append(parts, stStatusErr.Render(m.status))
		} else {
			parts = append(parts, stStatusOK.Render(m.status))
		}
	}
	if len(m.snap.Warnings) > 0 {
		parts = append(parts, stYellow.Render("⚠ "+strings.Join(m.snap.Warnings, " · ")))
	}
	if len(parts) == 0 {
		parts = append(parts, stDim.Render("pronto"))
	}
	return truncateANSI(strings.Join(parts, stDim.Render(" · ")), m.width)
}

// truncateANSI corta uma linha já estilizada pela largura de célula, sem
// quebrar sequências ANSI no meio (corte por segmento estilizado).
func truncateANSI(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	// caminho simples: re-renderiza sem estilo se estourar (raro: terminal
	// muito estreito) — perder cor é melhor que quebrar o layout.
	plain := stripANSI(s)
	return truncate(plain, w)
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case inEsc:
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
		case r == '\x1b':
			inEsc = true
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
