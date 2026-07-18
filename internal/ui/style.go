package ui

import "github.com/charmbracelet/lipgloss"

// Paleta discreta e adaptativa (dark/light): vermelho/laranja = bloqueio,
// verde = pronto para fechar, azul = fila, amarelo = higiene.
var (
	cRed    = lipgloss.AdaptiveColor{Light: "160", Dark: "203"}
	cOrange = lipgloss.AdaptiveColor{Light: "166", Dark: "215"}
	cGreen  = lipgloss.AdaptiveColor{Light: "28", Dark: "78"}
	cBlue   = lipgloss.AdaptiveColor{Light: "26", Dark: "75"}
	cYellow = lipgloss.AdaptiveColor{Light: "136", Dark: "179"}
	cDim    = lipgloss.AdaptiveColor{Light: "245", Dark: "243"}
	cText   = lipgloss.AdaptiveColor{Light: "236", Dark: "252"}
)

var (
	stTitle     = lipgloss.NewStyle().Bold(true)
	stDim       = lipgloss.NewStyle().Foreground(cDim)
	stRed       = lipgloss.NewStyle().Foreground(cRed)
	stOrange    = lipgloss.NewStyle().Foreground(cOrange)
	stGreen     = lipgloss.NewStyle().Foreground(cGreen)
	stBlue      = lipgloss.NewStyle().Foreground(cBlue)
	stYellow    = lipgloss.NewStyle().Foreground(cYellow)
	stTab       = lipgloss.NewStyle().Foreground(cDim).Padding(0, 1)
	stTabActive = lipgloss.NewStyle().Bold(true).Foreground(cText).Underline(true).Padding(0, 1)
	stSelected  = lipgloss.NewStyle().Bold(true).Background(lipgloss.AdaptiveColor{Light: "254", Dark: "237"})
	stHeaderRow = lipgloss.NewStyle().Bold(true).Foreground(cDim)
	stStatusErr = lipgloss.NewStyle().Foreground(cRed)
	stStatusOK  = lipgloss.NewStyle().Foreground(cGreen)
	stPane      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cDim).Padding(0, 1)
)

// colorFor devolve o estilo do ícone por seção.
func colorFor(k rowKind) lipgloss.Style {
	switch k {
	case rowGate:
		return stRed
	case rowSession:
		return stOrange
	case rowShip:
		return stGreen
	case rowQueue:
		return stBlue
	case rowOrphan:
		return stYellow
	default:
		return stDim
	}
}

// truncate corta s para caber em w colunas de célula (ANSI-unaware: usar
// antes de aplicar estilo), com reticências.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	var b []rune
	for _, r := range s {
		if lipgloss.Width(string(b)+string(r)) > w-1 {
			break
		}
		b = append(b, r)
	}
	return string(b) + "…"
}
