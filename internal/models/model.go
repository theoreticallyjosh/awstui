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
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
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
	stateECR
)

// Model represents the state of our TUI application.
type Model struct {
	ec2Model    ec2Model
	ecsModel    ecsModel
	ecrModel    ecrModel
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
	ecrSvc := ecr.New(sess)
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
	ec2List.Styles.Title = styles.SubHeaderStyle
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
	ecsClusterList.Styles.Title = styles.SubHeaderStyle
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
	ecsServiceList.Title = ""
	ecsServiceList.SetShowStatusBar(false)
	ecsServiceList.SetFilteringEnabled(true)
	ecsServiceList.Styles.Title = styles.SubHeaderStyle
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

	// Initialize the ECR repository list model
	ecrRepositoryList := list.New([]list.Item{}, ItemDelegate{}, 0, 20)
	ecrRepositoryList.Title = "ECR Repositories"
	ecrRepositoryList.SetShowStatusBar(false)
	ecrRepositoryList.SetFilteringEnabled(true)
	ecrRepositoryList.Styles.Title = styles.SubHeaderStyle
	ecrRepositoryList.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(styles.TokyoNightGreen)
	ecrRepositoryList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(styles.TokyoNightGreen)
	ecrRepositoryList.Styles.NoItems = styles.StatusStyle.UnsetPaddingLeft()
	ecrRepositoryList.SetStatusBarItemName("repository", "repositories")
	ecrRepositoryList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Refresh,
		}
	}

	// Initialize the ECR repository list model
	ecrImageList := list.New([]list.Item{}, ItemDelegate{}, 0, 20)
	ecrImageList.Title = "ECR Images"
	ecrImageList.SetShowStatusBar(false)
	ecrImageList.SetFilteringEnabled(true)
	ecrImageList.Styles.Title = styles.SubHeaderStyle
	ecrImageList.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(styles.TokyoNightGreen)
	ecrImageList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(styles.TokyoNightGreen)
	ecrImageList.Styles.NoItems = styles.StatusStyle.UnsetPaddingLeft()
	ecrImageList.SetStatusBarItemName("image", "images")
	ecrImageList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Refresh,
			listkeys.Pull,
			listkeys.Push,
		}
	}

	pager := paginator.New()
	pager.Type = paginator.Dots
	pager.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	pager.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")

	// Create initial model
	m := Model{
		status:      "Select an option.",
		keys:        listkeys,
		state:       stateMenu,
		menuChoices: []string{"EC2 Instances", "ECS Clusters", "ECR Repositories"},
		menuCursor:  0,
		spinner:     s,
	}

	ec2Model := ec2Model{
		parent:       &m,
		status:       "Loading instances...",
		ec2Svc:       ec2Svc,
		instanceList: ec2List,
		keys:         listkeys,
	}

	ecsModel := ecsModel{
		parent:            &m,
		ecsSvc:            ecsSvc,
		status:            "Loading clusters...",
		cloudwatchlogsSvc: cloudwatchlogsSvc,
		clusterList:       ecsClusterList,
		serviceList:       ecsServiceList,
		paginator:         pager,
		keys:              listkeys,
		state:             ecsStateClusterList,
	}

	ecrModel := ecrModel{
		parent:         &m,
		ecrSvc:         ecrSvc,
		status:         "Loading repositories...",
		repositoryList: ecrRepositoryList,
		imageList:      ecrImageList,
		keys:           listkeys,
		state:          ecrStateRepositoryList,
	}

	m.ec2Model = ec2Model
	m.ecsModel = ecsModel
	m.ecrModel = ecrModel

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
		m.ecrModel, cmd = m.ecrModel.Update(msg)
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
					return m, m.ec2Model.Init()
				case "ECS Clusters":
					m.state = stateECS
					return m, m.ecsModel.Init()
				case "ECR Repositories":
					m.state = stateECR
					return m, m.ecrModel.Init()
				}
			case "q", "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		case stateEC2, stateECS, stateECR:
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
				if m.state == stateECR {
					if m.ecrModel.state != ecrStateRepositoryList {
						m.ecrModel, cmd = m.ecrModel.Update(msg)
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
	case stateECR:
		m.ecrModel, cmd = m.ecrModel.Update(msg)
	}

	return m, cmd
}

// View renders the TUI.
func (m Model) View() string {
	var s strings.Builder

	s.WriteString(styles.HeaderStyle.Render("AWS Resource Manager") + "\n")

	if m.err != nil {
		s.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n")
	}

	switch m.state {
	case stateMenu:
		s.WriteString("\nSelect a resource type:\n\n")
		for i, choice := range m.menuChoices {
			cursor := ""
			if m.menuCursor == i {
				cursor = ""
				s.WriteString(fmt.Sprintf("%s %s\n", cursor, styles.SelectedItemStyle.Render(choice)))
			} else {
				s.WriteString(fmt.Sprintf("%s %s\n", cursor, styles.MenuItemStyle.Render(choice)))
			}
		}
		s.WriteString(styles.StatusStyle.Render("\n(Press 'q' or 'ctrl+c' to quit)") + "\n")

	case stateEC2:
		s.WriteString(m.ec2Model.View())
		var status string
		if m.ec2Model.status != "Ready" && m.ec2Model.status != "Error" {
			status += styles.StatusStyle.Render(fmt.Sprintf("\n%s %s", m.spinner.View(), m.ec2Model.status))
		} else if m.ec2Model.confirming {
			status += styles.ConfirmStyle.Render(fmt.Sprintf("\n%s", m.ec2Model.status))
		} else {
			status += styles.StatusStyle.Render(fmt.Sprintf("\nStatus: %s", m.ec2Model.status))
		}
		s.WriteString(status)

	case stateECS:
		s.WriteString(m.ecsModel.View())
		var status string
		if m.ecsModel.status != "Ready" && m.ecsModel.status != "Error" {
			status += styles.StatusStyle.Render(fmt.Sprintf("\n%s %s", m.spinner.View(), m.ecsModel.status))
		} else if m.ec2Model.confirming {
			status += styles.ConfirmStyle.Render(fmt.Sprintf("\n%s", m.ecsModel.status))
		} else {
			status += styles.StatusStyle.Render(fmt.Sprintf("\nStatus: %s", m.ecsModel.status))
		}
		s.WriteString(status)
	case stateECR:
		s.WriteString(m.ecrModel.View())
		var status string
		if m.ecrModel.status != "Ready" && m.ecrModel.status != "Error" {
			status += styles.StatusStyle.Render(fmt.Sprintf("\n%s %s", m.spinner.View(), m.ecrModel.status))
		} else {
			status += styles.StatusStyle.Render(fmt.Sprintf("\nStatus: %s", m.ecrModel.status))
		}
		s.WriteString(status)
	}

	return styles.AppStyle.Render(s.String())
}
