package tui

import "github.com/charmbracelet/lipgloss"

/* 	Project UI themes are on this file.
	If you want to add a new theme just add a new function here
	Functions should be like this:
	func [your theme name]Theme() Theme{
		return Theme{
			PrimaryColor:   lipgloss.Color("Your color"),
			SecondaryColor: lipgloss.Color("Your color"),
			BackgroundColor: lipgloss.Color("Your color"),
			TextColor:      lipgloss.Color("Your color"),
			SelectedColor:  lipgloss.Color("Your color"),
			ErrorColor:     lipgloss.Color("Your color"),
		}
	}
 */

type Theme struct {
	PrimaryColor   lipgloss.Color
	SecondaryColor lipgloss.Color
	BackgroundColor lipgloss.Color
	TextColor      lipgloss.Color
	SelectedColor  lipgloss.Color
	ErrorColor     lipgloss.Color
}

func DefaultTheme() Theme {
	return Theme{
		PrimaryColor:   lipgloss.Color("#7D56F4"),
		SecondaryColor: lipgloss.Color("#04B575"),
		BackgroundColor: lipgloss.Color("#1A1B26"),
		TextColor:      lipgloss.Color("#C0CAF5"),
		SelectedColor:  lipgloss.Color("#3D59A1"),
		ErrorColor:     lipgloss.Color("#F7768E"),
	}
}
