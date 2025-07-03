package main

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles incoming messages and updates the model's state.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ec2Model, cmd = m.ec2Model.Update(msg)
		m.ecsModel, cmd = m.ecsModel.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		switch m.state {
		case stateMenu:
			switch msg.String() {
			case "up", "k":
				if m.menuCursor > 0 {
					m.menuCursor--
				}
			case "down", "j":
				if m.menuCursor < len(m.menuChoices)-1 {
					m.menuCursor++
				}
			case "enter":
				selectedChoice := m.menuChoices[m.menuCursor]
				switch selectedChoice {
				case "EC2 Instances":
					m.state = stateEC2
					m.status = "Loading EC2 instances..."
					return m, m.ec2Model.Init()
				case "ECS Clusters":
					m.state = stateECS
					m.status = "Loading ECS clusters..."
					return m, m.ecsModel.Init()
				}
			case "q", "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		case stateEC2, stateECS:
			if key.Matches(msg, key.NewBinding(key.WithKeys("backspace", "esc"), key.WithHelp("backspace/esc", "back"))) {
				if m.state == stateEC2 {
					if m.ec2Model.showDetails {
						m.ec2Model, cmd = m.ec2Model.Update(msg)
						return m, cmd
					}
				}
				if m.state == stateECS {
					if m.ecsModel.state != ecsStateClusterList {
						m.ecsModel, cmd = m.ecsModel.Update(msg)
						return m, cmd
					}
				}
				m.state = stateMenu
				m.status = "Select an option."
				m.err = nil
				return m, nil
			}
		}
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case errMsg:
		m.err = msg
		m.status = "Error"
		return m, nil
	}

	switch m.state {
	case stateEC2:
		m.ec2Model, cmd = m.ec2Model.Update(msg)
	case stateECS:
		m.ecsModel, cmd = m.ecsModel.Update(msg)
	}

	return m, cmd
}
