// Package ui é a mesa única do work-os: renderiza o Snapshot e despacha ações
// nos comandos existentes. Nenhuma lógica de domínio vive aqui — só projeção
// (internal/collect) e despacho (internal/act).
package ui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/henriquemeca/management-hub/internal/act"
	"github.com/henriquemeca/management-hub/internal/collect"
)

const (
	refreshEvery = 20 * time.Second
	narrowWidth  = 110 // abaixo disso o detalhe vira tela cheia no enter
)

// mode é o sub-estado de interação da TUI.
type mode int

const (
	modeList mode = iota
	modeDetail
	modeGateInput
	modeShipConfirm
	modeSpawnPick
	modeCapture
)

// spawnProfile é uma entrada do picker de modelo/effort (vazio = default do
// linear-sync: trailer agent-config > política).
type spawnProfile struct {
	label, model, effort string
}

var spawnProfiles = []spawnProfile{
	{label: "default (trailer/política)"},
	{label: "sonnet · medium", model: "sonnet", effort: "medium"},
	{label: "sonnet · high", model: "sonnet", effort: "high"},
	{label: "opus · high", model: "opus", effort: "high"},
	{label: "haiku · low", model: "haiku", effort: "low"},
}

// Mensagens internas do loop.
type (
	snapshotMsg struct{ snap collect.Snapshot }
	tickMsg     time.Time
	detailMsg   struct {
		key    string
		detail *collect.IssueDetail
		err    error
	}
	actionMsg struct {
		res  act.Result
		hint string // dica pós-ação (ex.: "rode t para a próxima rodada")
	}
)

// Model é o estado completo da TUI (tea.Model por valor).
type Model struct {
	snap   collect.Snapshot
	rows   []row
	cursor int
	tab    int // 0 = AGORA, 1 = PLANO

	width, height int
	mode          mode
	collecting    bool
	busy          string // rótulo de ação em andamento (status bar)
	status        string
	statusErr     bool
	lastRefresh   time.Time

	spin     spinner.Model
	detailVP viewport.Model
	detail   *collect.IssueDetail // detalhe buscado (enter em issue)
	detRow   row                  // linha que originou o detalhe

	input    textinput.Model // resposta de gate
	capTitle textinput.Model
	capDesc  textinput.Model
	capFocus int    // 0 = título, 1 = descrição
	capDir   string // cwd do work (repo do projeto selecionado)
	pickIdx  int
	pickRow  row // linha alvo do picker/confirmação

	smoke bool
}

// New cria o Model inicial (uso interativo).
func New() Model {
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = stBlue

	in := textinput.New()
	in.Placeholder = "resposta… (enter envia, esc cancela)"
	in.CharLimit = 0

	ct := textinput.New()
	ct.Placeholder = "título da captura"
	ct.CharLimit = 0
	cd := textinput.New()
	cd.Placeholder = "descrição (opcional)"
	cd.CharLimit = 0

	return Model{
		spin:     sp,
		input:    in,
		capTitle: ct,
		capDesc:  cd,
		detailVP: viewport.New(0, 0),
		// a primeira coleta dispara no Init — nasce coletando para o header
		// mostrar o spinner desde o primeiro frame
		collecting: true,
	}
}

// NewSmoke monta um Model já com snapshot e tamanho fixo — um frame headless.
func NewSmoke(snap collect.Snapshot, width, height int) Model {
	m := New()
	m.smoke = true
	m.width, m.height = width, height
	m.applySnapshot(snap)
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(collectCmd(), tickCmd(), m.spin.Tick)
}

func collectCmd() tea.Cmd {
	return func() tea.Msg {
		snap := collect.Collect(context.Background(), collect.Options{})
		return snapshotMsg{snap: snap}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshEvery, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func detailCmd(r row) tea.Cmd {
	key := r.key
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		d, err := collect.FetchIssueDetail(ctx, key)
		return detailMsg{key: key, detail: d, err: err}
	}
}

func actionCmd(hint string, fn func() act.Result) tea.Cmd {
	return func() tea.Msg { return actionMsg{res: fn(), hint: hint} }
}

// applySnapshot troca a foto e rederiva as linhas mantendo a seleção estável.
func (m *Model) applySnapshot(snap collect.Snapshot) {
	prevID := ""
	if r := m.selected(); r != nil {
		prevID = r.id
	}
	m.snap = snap
	m.collecting = false
	m.lastRefresh = snap.CollectedAt
	m.rebuildRows()
	if prevID != "" {
		for i, r := range m.rows {
			if r.selectable && r.id == prevID {
				m.cursor = i
				break
			}
		}
	}
	m.clampCursor()
}

func (m *Model) rebuildRows() {
	now := time.Now()
	if m.tab == 0 {
		m.rows = agoraRows(m.snap, now)
	} else {
		m.rows = planoRows(m.snap, now)
	}
	m.clampCursor()
}

// clampCursor garante cursor sobre linha selecionável (headers são pulados).
func (m *Model) clampCursor() {
	if len(m.rows) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.rows[m.cursor].selectable {
		return
	}
	for i := m.cursor; i < len(m.rows); i++ {
		if m.rows[i].selectable {
			m.cursor = i
			return
		}
	}
	for i := m.cursor; i >= 0; i-- {
		if m.rows[i].selectable {
			m.cursor = i
			return
		}
	}
}

func (m *Model) move(delta int) {
	i := m.cursor
	for {
		i += delta
		if i < 0 || i >= len(m.rows) {
			return
		}
		if m.rows[i].selectable {
			m.cursor = i
			return
		}
	}
}

func (m *Model) selected() *row {
	if m.cursor >= 0 && m.cursor < len(m.rows) && m.rows[m.cursor].selectable {
		return &m.rows[m.cursor]
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.sizeViewport()
		return m, nil

	case tickMsg:
		if m.collecting {
			return m, tickCmd()
		}
		m.collecting = true
		return m, tea.Batch(collectCmd(), tickCmd(), m.spin.Tick)

	case snapshotMsg:
		m.applySnapshot(msg.snap)
		m.sizeViewport()
		return m, nil

	case detailMsg:
		if msg.err != nil {
			m.setStatus("✗ detalhe "+msg.key+" — "+msg.err.Error(), true)
			return m, nil
		}
		m.detail = msg.detail
		m.mode = modeDetail
		m.sizeViewport()
		m.detailVP.SetContent(m.detailContent())
		m.detailVP.GotoTop()
		return m, nil

	case actionMsg:
		m.busy = ""
		summary := msg.res.Summary()
		if msg.res.OK() && msg.hint != "" {
			summary += " · " + msg.hint
		}
		m.setStatus(summary, !msg.res.OK())
		// toda ação dispara refresh
		if !m.collecting {
			m.collecting = true
			return m, tea.Batch(collectCmd(), m.spin.Tick)
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		if m.collecting || m.busy != "" {
			return m, cmd
		}
		return m, nil // spinner parado quando não há trabalho

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) setStatus(s string, isErr bool) {
	m.status = s
	m.statusErr = isErr
}

func (m *Model) startBusy(label string) tea.Cmd {
	m.busy = label
	return m.spin.Tick
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.mode {
	case modeGateInput:
		switch key {
		case "esc":
			m.mode = modeList
			return m, nil
		case "enter":
			text := m.input.Value()
			if text == "" {
				return m, nil
			}
			g := m.pickRow.gate
			if g == nil {
				m.mode = modeList
				return m, nil
			}
			id := g.ID
			m.mode = modeList
			cmd := m.startBusy("gate " + id)
			return m, tea.Batch(cmd, actionCmd("rode t para a próxima rodada",
				func() act.Result { return act.ResolveGate(id, text) }))
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd

	case modeShipConfirm:
		switch key {
		case "enter", "y":
			issueKey := m.pickRow.key
			m.mode = modeList
			if issueKey == "" {
				m.setStatus("✗ ship — sem issue key derivável", true)
				return m, nil
			}
			cmd := m.startBusy("ship " + issueKey)
			return m, tea.Batch(cmd, actionCmd("", func() act.Result { return act.Ship(issueKey) }))
		case "esc", "n", "q":
			m.mode = modeList
			return m, nil
		}
		return m, nil

	case modeSpawnPick:
		switch key {
		case "esc", "q":
			m.mode = modeList
			return m, nil
		case "up", "k":
			if m.pickIdx > 0 {
				m.pickIdx--
			}
			return m, nil
		case "down", "j":
			if m.pickIdx < len(spawnProfiles)-1 {
				m.pickIdx++
			}
			return m, nil
		case "enter":
			p := spawnProfiles[m.pickIdx]
			issueKey := m.pickRow.key
			m.mode = modeList
			if issueKey == "" {
				return m, nil
			}
			cmd := m.startBusy("spawn " + issueKey)
			return m, tea.Batch(cmd, actionCmd("",
				func() act.Result { return act.Spawn(issueKey, p.model, p.effort) }))
		}
		return m, nil

	case modeCapture:
		switch key {
		case "esc":
			m.mode = modeList
			return m, nil
		case "tab", "shift+tab":
			m.capFocus = 1 - m.capFocus
			if m.capFocus == 0 {
				m.capTitle.Focus()
				m.capDesc.Blur()
			} else {
				m.capDesc.Focus()
				m.capTitle.Blur()
			}
			return m, nil
		case "enter":
			if m.capFocus == 0 { // enter no título → desce para a descrição
				m.capFocus = 1
				m.capDesc.Focus()
				m.capTitle.Blur()
				return m, nil
			}
			title := m.capTitle.Value()
			desc := m.capDesc.Value()
			dir := m.capDir
			if title == "" {
				m.setStatus("✗ capturar — título vazio", true)
				return m, nil
			}
			m.mode = modeList
			cmd := m.startBusy("capturar")
			return m, tea.Batch(cmd, actionCmd("",
				func() act.Result { return act.Capture(title, desc, dir) }))
		}
		var cmd tea.Cmd
		if m.capFocus == 0 {
			m.capTitle, cmd = m.capTitle.Update(msg)
		} else {
			m.capDesc, cmd = m.capDesc.Update(msg)
		}
		return m, cmd

	case modeDetail:
		switch key {
		case "esc", "q":
			m.mode = modeList
			m.detail = nil
			return m, nil
		case "o", "t", "s", "S", "g":
			return m.actionKey(key, m.detRow)
		}
		var cmd tea.Cmd
		m.detailVP, cmd = m.detailVP.Update(msg)
		return m, cmd
	}

	// modeList
	switch key {
	case "q":
		return m, tea.Quit
	case "tab":
		m.tab = 1 - m.tab
		m.cursor = 0
		m.rebuildRows()
		return m, nil
	case "down", "j":
		m.move(1)
		return m, nil
	case "up", "k":
		m.move(-1)
		return m, nil
	case "r":
		if m.collecting {
			return m, nil
		}
		m.collecting = true
		return m, tea.Batch(collectCmd(), m.spin.Tick)
	case "n":
		m.mode = modeCapture
		m.capTitle.SetValue("")
		m.capDesc.SetValue("")
		m.capFocus = 0
		m.capTitle.Width = m.modalInnerWidth()
		m.capDesc.Width = m.modalInnerWidth()
		m.capTitle.Focus()
		m.capDesc.Blur()
		m.capDir = ""
		if r := m.selected(); r != nil && r.issue != nil {
			m.capDir = act.RepoDir(r.issue.Description)
		}
		return m, textinput.Blink
	case "enter":
		r := m.selected()
		if r == nil {
			return m, nil
		}
		return m.primaryAction(*r)
	case "o", "t", "s", "S", "g":
		r := m.selected()
		if r == nil && key != "t" {
			return m, nil
		}
		var rr row
		if r != nil {
			rr = *r
		}
		return m.actionKey(key, rr)
	}
	return m, nil
}

// primaryAction é o enter por tipo de linha.
func (m Model) primaryAction(r row) (tea.Model, tea.Cmd) {
	switch r.kind {
	case rowGate:
		m.mode = modeGateInput
		m.pickRow = r
		m.input.SetValue("")
		m.input.Width = m.modalInnerWidth()
		m.input.Focus()
		return m, textinput.Blink
	case rowShip:
		m.mode = modeShipConfirm
		m.pickRow = r
		return m, nil
	case rowQueue:
		m.mode = modeSpawnPick
		m.pickRow = r
		m.pickIdx = 0
		return m, nil
	case rowSession:
		if r.window != "" {
			cmd := m.startBusy("focar " + r.window)
			return m, tea.Batch(cmd, actionCmd("",
				func() act.Result { return act.FocusWindow(r.window) }))
		}
		return m, nil
	case rowIssue:
		m.detRow = r
		m.detail = nil
		cmd := m.startBusy("detalhe " + r.key)
		return m, tea.Batch(cmd, detailCmd(r))
	case rowOrphan:
		m.detRow = r
		m.detail = nil
		m.mode = modeDetail
		m.sizeViewport()
		m.detailVP.SetContent(m.detailContent())
		m.detailVP.GotoTop()
		return m, nil
	}
	return m, nil
}

// actionKey despacha as teclas de ação (o/t/s/S/g) sobre uma linha.
func (m Model) actionKey(key string, r row) (tea.Model, tea.Cmd) {
	switch key {
	case "o":
		if r.url == "" {
			m.setStatus("✗ abrir — sem URL para esta linha", true)
			return m, nil
		}
		url := r.url
		return m, actionCmd("", func() act.Result { return act.OpenURL(url) })
	case "t":
		only := r.key
		label := "triage"
		if only != "" {
			label += " " + only
		}
		cmd := m.startBusy(label)
		return m, tea.Batch(cmd, actionCmd("",
			func() act.Result { return act.Triage(only) }))
	case "s":
		if r.key == "" {
			m.setStatus("✗ spawn — selecione uma issue", true)
			return m, nil
		}
		issueKey := r.key
		cmd := m.startBusy("spawn " + issueKey)
		return m, tea.Batch(cmd, actionCmd("",
			func() act.Result { return act.Spawn(issueKey, "", "") }))
	case "S":
		if r.key == "" {
			m.setStatus("✗ ship — selecione uma issue", true)
			return m, nil
		}
		m.mode = modeShipConfirm
		m.pickRow = r
		return m, nil
	case "g":
		if r.window == "" {
			m.setStatus("✗ focar — linha sem janela tmux", true)
			return m, nil
		}
		win := r.window
		cmd := m.startBusy("focar " + win)
		return m, tea.Batch(cmd, actionCmd("",
			func() act.Result { return act.FocusWindow(win) }))
	}
	return m, nil
}
