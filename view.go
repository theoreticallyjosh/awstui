package main

import (
	"fmt"
	"strings"
)

// View renders the TUI.
func (m model) View() string {
	var s strings.Builder

	s.WriteString(headerStyle.Render("AWS Resource Manager\n"))

	if m.err != nil {
		s.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v\n", m.err)))
	}

	switch m.state {
	case stateMenu:
		s.WriteString("\nSelect a resource type:\n\n")
		for i, choice := range m.menuChoices {
			cursor := " "
			if m.menuCursor == i {
				cursor = ">"
				s.WriteString(fmt.Sprintf("%s %s\n", cursor, selectedItemStyle.Render(choice)))
			} else {
				s.WriteString(fmt.Sprintf("%s %s\n", cursor, menuItemStyle.Render(choice)))
			}
		}
		s.WriteString(statusStyle.Render("\n(Press 'q' or 'ctrl+c' to quit)\n"))

	case stateEC2:
		s.WriteString(m.ec2Model.View())
	case stateECS:
		s.WriteString(m.ecsModel.View())
	}

	return appStyle.Render(s.String())
}
