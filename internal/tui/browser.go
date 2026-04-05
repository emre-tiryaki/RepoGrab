package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/emre-tiryaki/repograb/internal/models"
)

type BrowserModel struct {
	Items	[]models.FileNode
	Cursor	int
	Selected	map[int]struct{}
	Theme Theme
}

func NewBrowserModel(items []models.FileNode, theme Theme) BrowserModel {
	return BrowserModel{
		Items: items,
		Selected: make(map[int]struct{}),
		Theme: theme,
	}
}

func (b BrowserModel) View() string {
	s := ""

	for i, item := range b.Items {
		cursor := " "
		if b.Cursor == i {
			cursor = ">"
		}

		checked := " [ ] "
		if _, ok := b.Selected[i]; ok {
			checked = " [x] "
		}

		icon := "📃"
		style := lipgloss.NewStyle().Foreground(b.Theme.TextColor)

		if item.Type == "dir" {
			icon = "📁"
			style = style.Foreground(b.Theme.PrimaryColor).Bold(true)
		}

		line := fmt.Sprintf("%s%s%s%s", cursor, checked, icon, item.Name)
		if b.Cursor == i {
			s += lipgloss.NewStyle().
				Background(b.Theme.SelectedColor).
				Width(50).
				Render(line) + "\n"
		} else {
			s += style.Render(line) + "\n"
		}
	}

	return s
}