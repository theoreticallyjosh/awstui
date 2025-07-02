package main

import (
	"fmt"
	"io"
	"log"
	"os/exec" // Import os/exec
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs" // Import ECS service
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list" // Import bubbles/list
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tokyo Night Color Palette
var (
	tokyoNightBlue    = lipgloss.Color("#7AA2F7")
	tokyoNightGreen   = lipgloss.Color("#9EEB49")
	tokyoNightYellow  = lipgloss.Color("#E0AF68")
	tokyoNightRed     = lipgloss.Color("#F7768E")
	tokyoNightPurple  = lipgloss.Color("#BB9AF7")
	tokyoNightCyan    = lipgloss.Color("#7DCFFF")
	tokyoNightGray    = lipgloss.Color("#A9B1D6")
	tokyoNightDarkBg  = lipgloss.Color("#1A1B26")
	tokyoNightLightFg = lipgloss.Color("#C0CAF5")
)

// Define styles using lipgloss
var (
	// Base style for the entire application
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	// Style for headers
	headerStyle = lipgloss.NewStyle().
			Foreground(tokyoNightPurple).
			Bold(true).
			PaddingBottom(1)

	// Style for status messages (e.g., loading, success)
	statusStyle = lipgloss.NewStyle().
			Foreground(tokyoNightGray).
			PaddingTop(1)

	// Style for error messages
	errorStyle = lipgloss.NewStyle().
			Foreground(tokyoNightRed).
			Bold(true).
			PaddingTop(1)

	// Style for confirmation prompts
	confirmStyle = lipgloss.NewStyle().
			Foreground(tokyoNightGreen).
			Bold(true).
			PaddingTop(1)

	// Style for detail view
	detailStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true).
			BorderForeground(tokyoNightBlue).
			Padding(1, 2)

	// List item styles
	titleStyle          = lipgloss.NewStyle().Bold(true)
	descriptionStyle    = lipgloss.NewStyle().Foreground(tokyoNightGray)
	selectedItemStyle   = lipgloss.NewStyle().Foreground(tokyoNightBlue)
	unselectedItemStyle = lipgloss.NewStyle()

	// Menu item styles
	menuItemStyle         = lipgloss.NewStyle()
	selectedMenuItemStyle = lipgloss.NewStyle().Foreground(tokyoNightBlue).String()
)

// ec2InstanceItem represents a single EC2 instance in the list.
type ec2InstanceItem struct {
	instance *ec2.Instance
}

// FilterValue implements list.Item.
func (i ec2InstanceItem) FilterValue() string {
	return getInstanceName(i.instance) + " " + aws.StringValue(i.instance.InstanceId) + " " + aws.StringValue(i.instance.State.Name)
}

// Title implements list.DefaultItem.
func (i ec2InstanceItem) Title() string {
	return titleStyle.Render(fmt.Sprintf("%s (%s)", getInstanceName(i.instance), aws.StringValue(i.instance.InstanceId)))
}

// Description implements list.DefaultItem.
func (i ec2InstanceItem) Description() string {
	return descriptionStyle.Render(fmt.Sprintf("Type: %s | State: %s | Public IP: %s",
		aws.StringValue(i.instance.InstanceType),
		aws.StringValue(i.instance.State.Name),
		aws.StringValue(i.instance.PublicIpAddress),
	))
}

// ecsClusterItem represents a single ECS cluster in the list.
type ecsClusterItem struct {
	cluster *ecs.Cluster
}

// FilterValue implements list.Item for ECS clusters.
func (i ecsClusterItem) FilterValue() string {
	return aws.StringValue(i.cluster.ClusterName) + " " + aws.StringValue(i.cluster.Status)
}

// Title implements list.DefaultItem for ECS clusters.
func (i ecsClusterItem) Title() string {
	return titleStyle.Render(fmt.Sprintf("%s", aws.StringValue(i.cluster.ClusterName)))
}

// Description implements list.DefaultItem for ECS clusters.
func (i ecsClusterItem) Description() string {
	return descriptionStyle.Render(fmt.Sprintf("Status: %s | Services: %d | Tasks: %d | Container Instances: %d",
		aws.StringValue(i.cluster.Status),
		aws.Int64Value(i.cluster.ActiveServicesCount),
		aws.Int64Value(i.cluster.RunningTasksCount),
		aws.Int64Value(i.cluster.RegisteredContainerInstancesCount),
	))
}

// ecsServiceItem represents a single ECS service in the list.
type ecsServiceItem struct {
	service *ecs.Service
}

// FilterValue implements list.Item for ECS services.
func (i ecsServiceItem) FilterValue() string {
	return aws.StringValue(i.service.ServiceName) + " " + aws.StringValue(i.service.Status)
}

// Title implements list.DefaultItem for ECS services.
func (i ecsServiceItem) Title() string {
	return titleStyle.Render(fmt.Sprintf("%s", aws.StringValue(i.service.ServiceName)))
}

// Description implements list.DefaultItem for ECS services.
func (i ecsServiceItem) Description() string {
	return descriptionStyle.Render(fmt.Sprintf("Status: %s | Desired: %d | Running: %d | Pending: %d",
		aws.StringValue(i.service.Status),
		aws.Int64Value(i.service.DesiredCount),
		aws.Int64Value(i.service.RunningCount),
		aws.Int64Value(i.service.PendingCount),
	))
}

// itemDelegate customizes how each item in the list is rendered.
type itemDelegate struct {
}

func (d itemDelegate) Height() int                               { return 2 }
func (d itemDelegate) Spacing() int                              { return 1 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title, description string
	var style lipgloss.Style

	if index == m.Index() {
		style = selectedItemStyle
	} else {
		style = unselectedItemStyle
	}

	switch i := item.(type) {
	case ec2InstanceItem:
		title = i.Title()
		description = i.Description()
	case ecsClusterItem:
		title = i.Title()
		description = i.Description()
	case ecsServiceItem: // Handle ECS service items
		title = i.Title()
		description = i.Description()
	default:
		return
	}

	str := fmt.Sprintf("%s\n%s", title, description)
	fmt.Fprint(w, style.Render(str))
}

// appState defines the current state of the application.
type appState int

const (
	stateMenu appState = iota
	stateEC2Instances
	stateECSClusters
	stateECSServices // New state for listing services within a cluster
)

// model represents the state of our TUI application.
type model struct {
	ec2Svc         *ec2.EC2      // AWS EC2 service client
	ecsSvc         *ecs.ECS      // AWS ECS service client
	instanceList   list.Model    // List of EC2 instances using bubbles/list
	clusterList    list.Model    // List of ECS clusters using bubbles/list
	serviceList    list.Model    // New: List of ECS services using bubbles/list
	status         string        // Current status message (e.g., "Loading...", "Ready")
	err            error         // Any error encountered
	spinner        spinner.Model // Spinner for loading states
	confirming     bool          // True if waiting for user confirmation
	action         string        // The action being confirmed ("stop" or "start" for EC2)
	actionID       *string       // The ID of the instance for the pending action
	showDetails    bool          // True if showing instance details (for EC2)
	detailInstance *ec2.Instance // The instance whose details are currently displayed (for EC2)
	detailCluster  *ecs.Cluster  // New: The ECS cluster whose services are currently displayed
	keys           *listKeyMap
	state          appState // Current application state (menu, ec2, ecs, ecsServices)
	menuCursor     int      // Cursor for menu selection
	menuChoices    []string // Menu options
}

// messages are used to pass data between commands and the Update function.
type (
	instancesFetchedMsg   []*ec2.Instance // Message when instances are fetched
	ecsClustersFetchedMsg []*ecs.Cluster  // Message when ECS clusters are fetched
	ecsServicesFetchedMsg []*ecs.Service  // New: Message when ECS services are fetched
	instanceActionMsg     string          // Message when an instance action (stop/start) is completed
	instanceDetailsMsg    *ec2.Instance   // Message when instance details are fetched
	sshFinishedMsg        error           // Message when SSH command finishes (nil for success, error for failure)
	errMsg                error           // Message for errors
)

// Init initializes the model and starts fetching data based on the initial state.
func (m model) Init() tea.Cmd {
	// Start the spinner animation and fetch data based on the current state
	switch m.state {
	case stateEC2Instances:
		return tea.Batch(m.spinner.Tick, fetchInstancesCmd(m.ec2Svc))
	case stateECSClusters:
		return tea.Batch(m.spinner.Tick, fetchECSClustersCmd(m.ecsSvc))
	case stateECSServices:
		if m.detailCluster != nil {
			return tea.Batch(m.spinner.Tick, fetchECSServicesCmd(m.ecsSvc, aws.StringValue(m.detailCluster.ClusterArn)))
		}
	}
	return nil
}

// Update handles incoming messages and updates the model's state.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.instanceList.SetSize(msg.Width-2*h, msg.Height-2*v)
		m.clusterList.SetSize(msg.Width-2*h, msg.Height-2*v)
		m.serviceList.SetSize(msg.Width-2*h, msg.Height-2*v) // Set size for service list too
		return m, nil
	case tea.KeyMsg:
		if m.state == stateMenu {
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
					m.state = stateEC2Instances
					m.status = "Loading EC2 instances..."
					return m, tea.Batch(m.spinner.Tick, fetchInstancesCmd(m.ec2Svc))
				case "ECS Clusters":
					m.state = stateECSClusters
					m.status = "Loading ECS clusters..."
					return m, tea.Batch(m.spinner.Tick, fetchECSClustersCmd(m.ecsSvc))
				}
			case "q", "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		}

		// Handle keys specific to EC2, ECS clusters, or ECS services lists
		switch m.state {
		case stateEC2Instances:
			if m.instanceList.FilterState() == list.Filtering {
				break
			}
			if m.confirming {
				// Handle confirmation keys for EC2
				switch msg.String() {
				case "y", "Y":
					m.confirming = false
					m.status = fmt.Sprintf("%sing instance %s...", m.action, *m.actionID)
					m.err = nil
					if m.action == "stop" {
						return m, tea.Batch(m.spinner.Tick, stopInstanceCmd(m.ec2Svc, m.actionID))
					} else if m.action == "start" {
						return m, tea.Batch(m.spinner.Tick, startInstanceCmd(m.ec2Svc, m.actionID))
					}
				case "n", "N":
					m.confirming = false
					m.status = "Action cancelled."
					m.action = ""
					m.actionID = nil
				}
				return m, nil // Don't pass key to list if confirming
			}

			if m.showDetails {
				// Handle detail view keys for EC2
				switch msg.String() {
				case "esc", "backspace":
					m.showDetails = false
					m.detailInstance = nil
					m.status = "Ready"
					m.err = nil
				}
				return m, nil // Don't pass key to list if showing details
			}

			// Handle keys for the main EC2 list view
			switch {
			case key.Matches(msg, m.keys.refresh):
				m.status = "Refreshing instances..."
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, fetchInstancesCmd(m.ec2Svc))
			case key.Matches(msg, m.keys.stop):
				if m.instanceList.SelectedItem() != nil {
					selectedItem := m.instanceList.SelectedItem().(ec2InstanceItem)
					selectedInstance := selectedItem.instance
					if *selectedInstance.State.Name == ec2.InstanceStateNameRunning {
						m.confirming = true
						m.action = "stop"
						m.actionID = selectedInstance.InstanceId
						m.status = fmt.Sprintf("Confirm stopping instance %s (%s)? (y/N)",
							getInstanceName(selectedInstance), *selectedInstance.InstanceId)
					} else {
						m.status = fmt.Sprintf("Instance %s is not running. Cannot stop.", getInstanceName(selectedInstance))
					}
				}
			case key.Matches(msg, m.keys.start):
				if m.instanceList.SelectedItem() != nil {
					selectedItem := m.instanceList.SelectedItem().(ec2InstanceItem)
					selectedInstance := selectedItem.instance
					if *selectedInstance.State.Name == ec2.InstanceStateNameStopped {
						m.confirming = true
						m.action = "start"
						m.actionID = selectedInstance.InstanceId
						m.status = fmt.Sprintf("Confirm starting instance %s (%s)? (y/N)",
							getInstanceName(selectedInstance), *selectedInstance.InstanceId)
					} else {
						m.status = fmt.Sprintf("Instance %s is not stopped. Cannot start.", getInstanceName(selectedInstance))
					}
				}
			case key.Matches(msg, m.keys.details): // View details
				if m.instanceList.SelectedItem() != nil {
					selectedItem := m.instanceList.SelectedItem().(ec2InstanceItem)
					selectedInstance := selectedItem.instance
					m.status = "Fetching instance details..."
					m.err = nil
					return m, tea.Batch(m.spinner.Tick, fetchInstanceDetailsCmd(m.ec2Svc, selectedInstance.InstanceId))
				}
			case key.Matches(msg, m.keys.ssh): // SSH into instance
				if m.instanceList.SelectedItem() != nil {
					selectedItem := m.instanceList.SelectedItem().(ec2InstanceItem)
					selectedInstance := selectedItem.instance
					publicIP := aws.StringValue(selectedInstance.PublicIpAddress)
					if publicIP == "" {
						m.status = "Selected instance has no public IP address for SSH."
						m.err = fmt.Errorf("no public IP for SSH")
					} else {
						m.status = fmt.Sprintf("Attempting to SSH into %s (%s)...", getInstanceName(selectedInstance), publicIP)
						m.err = nil
						return m, sshIntoInstanceCmd(publicIP, aws.StringValue(selectedInstance.KeyName))
					}
				}
			}
		case stateECSClusters:
			if m.clusterList.FilterState() == list.Filtering {
				break
			}
			switch {
			case key.Matches(msg, m.keys.refresh):
				m.status = "Refreshing ECS clusters..."
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, fetchECSClustersCmd(m.ecsSvc))
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select cluster"))):
				if m.clusterList.SelectedItem() != nil {
					selectedItem := m.clusterList.SelectedItem().(ecsClusterItem)
					m.detailCluster = selectedItem.cluster
					m.state = stateECSServices
					m.status = fmt.Sprintf("Loading services for cluster %s...", aws.StringValue(m.detailCluster.ClusterName))
					return m, tea.Batch(m.spinner.Tick, fetchECSServicesCmd(m.ecsSvc, aws.StringValue(m.detailCluster.ClusterArn)))
				}
			}
		case stateECSServices: // New state for ECS Services
			if m.serviceList.FilterState() == list.Filtering {
				break
			}
			switch {
			case key.Matches(msg, m.keys.refresh):
				m.status = fmt.Sprintf("Refreshing services for cluster %s...", aws.StringValue(m.detailCluster.ClusterName))
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, fetchECSServicesCmd(m.ecsSvc, aws.StringValue(m.detailCluster.ClusterArn)))
			}
		}

		// Global keys (e.g., quit, back to menu/previous view)
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q/ctrl+c", "quit"))):
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("backspace", "esc"), key.WithHelp("backspace/esc", "back"))):
			if m.state == stateEC2Instances || m.state == stateECSClusters {
				m.state = stateMenu
				m.status = "Select an option."
				m.err = nil
				m.confirming = false
				m.showDetails = false
				m.detailInstance = nil
				m.detailCluster = nil // Clear detail cluster when going back to menu
				return m, nil
			} else if m.state == stateECSServices {
				m.state = stateECSClusters
				m.status = "Ready"
				m.err = nil
				m.serviceList.SetItems([]list.Item{}) // Clear services when going back to clusters
				return m, nil
			}
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case instancesFetchedMsg:
		// Instances fetched successfully, update the list items
		listItems := make([]list.Item, len(msg))
		for i, instance := range msg {
			listItems[i] = ec2InstanceItem{instance: instance}
		}
		m.instanceList.SetItems(listItems)
		m.status = "Ready"
		m.err = nil
		return m, nil

	case ecsClustersFetchedMsg:
		// ECS Clusters fetched successfully, update the list items
		listItems := make([]list.Item, len(msg))
		for i, cluster := range msg {
			listItems[i] = ecsClusterItem{cluster: cluster}
		}
		m.clusterList.SetItems(listItems)
		m.status = "Ready"
		m.err = nil
		return m, nil

	case ecsServicesFetchedMsg: // New: Handle fetched ECS services
		listItems := make([]list.Item, len(msg))
		for i, service := range msg {
			listItems[i] = ecsServiceItem{service: service}
		}
		m.serviceList.SetItems(listItems)
		m.status = "Ready"
		m.err = nil
		return m, nil

	case instanceActionMsg:
		// Instance action completed, refresh the list
		m.status = fmt.Sprintf("Instance %s %s. Refreshing...", *m.actionID, msg)
		m.err = nil
		m.action = ""
		m.actionID = nil
		return m, tea.Batch(m.spinner.Tick, fetchInstancesCmd(m.ec2Svc))

	case instanceDetailsMsg: // Instance details fetched
		m.detailInstance = msg
		m.showDetails = true
		m.status = "Ready"
		m.err = nil
		return m, nil

	case sshExitMsg: // SSH command finished
		if msg.err != nil {
			m.err = fmt.Errorf("SSH command failed: %s", msg.err)
			m.status = "SSH Failed"
		} else {
			m.status = "SSH session ended."
			m.err = nil // Clear any previous error
		}
		// After SSH, refresh the instance list to get latest state
		return m, tea.Batch(m.spinner.Tick, fetchInstancesCmd(m.ec2Svc))

	case errMsg:
		// An error occurred
		m.err = msg
		m.status = "Error"
		m.confirming = false // Reset confirmation state on error
		m.action = ""
		m.actionID = nil
		m.showDetails = false // Exit detail view on error
		m.detailInstance = nil
		return m, nil
	}

	// Pass messages to the appropriate list component based on current state
	if m.state == stateEC2Instances {
		m.instanceList, cmd = m.instanceList.Update(msg)
	} else if m.state == stateECSClusters {
		m.clusterList, cmd = m.clusterList.Update(msg)
	} else if m.state == stateECSServices { // New: Update service list
		m.serviceList, cmd = m.serviceList.Update(msg)
	}
	return m, cmd
}

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
			cursor := ""
			if m.menuCursor == i {
				s.WriteString(fmt.Sprintf("%s %s\n", cursor, selectedItemStyle.Render(choice)))
			} else {
				s.WriteString(fmt.Sprintf("%s %s\n", cursor, menuItemStyle.Render(choice)))
			}
		}
		s.WriteString(statusStyle.Render("\n(Press 'q' or 'ctrl+c' to quit)\n"))

	case stateEC2Instances:
		if m.showDetails {
			// Render instance details view
			if m.detailInstance != nil {
				s.WriteString("\n" + detailStyle.Render(
					fmt.Sprintf("Instance ID:   %s\n", aws.StringValue(m.detailInstance.InstanceId))+
						fmt.Sprintf("Name:          %s\n", getInstanceName(m.detailInstance))+
						fmt.Sprintf("State:         %s\n", aws.StringValue(m.detailInstance.State.Name))+
						fmt.Sprintf("Type:          %s\n", aws.StringValue(m.detailInstance.InstanceType))+
						fmt.Sprintf("Launch Time:   %s\n", aws.TimeValue(m.detailInstance.LaunchTime).Format(time.RFC822))+
						fmt.Sprintf("Public IP:     %s\n", aws.StringValue(m.detailInstance.PublicIpAddress))+
						fmt.Sprintf("Private IP:    %s\n", aws.StringValue(m.detailInstance.PrivateIpAddress))+
						fmt.Sprintf("Availability Zone: %s\n", aws.StringValue(m.detailInstance.Placement.AvailabilityZone))+
						fmt.Sprintf("VPC ID:        %s\n", aws.StringValue(m.detailInstance.VpcId))+
						fmt.Sprintf("Subnet ID:     %s\n", aws.StringValue(m.detailInstance.SubnetId))+
						"\nPress 'esc' or 'backspace' to go back."+
						"\n"+statusStyle.Render(fmt.Sprintf("Status: %s", m.status)),
				))
			} else {
				s.WriteString(statusStyle.Render("No details available.\n"))
			}
		} else {
			// Render instance list view using bubbles/list
			if len(m.instanceList.Items()) == 0 && m.status == "Ready" {
				s.WriteString(statusStyle.Render("No EC2 instances found in this region.\n"))
			} else {
				s.WriteString(m.instanceList.View())
			}

			if m.status != "Ready" && m.status != "Error" {
				s.WriteString(statusStyle.Render(fmt.Sprintf("\n%s %s", m.spinner.View(), m.status)))
			} else if m.confirming {
				s.WriteString(confirmStyle.Render(fmt.Sprintf("\n%s", m.status)))
			} else {
				s.WriteString(statusStyle.Render(fmt.Sprintf("\nStatus: %s", m.status)))
			}
		}
	case stateECSClusters:
		// Render ECS cluster list view
		if len(m.clusterList.Items()) == 0 && m.status == "Ready" {
			s.WriteString(statusStyle.Render("No ECS clusters found in this region.\n"))
		} else {
			s.WriteString(m.clusterList.View())
		}

		if m.status != "Ready" && m.status != "Error" {
			s.WriteString(statusStyle.Render(fmt.Sprintf("\n%s %s", m.spinner.View(), m.status)))
		} else {
			s.WriteString(statusStyle.Render(fmt.Sprintf("\nStatus: %s", m.status)))
		}
	case stateECSServices: // New: Render ECS service list view
		s.WriteString(headerStyle.Render(fmt.Sprintf("ECS Services in Cluster: %s\n", aws.StringValue(m.detailCluster.ClusterName))))
		if len(m.serviceList.Items()) == 0 && m.status == "Ready" {
			s.WriteString(statusStyle.Render("No ECS services found in this cluster.\n"))
		} else {
			s.WriteString(m.serviceList.View())
		}

		if m.status != "Ready" && m.status != "Error" {
			s.WriteString(statusStyle.Render(fmt.Sprintf("\n%s %s", m.spinner.View(), m.status)))
		} else {
			s.WriteString(statusStyle.Render(fmt.Sprintf("\nStatus: %s", m.status)))
		}
	}

	return appStyle.Render(s.String())
}

// fetchInstancesCmd fetches EC2 instances from AWS.
func fetchInstancesCmd(svc *ec2.EC2) tea.Cmd {
	return func() tea.Msg {
		// Describe all instances
		result, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{})
		if err != nil {
			return errMsg(fmt.Errorf("failed to describe instances: %w", err))
		}

		var instances []*ec2.Instance
		for _, reservation := range result.Reservations {
			for _, instance := range reservation.Instances {
				// Filter out terminated instances for a cleaner view
				if *instance.State.Name != ec2.InstanceStateNameTerminated {
					instances = append(instances, instance)
				}
			}
		}
		return instancesFetchedMsg(instances)
	}
}

// fetchInstanceDetailsCmd fetches details for a specific EC2 instance.
func fetchInstanceDetailsCmd(svc *ec2.EC2, instanceID *string) tea.Cmd {
	return func() tea.Msg {
		result, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
			InstanceIds: []*string{instanceID},
		})
		if err != nil {
			return errMsg(fmt.Errorf("failed to describe instance %s details: %w", *instanceID, err))
		}

		if len(result.Reservations) > 0 && len(result.Reservations[0].Instances) > 0 {
			return instanceDetailsMsg(result.Reservations[0].Instances[0])
		}
		return errMsg(fmt.Errorf("instance %s not found", *instanceID))
	}
}

// stopInstanceCmd stops a specific EC2 instance.
func stopInstanceCmd(svc *ec2.EC2, instanceID *string) tea.Cmd {
	return func() tea.Msg {
		_, err := svc.StopInstances(&ec2.StopInstancesInput{
			InstanceIds: []*string{instanceID},
		})
		if err != nil {
			return errMsg(fmt.Errorf("failed to stop instance %s: %w", *instanceID, err))
		}
		// Wait a bit for the state to propagate before refreshing
		time.Sleep(2 * time.Second)
		return instanceActionMsg("stopped")
	}
}

// startInstanceCmd starts a specific EC2 instance.
func startInstanceCmd(svc *ec2.EC2, instanceID *string) tea.Cmd {
	return func() tea.Msg {
		_, err := svc.StartInstances(&ec2.StartInstancesInput{
			InstanceIds: []*string{instanceID},
		})
		if err != nil {
			return errMsg(fmt.Errorf("failed to start instance %s: %w", *instanceID, err))
		}
		// Wait a bit for the state to propagate before refreshing
		time.Sleep(2 * time.Second)
		return instanceActionMsg("started")
	}
}

// fetchECSClustersCmd fetches ECS clusters from AWS.
func fetchECSClustersCmd(svc *ecs.ECS) tea.Cmd {
	return func() tea.Msg {
		// List all cluster ARNs
		listResult, err := svc.ListClusters(&ecs.ListClustersInput{})
		if err != nil {
			return errMsg(fmt.Errorf("failed to list ECS clusters: %w", err))
		}

		if len(listResult.ClusterArns) == 0 {
			return ecsClustersFetchedMsg([]*ecs.Cluster{})
		}

		// Describe clusters to get detailed information
		describeResult, err := svc.DescribeClusters(&ecs.DescribeClustersInput{
			Clusters: listResult.ClusterArns,
		})
		if err != nil {
			return errMsg(fmt.Errorf("failed to describe ECS clusters: %w", err))
		}

		return ecsClustersFetchedMsg(describeResult.Clusters)
	}
}

// fetchECSServicesCmd fetches ECS services for a given cluster ARN.
func fetchECSServicesCmd(svc *ecs.ECS, clusterArn string) tea.Cmd {
	return func() tea.Msg {
		// List all service ARNs in the cluster
		listResult, err := svc.ListServices(&ecs.ListServicesInput{
			Cluster: aws.String(clusterArn),
		})
		if err != nil {
			return errMsg(fmt.Errorf("failed to list ECS services for cluster %s: %w", clusterArn, err))
		}

		if len(listResult.ServiceArns) == 0 {
			return ecsServicesFetchedMsg([]*ecs.Service{})
		}

		// Describe services to get detailed information
		describeResult, err := svc.DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  aws.String(clusterArn),
			Services: listResult.ServiceArns,
		})
		if err != nil {
			return errMsg(fmt.Errorf("failed to describe ECS services for cluster %s: %w", clusterArn, err))
		}

		return ecsServicesFetchedMsg(describeResult.Services)
	}
}

type sshExitMsg struct {
	err error
}

// sshIntoInstanceCmd executes an SSH command to connect to the given IP.
func sshIntoInstanceCmd(publicIP string, keyName string) tea.Cmd {
	return tea.ExecProcess(exec.Command("ssh", "-i", "~/.ssh/"+keyName+".pem", "ec2-user@"+publicIP), func(err error) tea.Msg { return sshExitMsg{err: err} })
}

// getInstanceName extracts the "Name" tag from an EC2 instance.
func getInstanceName(instance *ec2.Instance) string {
	for _, tag := range instance.Tags {
		if aws.StringValue(tag.Key) == "Name" {
			return aws.StringValue(tag.Value)
		}
	}
	return "N/A"
}

// getSecurityGroupNames extracts security group names from an EC2 instance.
func getSecurityGroupNames(sgs []*ec2.SecurityGroupIdentifier) string {
	var names []string
	for _, sg := range sgs {
		names = append(names, aws.StringValue(sg.GroupName))
	}
	if len(names) == 0 {
		return "N/A"
	}
	return strings.Join(names, ", ")
}

type listKeyMap struct {
	details key.Binding
	start   key.Binding
	stop    key.Binding
	ssh     key.Binding
	refresh key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		details: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "details"),
		),
		stop: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "stop"),
		),
		start: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "start"),
		),
		ssh: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "ssh"),
		),
		refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
	}
}

func main() {
	// Initialize AWS session
	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}

	// Create EC2 service client
	ec2Svc := ec2.New(sess)
	// Create ECS service client
	ecsSvc := ecs.New(sess)

	// Initialize the spinner model
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(tokyoNightCyan) // Purple color for spinner

	var listkeys = newListKeyMap()

	// Initialize the EC2 list model
	ec2List := list.New([]list.Item{}, itemDelegate{}, 0, 20) // Width and height will be set by Bubble Tea
	ec2List.Title = "EC2 Instances"
	ec2List.SetShowStatusBar(false)
	ec2List.SetFilteringEnabled(true)
	ec2List.Styles.Title = headerStyle
	ec2List.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(tokyoNightGreen)
	ec2List.Styles.FilterCursor = lipgloss.NewStyle().Foreground(tokyoNightGreen)
	ec2List.Styles.NoItems = statusStyle.UnsetPaddingLeft()
	ec2List.SetStatusBarItemName("instance", "instances")
	ec2List.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.details,
			listkeys.start,
			listkeys.stop,
			listkeys.ssh,
			listkeys.refresh,
		}

	}

	// Initialize the ECS cluster list model
	ecsClusterList := list.New([]list.Item{}, itemDelegate{}, 0, 20) // Width and height will be set by Bubble Tea
	ecsClusterList.Title = "ECS Clusters"
	ecsClusterList.SetShowStatusBar(false)
	ecsClusterList.SetFilteringEnabled(true)
	ecsClusterList.Styles.Title = headerStyle
	ecsClusterList.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(tokyoNightGreen)
	ecsClusterList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(tokyoNightGreen)
	ecsClusterList.Styles.NoItems = statusStyle.UnsetPaddingLeft()
	ecsClusterList.SetStatusBarItemName("cluster", "clusters")
	ecsClusterList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.refresh, // Only refresh for now, add more ECS specific keys later
		}
	}

	// Initialize the ECS service list model
	ecsServiceList := list.New([]list.Item{}, itemDelegate{}, 0, 20) // Width and height will be set by Bubble Tea
	ecsServiceList.Title = "ECS Services"
	ecsServiceList.SetShowStatusBar(false)
	ecsServiceList.SetFilteringEnabled(true)
	ecsServiceList.Styles.Title = headerStyle
	ecsServiceList.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(tokyoNightGreen)
	ecsServiceList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(tokyoNightGreen)
	ecsServiceList.Styles.NoItems = statusStyle.UnsetPaddingLeft()
	ecsServiceList.SetStatusBarItemName("service", "services")
	ecsServiceList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.refresh, // Only refresh for now, add more ECS specific keys later
		}
	}

	// Create initial model
	m := model{
		ec2Svc:       ec2Svc,
		ecsSvc:       ecsSvc, // Assign ECS service client
		status:       "Select an option.",
		spinner:      s,
		instanceList: ec2List,
		clusterList:  ecsClusterList, // Assign ECS cluster list model
		serviceList:  ecsServiceList, // Assign ECS service list model
		keys:         listkeys,
		state:        stateMenu, // Start in the menu state
		menuChoices:  []string{"EC2 Instances", "ECS Clusters"},
		menuCursor:   0,
	}

	// Start the Bubble Tea program
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
}
