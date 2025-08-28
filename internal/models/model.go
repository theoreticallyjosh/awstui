package models

import (
	"fmt"
	"log"
	"strings"

	"github.com/theoreticallyjosh/awstui/internal/keys"
	"github.com/theoreticallyjosh/awstui/internal/messages"
	"github.com/theoreticallyjosh/awstui/internal/styles"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
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
	stateSFN
	stateBatch
)

// Model represents the state of our TUI application.
type Model struct {
	ec2Model    ec2Model
	ecsModel    ecsModel
	ecrModel    ecrModel
	sfnModel    sfnModel
	batchModel  batchModel
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

// configureList sets common styles and properties for a list.Model.
func configureList(l *list.Model, listkeys *keys.ListKeyMap) {
	st := list.DefaultStyles()
	st.Title = styles.SubHeaderStyle
	st.NoItems = styles.StatusStyle.UnsetPaddingLeft()
	st.StatusBar = styles.StatusStyle
	l.Help.Styles.ShortDesc = styles.HelpStyle
	l.Help.Styles.FullDesc = styles.HelpStyle
	l.Help.Styles.ShortKey = styles.HelpStyle
	l.Help.Styles.FullKey = styles.HelpStyle
	l.Paginator.ActiveDot = styles.ActivePager.Render("•")
	l.Paginator.InactiveDot = styles.InactivePager.Render("•")
	l.Styles = st
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
}

// newSpinner creates and configures a new spinner model.
func newSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.StatusStyle
	return s
}

// newAWSClients initializes and returns AWS service clients.
func newAWSClients() (*ec2.EC2, *ecs.ECS, *ecr.ECR, *cloudwatchlogs.CloudWatchLogs, *sfn.SFN, *batch.Batch) {
	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}

	return ec2.New(sess), ecs.New(sess), ecr.New(sess), cloudwatchlogs.New(sess), sfn.New(sess), batch.New(sess)
}

// newMainMenu creates and configures the main menu list.
func newMainMenu(listkeys *keys.ListKeyMap) list.Model {
	items := []list.Item{
		resourceItem{title: "EC2", desc: "Elastic Compute Cloud"},
		resourceItem{title: "ECS", desc: "Elastic Container Service"},
		resourceItem{title: "ECR", desc: "Elastic Container Registry"},
		resourceItem{title: "Step Functions", desc: "Step Functions"},
		resourceItem{title: "Batch", desc: "Batch Jobs"},
	}

	mainList := list.New(items, ItemDelegate{}, 0, 0)
	configureList(&mainList, listkeys)
	mainList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{listkeys.Choose}
	}
	mainList.AdditionalFullHelpKeys = mainList.AdditionalShortHelpKeys
	return mainList
}

// newEC2List creates and configures the EC2 instance list.
func newEC2List(listkeys *keys.ListKeyMap) list.Model {
	ec2List := list.New([]list.Item{}, ItemDelegate{}, 0, 0)
	configureList(&ec2List, listkeys)
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
	return ec2List
}

// newECSClusterList creates and configures the ECS cluster list.
func newECSClusterList(listkeys *keys.ListKeyMap) list.Model {
	ecsClusterList := list.New([]list.Item{}, ItemDelegate{}, 0, 0)
	configureList(&ecsClusterList, listkeys)
	ecsClusterList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Choose,
			listkeys.Refresh,
		}
	}
	ecsClusterList.AdditionalShortHelpKeys = ecsClusterList.AdditionalFullHelpKeys
	return ecsClusterList
}

// newECSServiceList creates and configures the ECS service list.
func newECSServiceList(listkeys *keys.ListKeyMap) list.Model {
	ecsServiceList := list.New([]list.Item{}, ItemDelegate{}, 0, 0)
	configureList(&ecsServiceList, listkeys)
	ecsServiceList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Details,
			listkeys.Stop,
			listkeys.Refresh,
			listkeys.Logs,
		}
	}
	ecsServiceList.AdditionalShortHelpKeys = ecsServiceList.AdditionalFullHelpKeys
	return ecsServiceList
}

// newECRRepositoryList creates and configures the ECR repository list.
func newECRRepositoryList(listkeys *keys.ListKeyMap) list.Model {
	ecrRepositoryList := list.New([]list.Item{}, ItemDelegate{}, 0, 0)
	configureList(&ecrRepositoryList, listkeys)
	ecrRepositoryList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Choose,
			listkeys.Refresh,
		}
	}
	ecrRepositoryList.AdditionalShortHelpKeys = ecrRepositoryList.AdditionalFullHelpKeys
	return ecrRepositoryList
}

// newECRImageList creates and configures the ECR image list.
func newECRImageList(listkeys *keys.ListKeyMap) list.Model {
	ecrImageList := list.New([]list.Item{}, ItemDelegate{}, 0, 0)
	configureList(&ecrImageList, listkeys)
	ecrImageList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Refresh,
			listkeys.Pull,
			listkeys.Push,
		}
	}
	ecrImageList.AdditionalShortHelpKeys = ecrImageList.AdditionalFullHelpKeys
	return ecrImageList
}

// newSFNList creates and configures the Step Functions state machine list.
func newSFNList(listkeys *keys.ListKeyMap) list.Model {
	sfnList := list.New([]list.Item{}, ItemDelegate{}, 0, 0)
	configureList(&sfnList, listkeys)
	sfnList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Choose,
			listkeys.Refresh,
			listkeys.StartExecution,
		}
	}
	sfnList.AdditionalShortHelpKeys = sfnList.AdditionalFullHelpKeys
	return sfnList
}

// newSFNExecutionList creates and configures the Step Functions execution list.
func newSFNExecutionList(listkeys *keys.ListKeyMap) list.Model {
	sfnExecutionList := list.New([]list.Item{}, ItemDelegate{}, 0, 0)
	configureList(&sfnExecutionList, listkeys)
	sfnExecutionList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Choose,
			listkeys.Refresh,
		}
	}
	sfnExecutionList.AdditionalShortHelpKeys = sfnExecutionList.AdditionalFullHelpKeys
	return sfnExecutionList
}

// newSFNExecutionHistoryList creates and configures the Step Functions execution history list.
func newSFNExecutionHistoryList(listkeys *keys.ListKeyMap) list.Model {
	sfnExecutionList := list.New([]list.Item{}, ItemDelegate{}, 0, 0)
	configureList(&sfnExecutionList, listkeys)
	sfnExecutionList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Refresh,
		}
	}
	sfnExecutionList.AdditionalShortHelpKeys = sfnExecutionList.AdditionalFullHelpKeys
	return sfnExecutionList
}

// newBatchJobQueueList creates and configures the Batch job queue list.
func newBatchJobQueueList(listkeys *keys.ListKeyMap) list.Model {
	batchJobQueueList := list.New([]list.Item{}, ItemDelegate{}, 0, 0)
	configureList(&batchJobQueueList, listkeys)
	batchJobQueueList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Choose,
			listkeys.Refresh,
		}
	}
	batchJobQueueList.AdditionalShortHelpKeys = batchJobQueueList.AdditionalFullHelpKeys
	return batchJobQueueList
}

// newBatchJobList creates and configures the Batch job list.
func newBatchJobList(listkeys *keys.ListKeyMap) list.Model {
	batchJobList := list.New([]list.Item{}, ItemDelegate{}, 0, 0)
	configureList(&batchJobList, listkeys)
	batchJobList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.Details,
			listkeys.Stop,
			listkeys.Refresh,
			listkeys.Logs,
		}
	}
	batchJobList.AdditionalShortHelpKeys = batchJobList.AdditionalFullHelpKeys
	return batchJobList
}

func newPaginator() paginator.Model {
	pager := paginator.New()
	pager.Type = paginator.Dots
	pager.ActiveDot = styles.ActivePager.Render("•")
	pager.InactiveDot = styles.InactivePager.Render("•")
	return pager
}

func NewModel() Model {
	ec2Svc, ecsSvc, ecrSvc, cloudwatchlogsSvc, sfnSvc, batchSvc := newAWSClients()
	s := newSpinner()
	listkeys := keys.NewListKeyMap()
	mainList := newMainMenu(listkeys)
	ec2List := newEC2List(listkeys)
	ecsClusterList := newECSClusterList(listkeys)
	ecsServiceList := newECSServiceList(listkeys)
	ecrRepositoryList := newECRRepositoryList(listkeys)
	ecrImageList := newECRImageList(listkeys)
	sfnList := newSFNList(listkeys)
	sfnExecutionList := newSFNExecutionList(listkeys)
	sfnExecutionHistoryList := newSFNExecutionHistoryList(listkeys)
	batchJobQueueList := newBatchJobQueueList(listkeys)
	batchJobList := newBatchJobList(listkeys)
	pager := newPaginator()

	m := Model{
		status:      "Select an option.",
		keys:        listkeys,
		state:       stateMenu,
		menuChoices: mainList,
		menuCursor:  0,
		spinner:     s,
		statusStyle: styles.StatusStyle,
	}

	m.ec2Model = ec2Model{
		parent:       &m,
		status:       "Loading instances...",
		ec2Svc:       ec2Svc,
		instanceList: ec2List,
		keys:         listkeys,
	}

	scaleInput := textinput.New()
	scaleInput.Placeholder = "Enter desired count"
	scaleInput.CharLimit = 10
	scaleInput.Width = 20

	m.ecsModel = ecsModel{
		parent:            &m,
		ecsSvc:            ecsSvc,
		status:            "Loading clusters...",
		cloudwatchlogsSvc: cloudwatchlogsSvc,
		clusterList:       ecsClusterList,
		serviceList:       ecsServiceList,
		scaleInput:        scaleInput,
		paginator:         pager,
		keys:              listkeys,
		state:             ecsStateClusterList,
	}

	m.ecrModel = ecrModel{
		parent:         &m,
		ecrSvc:         ecrSvc,
		status:         "Loading repositories...",
		repositoryList: ecrRepositoryList,
		imageList:      ecrImageList,
		keys:           listkeys,
		state:          ecrStateRepositoryList,
	}

	m.sfnModel = sfnModel{
		parent:               &m,
		sfnSvc:               sfnSvc,
		status:               "Loading state machines...",
		sfnList:              sfnList,
		executionList:        sfnExecutionList,
		executionHistoryList: sfnExecutionHistoryList,
		keys:                 listkeys,
		state:                sfnStateList,
		inputArea:            textarea.New(),
	}

	m.batchModel = batchModel{
		parent:            &m,
		batchSvc:          batchSvc,
		status:            "Loading job queues...",
		cloudwatchlogsSvc: cloudwatchlogsSvc,
		jobQueueList:      batchJobQueueList,
		jobList:           batchJobList,
		paginator:         pager,
		keys:              listkeys,
		state:             batchStateJobQueueList,
	}

	return m
}

func (m *Model) handleMenuChoose() tea.Cmd {
	selectedChoice := m.menuChoices.SelectedItem().FilterValue()
	switch selectedChoice {
	case "EC2":
		m.state = stateEC2
		return m.ec2Model.Init()
	case "ECS":
		m.state = stateECS
		return m.ecsModel.Init()
	case "ECR":
		m.state = stateECR
		return m.ecrModel.Init()
	case "Step Functions":
		m.state = stateSFN
		return m.sfnModel.Init()
	case "Batch":
		m.state = stateBatch
		return m.batchModel.Init()
	}
	return nil
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
		m.width = msg.Width - h
		m.height = msg.Height - v
		msg.Height = m.height - 3
		msg.Width = m.width

		m.menuChoices.SetSize(msg.Width, msg.Height)

		m.ec2Model, cmd = m.ec2Model.Update(msg)
		m.ecsModel, cmd = m.ecsModel.Update(msg)
		m.ecrModel, cmd = m.ecrModel.Update(msg)
		m.sfnModel, cmd = m.sfnModel.Update(msg)
		m.batchModel, cmd = m.batchModel.Update(msg)
	case tea.KeyMsg:
		switch m.state {
		case stateMenu:
			switch {
			case key.Matches(msg, m.keys.Choose):
				return m, m.handleMenuChoose()
			}
			m.menuChoices, cmd = m.menuChoices.Update(msg)

			return m, cmd
		case stateEC2, stateECS, stateECR, stateSFN, stateBatch:
			if m.ec2Model.instanceList.FilterState() == list.Filtering || m.ecsModel.serviceList.FilterState() == list.Filtering || m.ecsModel.clusterList.FilterState() == list.Filtering || m.ecrModel.repositoryList.FilterState() == list.Filtering || m.ecrModel.imageList.FilterState() == list.Filtering {
				break
			}
			if key.Matches(msg, key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))) {
				return m, m.handleEscKey(msg)
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

	// Delegate updates to sub-models based on current state
	switch m.state {
	case stateEC2:
		m.ec2Model, cmd = m.ec2Model.Update(msg)
	case stateECS:
		m.ecsModel, cmd = m.ecsModel.Update(msg)
	case stateECR:
		m.ecrModel, cmd = m.ecrModel.Update(msg)
	case stateSFN:
		m.sfnModel, cmd = m.sfnModel.Update(msg)
	case stateBatch:
		m.batchModel, cmd = m.batchModel.Update(msg)
	case stateMenu:
		m.menuChoices, cmd = m.menuChoices.Update(msg)
	}

	return m, cmd
}

// handleEscKey manages the logic for the escape key, allowing navigation back through states.
func (m *Model) handleEscKey(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch m.state {
	case stateEC2:
		if m.ec2Model.showDetails {
			m.ec2Model, cmd = m.ec2Model.Update(msg)
			return cmd
		}
	case stateECS:
		if m.ecsModel.state != ecsStateClusterList {
			m.ecsModel, cmd = m.ecsModel.Update(msg)
			return cmd
		}
	case stateECR:
		if m.ecrModel.state != ecrStateRepositoryList {
			m.ecrModel, cmd = m.ecrModel.Update(msg)
			return cmd
		}
	case stateBatch:
		if m.batchModel.state != batchStateJobQueueList {
			m.batchModel, cmd = m.batchModel.Update(msg)
			return cmd
		}
	case stateSFN:
		if m.sfnModel.state != sfnStateList {
			m.sfnModel, cmd = m.sfnModel.Update(msg)
			return cmd
		}
	}

	m.state = stateMenu
	m.status = "Select an option."
	m.err = nil
	return nil
}

// Header generates the header string for the application.
func (m Model) Header(items []string) string {
	ret := styles.HeaderStyle.Render(" 󰸏  AWS TUI ")
	for i, h := range items {
		if i > 0 {
			ret += styles.HeaderBarStyle.Render(" > ")
		} else {

			ret += styles.HeaderBarStyle.Render(" ")
		}
		ret += styles.SubHeaderStyle.Render(h)
	}
	remainingWidth := m.width - lipgloss.Width(ret)
	padding := styles.HeaderBarStyle.Width(remainingWidth).Render("") + "\n\n"
	return ret + padding
}

// View renders the TUI.
func (m Model) View() string {
	var s strings.Builder

	if m.err != nil {
		s.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n")
	}
	var status, spinner string
	switch m.state {
	case stateMenu:
		s.WriteString(m.Header(nil))
		s.WriteString(m.menuChoices.View())
		status = "Status: Ready"
	case stateEC2:
		s.WriteString(m.Header(m.ec2Model.Header))
		s.WriteString(m.ec2Model.View())
		status, spinner = m.renderStatusAndSpinner(m.ec2Model.status, m.ec2Model.confirming)
	case stateECS:
		s.WriteString(m.Header(m.ecsModel.header))
		s.WriteString(m.ecsModel.View())
		status, spinner = m.renderStatusAndSpinner(m.ecsModel.status, m.ecsModel.confirming)
	case stateECR:
		s.WriteString(m.Header(m.ecrModel.header))
		s.WriteString(m.ecrModel.View())
		status, spinner = m.renderStatusAndSpinner(m.ecrModel.status, false)
	case stateSFN:
		s.WriteString(m.Header(m.sfnModel.header))
		s.WriteString(m.sfnModel.View())
		status, spinner = m.renderStatusAndSpinner(m.sfnModel.status, false)
	case stateBatch:
		s.WriteString(m.Header(m.batchModel.header))
		s.WriteString(m.batchModel.View())
		status, spinner = m.renderStatusAndSpinner(m.batchModel.status, m.batchModel.confirming)
	}

	st := m.statusStyle.Render(spinner) + m.statusStyle.Render(status)

	remainingWidth := m.width - lipgloss.Width(st)
	remainingHeight := m.height - lipgloss.Height(s.String())
	padding := m.statusStyle.Width(remainingWidth).Render("")

	s.WriteString(lipgloss.NewStyle().Height(remainingHeight).Render(""))

	s.WriteString("\n" + st + padding)

	return styles.AppStyle.Render(s.String())
}

// renderStatusAndSpinner generates the status and spinner string based on the provided status and confirming state.
func (m Model) renderStatusAndSpinner(currentStatus string, confirming bool) (string, string) {
	var status, spinner string
	if currentStatus != "Ready" && currentStatus != "Error" {
		status = currentStatus
		spinner = m.spinner.View()
	} else if confirming {
		status = currentStatus
	} else {
		status = fmt.Sprintf("Status: %s", currentStatus)
	}
	return status, spinner
}
