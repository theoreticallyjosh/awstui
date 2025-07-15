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
	menuChoices list.Model
	keys        *keys.ListKeyMap
	width       int
	height      int
	statusStyle lipgloss.Style
}

func setListStyle(l *list.Model) {
	st := list.DefaultStyles()

	st.Title = styles.SubHeaderStyle
	st.FilterPrompt = lipgloss.NewStyle().Foreground(styles.TokyoNightGreen)
	st.FilterCursor = lipgloss.NewStyle().Foreground(styles.TokyoNightGreen)
	st.NoItems = styles.StatusStyle.UnsetPaddingLeft()
	st.StatusBar = styles.StatusStyle
	st.NoItems = styles.StatusStyle.UnsetPaddingLeft()
	l.Help.Styles.ShortDesc = styles.HelpStyle
	l.Help.Styles.FullDesc = styles.HelpStyle
	l.Help.Styles.ShortKey = styles.HelpStyle
	l.Help.Styles.FullKey = styles.HelpStyle
	l.Styles = st
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
	s.Style = styles.StatusStyle

	var listkeys = keys.NewListKeyMap()

	items := []list.Item{
		resourceItem{title: "EC2", desc: "Elastic Compute Cloud"},
		resourceItem{title: "ECS", desc: "Elastic Container Service"},
		resourceItem{title: "ECR", desc: "Elastic Container Registry"},
	}

	mainList := list.New(items, ItemDelegate{}, 0, 6)
	mainList.Title = "Resources"
	mainList.SetShowStatusBar(false)
	mainList.SetFilteringEnabled(false)
	setListStyle(&mainList)
	mainList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Choose,
		}
	}
	mainList.AdditionalFullHelpKeys = mainList.AdditionalShortHelpKeys

	// Initialize the EC2 list model
	ec2List := list.New([]list.Item{}, ItemDelegate{}, 0, 20)
	ec2List.Title = "EC2 Instances"
	ec2List.SetShowStatusBar(false)
	ec2List.SetFilteringEnabled(true)
	setListStyle(&ec2List)

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
	ec2List.AdditionalShortHelpKeys = ec2List.AdditionalFullHelpKeys

	// Initialize the ECS cluster list model
	ecsClusterList := list.New([]list.Item{}, ItemDelegate{}, 0, 20)
	ecsClusterList.Title = "ECS Clusters"
	ecsClusterList.SetShowStatusBar(false)
	ecsClusterList.SetFilteringEnabled(true)
	setListStyle(&ecsClusterList)
	ecsClusterList.SetStatusBarItemName("cluster", "clusters")
	ecsClusterList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Choose,
			listkeys.Refresh,
		}
	}
	ecsClusterList.AdditionalShortHelpKeys = ecsClusterList.AdditionalFullHelpKeys

	// Initialize the ECS service list model
	ecsServiceList := list.New([]list.Item{}, ItemDelegate{}, 0, 20)
	ecsServiceList.Title = ""
	ecsServiceList.SetShowStatusBar(false)
	ecsServiceList.SetFilteringEnabled(true)
	setListStyle(&ecsServiceList)
	ecsServiceList.SetStatusBarItemName("service", "services")
	ecsServiceList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Details,
			listkeys.Stop,
			listkeys.Refresh,
			listkeys.Logs,
		}
	}
	ecsServiceList.AdditionalShortHelpKeys = ecsServiceList.AdditionalFullHelpKeys

	// Initialize the ECR repository list model
	ecrRepositoryList := list.New([]list.Item{}, ItemDelegate{}, 0, 20)
	ecrRepositoryList.Title = "ECR Repositories"
	ecrRepositoryList.SetShowStatusBar(false)
	ecrRepositoryList.SetFilteringEnabled(true)
	setListStyle(&ecrRepositoryList)
	ecrRepositoryList.SetStatusBarItemName("repository", "repositories")
	ecrRepositoryList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Choose,
			listkeys.Refresh,
		}
	}
	ecrRepositoryList.AdditionalShortHelpKeys = ecrRepositoryList.AdditionalFullHelpKeys

	// Initialize the ECR repository list model
	ecrImageList := list.New([]list.Item{}, ItemDelegate{}, 0, 20)
	ecrImageList.Title = ""
	ecrImageList.SetShowStatusBar(false)
	ecrImageList.SetFilteringEnabled(true)
	setListStyle(&ecrImageList)
	ecrImageList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Refresh,
			listkeys.Pull,
			listkeys.Push,
		}
	}
	ecrImageList.AdditionalShortHelpKeys = ecrImageList.AdditionalFullHelpKeys

	pager := paginator.New()
	pager.Type = paginator.Dots
	pager.ActiveDot = styles.ActivePager.Render("•")
	pager.InactiveDot = styles.InactivePager.Render("•")

	// Create initial model
	m := Model{
		status:      "Select an option.",
		keys:        listkeys,
		state:       stateMenu,
		menuChoices: mainList,
		menuCursor:  0,
		spinner:     s,
		statusStyle: styles.StatusStyle,
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
		h, v := styles.AppStyle.GetFrameSize()
		m.width = msg.Width - 3*h
		m.height = msg.Height - 2*v
		m.ec2Model, cmd = m.ec2Model.Update(msg)
		m.ecsModel, cmd = m.ecsModel.Update(msg)
		m.ecrModel, cmd = m.ecrModel.Update(msg)
		m.menuChoices.SetSize(m.width, m.height)
		// m.menuChoices, cmd = m.menuChoices.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		switch m.state {
		case stateMenu:
			switch {
			case key.Matches(msg, m.keys.Choose):
				selectedChoice := m.menuChoices.SelectedItem().FilterValue()
				switch selectedChoice {
				case "EC2":
					m.state = stateEC2
					return m, m.ec2Model.Init()
				case "ECS":
					m.state = stateECS
					return m, m.ecsModel.Init()
				case "ECR":
					m.state = stateECR
					return m, m.ecrModel.Init()
				}
			}
			m.menuChoices, cmd = m.menuChoices.Update(msg)

			return m, cmd
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
	case stateMenu:
		m.menuChoices, cmd = m.menuChoices.Update(msg)
	}

	return m, cmd
}

// View renders the TUI.
func (m Model) View() string {
	var s strings.Builder

	s.WriteString(styles.HeaderStyle.Render("AWS Resource Manager") + " > ")

	if m.err != nil {
		s.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n")
	}
	var status, spinner string
	switch m.state {
	case stateMenu:
		s.WriteString(m.menuChoices.View())
	case stateEC2:
		s.WriteString(m.ec2Model.View())
		if m.ec2Model.status != "Ready" && m.ec2Model.status != "Error" {
			status = m.ec2Model.status
			spinner = m.spinner.View()
		} else if m.ec2Model.confirming {
			status = m.ec2Model.status
		} else {
			status += fmt.Sprintf("Status: %s", m.ec2Model.status)
		}

	case stateECS:
		s.WriteString(m.ecsModel.View())
		if m.ecsModel.status != "Ready" && m.ecsModel.status != "Error" {
			status = m.ecsModel.status
			spinner = m.spinner.View()
		} else if m.ec2Model.confirming {
			status = fmt.Sprintf("%s", m.ecsModel.status)
		} else {
			status = fmt.Sprintf("Status: %s", m.ecsModel.status)
		}
	case stateECR:
		s.WriteString(m.ecrModel.View())
		var status string
		if m.ecrModel.status != "Ready" && m.ecrModel.status != "Error" {
			status += m.ecrModel.status
			spinner = m.spinner.View()
		} else {
			status += fmt.Sprintf("Status: %s", m.ecrModel.status)
		}
	}

	remainingHeight := m.height - lipgloss.Height(s.String())
	for range remainingHeight {
		s.WriteString("\n")
	}

	st := m.statusStyle.Render(spinner) + m.statusStyle.Render(status)

	remainingWidth := m.width - lipgloss.Width(st)
	padding := m.statusStyle.Width(remainingWidth).Render("")

	s.WriteString("\n" + st + padding)

	return styles.AppStyle.Render(s.String())
}
