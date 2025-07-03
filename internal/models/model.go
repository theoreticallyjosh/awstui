package models

import (
	"awstui/internal/keys"
	"awstui/internal/messages"
	"awstui/internal/styles"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// appState defines the current state of the application.
type appState int

const (
	stateMenu appState = iota
	stateEC2
	stateECS
)

// Model represents the state of our TUI application.
type Model struct {
	ec2Model    ec2Model
	ecsModel    ecsModel
	spinner     spinner.Model
	status      string
	err         error
	state       appState
	menuCursor  int
	menuChoices []string
	keys        *keys.ListKeyMap
}

func NewModel() Model {

	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}

	// Create AWS service clients
	ec2Svc := ec2.New(sess)
	ecsSvc := ecs.New(sess)
	cloudwatchlogsSvc := cloudwatchlogs.New(sess)

	// Initialize the spinner model
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.TokyoNightCyan)

	var listkeys = keys.NewListKeyMap()
	// Initialize the EC2 list model
	ec2List := list.New([]list.Item{}, ItemDelegate{}, 0, 20)
	ec2List.Title = "EC2 Instances"
	ec2List.SetShowStatusBar(false)
	ec2List.SetFilteringEnabled(true)
	ec2List.Styles.Title = styles.HeaderStyle
	ec2List.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(styles.TokyoNightGreen)
	ec2List.Styles.FilterCursor = lipgloss.NewStyle().Foreground(styles.TokyoNightGreen)
	ec2List.Styles.NoItems = styles.StatusStyle.UnsetPaddingLeft()
	ec2List.SetStatusBarItemName("instance", "instances")
	ec2List.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Details,
			listkeys.Start,
			listkeys.Stop,
			listkeys.Ssh,
			listkeys.Refresh,
		}
	}

	// Initialize the ECS cluster list model
	ecsClusterList := list.New([]list.Item{}, ItemDelegate{}, 0, 20)
	ecsClusterList.Title = "ECS Clusters"
	ecsClusterList.SetShowStatusBar(false)
	ecsClusterList.SetFilteringEnabled(true)
	ecsClusterList.Styles.Title = styles.HeaderStyle
	ecsClusterList.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(styles.TokyoNightGreen)
	ecsClusterList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(styles.TokyoNightGreen)
	ecsClusterList.Styles.NoItems = styles.StatusStyle.UnsetPaddingLeft()
	ecsClusterList.SetStatusBarItemName("cluster", "clusters")
	ecsClusterList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Refresh,
		}
	}

	// Initialize the ECS service list model
	ecsServiceList := list.New([]list.Item{}, ItemDelegate{}, 0, 20)
	ecsServiceList.Title = "ECS Services"
	ecsServiceList.SetShowStatusBar(false)
	ecsServiceList.SetFilteringEnabled(true)
	ecsServiceList.Styles.Title = styles.HeaderStyle
	ecsServiceList.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(styles.TokyoNightGreen)
	ecsServiceList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(styles.TokyoNightGreen)
	ecsServiceList.Styles.NoItems = styles.StatusStyle.UnsetPaddingLeft()
	ecsServiceList.SetStatusBarItemName("service", "services")
	ecsServiceList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Details,
			listkeys.Stop,
			listkeys.Refresh,
			listkeys.Logs,
		}
	}

	ec2Model := ec2Model{
		ec2Svc:       ec2Svc,
		instanceList: ec2List,
		spinner:      s,
		keys:         listkeys,
	}

	ecsModel := ecsModel{
		ecsSvc:            ecsSvc,
		cloudwatchlogsSvc: cloudwatchlogsSvc,
		clusterList:       ecsClusterList,
		serviceList:       ecsServiceList,
		spinner:           s,
		keys:              listkeys,
		state:             ecsStateClusterList,
	}

	// Create initial model
	m := Model{
		ec2Model:    ec2Model,
		ecsModel:    ecsModel,
		status:      "Select an option.",
		spinner:     s,
		keys:        listkeys,
		state:       stateMenu,
		menuChoices: []string{"EC2 Instances", "ECS Clusters"},
		menuCursor:  0,
	}

	return m

}

// Init initializes the model and starts fetching data based on the initial state.
func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles incoming messages and updates the model's state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case messages.ErrMsg:
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

// View renders the TUI.
func (m Model) View() string {
	var s strings.Builder

	s.WriteString(styles.HeaderStyle.Render("AWS Resource Manager\n"))

	if m.err != nil {
		s.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("Error: %v\n", m.err)))
	}

	switch m.state {
	case stateMenu:
		s.WriteString("\nSelect a resource type:\n\n")
		for i, choice := range m.menuChoices {
			cursor := " "
			if m.menuCursor == i {
				cursor = ">"
				s.WriteString(fmt.Sprintf("%s %s\n", cursor, styles.SelectedItemStyle.Render(choice)))
			} else {
				s.WriteString(fmt.Sprintf("%s %s\n", cursor, styles.MenuItemStyle.Render(choice)))
			}
		}
		s.WriteString(styles.StatusStyle.Render("\n(Press 'q' or 'ctrl+c' to quit)\n"))

	case stateEC2:
		s.WriteString(m.ec2Model.View())
	case stateECS:
		s.WriteString(m.ecsModel.View())
	}

	return styles.AppStyle.Render(s.String())
}
