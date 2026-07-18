package ui

import (
	"strings"
	"testing"
)

// Renderização headless: nenhum frame pode entrar em pânico nem estourar a
// largura do terminal, nas duas abas e nos dois layouts (largo/estreito).
func TestViewRendersBothTabsAndLayouts(t *testing.T) {
	for _, size := range []struct{ w, h int }{{120, 35}, {100, 30}} {
		m := NewSmoke(snapFixture(), size.w, size.h)
		for tab := 0; tab < 2; tab++ {
			m.tab = tab
			m.rebuildRows()
			frame := m.View()
			if frame == "" {
				t.Fatalf("frame vazio (tab %d, %dx%d)", tab, size.w, size.h)
			}
			for n, line := range strings.Split(frame, "\n") {
				if got := visibleWidth(line); got > size.w {
					t.Errorf("tab %d %dx%d: linha %d com %d colunas (máx %d): %q",
						tab, size.w, size.h, n, got, size.w, stripANSI(line))
				}
			}
		}
	}
}

func TestViewShowsSectionsFromFixture(t *testing.T) {
	m := NewSmoke(snapFixture(), 120, 35)
	frame := stripANSI(m.View())
	for _, want := range []string{"WOS", "⛔1", "⏸1", "⇧1", "○1", "slots 2/2", "decisões pendentes", "fila"} {
		if !strings.Contains(frame, want) {
			t.Errorf("frame sem %q", want)
		}
	}
	m.tab = 1
	m.rebuildRows()
	frame = stripANSI(m.View())
	for _, want := range []string{"management-hub", "dotfiles", "em-interview", "pronta #1", "falta-ship"} {
		if !strings.Contains(frame, want) {
			t.Errorf("PLANO sem %q", want)
		}
	}
}

func visibleWidth(line string) int {
	return len([]rune(stripANSI(line))) // aproximação: só para detectar estouro grosseiro
}
