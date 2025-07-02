package main

import (
	"fmt"
	"log"
	"os/exec" // Import os/exec
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Define styles using lipgloss
var (
	// Base style for the entire application
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	// Style for headers
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true).
			PaddingBottom(1)

	// Style for instance details
	instanceStyle = lipgloss.NewStyle().PaddingLeft(2)

	// Style for selected instance
	selectedInstanceStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true).
				PaddingLeft(1)

	// Style for status messages (e.g., loading, success)
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			PaddingTop(1)

	// Style for error messages
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true).
			PaddingTop(1)

	// Style for confirmation prompts
	confirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true).
			PaddingTop(1)

	// Style for detail view
	detailStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)
)

// model represents the state of our TUI application.
type model struct {
	ec2Svc         *ec2.EC2        // AWS EC2 service client
	instances      []*ec2.Instance // List of EC2 instances
	cursor         int             // Cursor position in the instance list
	status         string          // Current status message (e.g., "Loading...", "Ready")
	err            error           // Any error encountered
	spinner        spinner.Model   // Spinner for loading states
	confirming     bool            // True if waiting for user confirmation
	action         string          // The action being confirmed ("stop" or "start")
	actionID       *string         // The ID of the instance for the pending action
	showDetails    bool            // True if showing instance details
	detailInstance *ec2.Instance   // The instance whose details are currently displayed
}

// messages are used to pass data between commands and the Update function.
type (
	instancesFetchedMsg []*ec2.Instance // Message when instances are fetched
	instanceActionMsg   string          // Message when an instance action (stop/start) is completed
	instanceDetailsMsg  *ec2.Instance   // Message when instance details are fetched
	sshFinishedMsg      error           // Message when SSH command finishes (nil for success, error for failure)
	errMsg              error           // Message for errors
)

// Init initializes the model and starts fetching EC2 instances.
func (m model) Init() tea.Cmd {
	// Start the spinner animation
	return tea.Batch(m.spinner.Tick, fetchInstancesCmd(m.ec2Svc))
}

// Update handles incoming messages and updates the model's state.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			// Exit the application
			return m, tea.Quit
		case "up", "k":
			// Move cursor up
			if !m.confirming && !m.showDetails {
				if m.cursor > 0 {
					m.cursor--
				}
			}
		case "down", "j":
			// Move cursor down
			if !m.confirming && !m.showDetails {
				if m.cursor < len(m.instances)-1 {
					m.cursor++
				}
			}
		case "r":
			// Refresh instances
			if !m.confirming && !m.showDetails {
				m.status = "Refreshing instances..."
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, fetchInstancesCmd(m.ec2Svc))
			}
		case "s":
			// Stop selected instance
			if !m.confirming && !m.showDetails && len(m.instances) > 0 {
				selectedInstance := m.instances[m.cursor]
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
		case "t":
			// Start selected instance
			if !m.confirming && !m.showDetails && len(m.instances) > 0 {
				selectedInstance := m.instances[m.cursor]
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
		case "d": // View details
			if !m.confirming && !m.showDetails && len(m.instances) > 0 {
				selectedInstance := m.instances[m.cursor]
				m.status = "Fetching instance details..."
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, fetchInstanceDetailsCmd(m.ec2Svc, selectedInstance.InstanceId))
			}
		case "h": // New: SSH into instance
			if !m.confirming && !m.showDetails && len(m.instances) > 0 {
				selectedInstance := m.instances[m.cursor]
				publicIP := aws.StringValue(selectedInstance.PublicIpAddress)
				keyName := *selectedInstance.KeyName
				if publicIP == "" {
					m.status = "Selected instance has no public IP address for SSH."
					m.err = fmt.Errorf("no public IP for SSH")
				} else {
					m.status = fmt.Sprintf("Attempting to SSH into %s (%s)...", getInstanceName(selectedInstance), publicIP)
					m.err = nil
					return m, sshIntoInstanceCmd(publicIP, keyName)
				}
			}
		case "esc", "backspace": // Exit detail view
			if m.showDetails {
				m.showDetails = false
				m.detailInstance = nil
				m.status = "Ready"
				m.err = nil
			}
		case "y", "Y":
			// Confirm action
			if m.confirming {
				m.confirming = false
				m.status = fmt.Sprintf("%sing instance %s...", m.action, *m.actionID)
				m.err = nil
				if m.action == "stop" {
					return m, tea.Batch(m.spinner.Tick, stopInstanceCmd(m.ec2Svc, m.actionID))
				} else if m.action == "start" {
					return m, tea.Batch(m.spinner.Tick, startInstanceCmd(m.ec2Svc, m.actionID))
				}
			}
		case "n", "N":
			// Deny action
			if m.confirming {
				m.confirming = false
				m.status = "Action cancelled."
				m.action = ""
				m.actionID = nil
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case instancesFetchedMsg:
		// Instances fetched successfully
		m.instances = msg
		m.status = "Ready"
		m.err = nil
		if len(m.instances) > 0 && m.cursor >= len(m.instances) {
			m.cursor = len(m.instances) - 1 // Adjust cursor if list shrunk
		} else if len(m.instances) == 0 {
			m.cursor = 0
		}
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

	case sshFinishedMsg: // New: SSH command finished
		if msg != nil {
			m.err = fmt.Errorf("SSH command failed: %w", msg)
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

	return m, nil
}

// View renders the TUI.
func (m model) View() string {
	var s strings.Builder

	s.WriteString(headerStyle.Render("EC2 Instance Manager\n"))
	s.WriteString("Use ↑↓ to navigate, 's' to stop, 't' to start, 'r' to refresh, 'd' for details, 'h' to SSH, 'q' or 'ctrl+c' to quit.\n\n") // Updated instructions

	if m.err != nil {
		s.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v\n", m.err)))
	}

	if m.showDetails {
		// Render instance details view
		if m.detailInstance != nil {
			s.WriteString(detailStyle.Render(
				fmt.Sprintf("Instance ID:   %s\n", aws.StringValue(m.detailInstance.InstanceId)) +
					fmt.Sprintf("Name:          %s\n", getInstanceName(m.detailInstance)) +
					fmt.Sprintf("State:         %s\n", aws.StringValue(m.detailInstance.State.Name)) +
					fmt.Sprintf("Type:          %s\n", aws.StringValue(m.detailInstance.InstanceType)) +
					fmt.Sprintf("Launch Time:   %s\n", aws.TimeValue(m.detailInstance.LaunchTime).Format(time.RFC822)) +
					fmt.Sprintf("Public IP:     %s\n", aws.StringValue(m.detailInstance.PublicIpAddress)) +
					fmt.Sprintf("Private IP:    %s\n", aws.StringValue(m.detailInstance.PrivateIpAddress)) +
					fmt.Sprintf("Availability Zone: %s\n", aws.StringValue(m.detailInstance.Placement.AvailabilityZone)) +
					fmt.Sprintf("VPC ID:        %s\n", aws.StringValue(m.detailInstance.VpcId)) +
					fmt.Sprintf("Subnet ID:     %s\n", aws.StringValue(m.detailInstance.SubnetId)) +
					// fmt.Sprintf("Security Groups: %s\n", getSecurityGroupNames(m.detailInstance.SecurityGroups)) +
					"\nPress 'esc' or 'backspace' to go back." +
					"\n" + statusStyle.Render(fmt.Sprintf("Status: %s", m.status)),
			))
		} else {
			s.WriteString(statusStyle.Render("No details available.\n"))
		}
	} else {
		// Render instance list view
		if len(m.instances) == 0 && m.status == "Ready" {
			s.WriteString(statusStyle.Render("No EC2 instances found in this region.\n"))
		} else {
			for i, instance := range m.instances {
				name := getInstanceName(instance)
				line := fmt.Sprintf("%s %-20s %-15s %s",
					*instance.InstanceId,
					name,
					*instance.InstanceType,
					*instance.State.Name,
				)

				if i == m.cursor {
					s.WriteString(selectedInstanceStyle.Render("> " + line))
				} else {
					s.WriteString(instanceStyle.Render("  " + line))
				}
				s.WriteString("\n")
			}
		}

		if m.status != "Ready" && m.status != "Error" {
			s.WriteString(statusStyle.Render(fmt.Sprintf("\n%s %s", m.spinner.View(), m.status)))
		} else if m.confirming {
			s.WriteString(confirmStyle.Render(fmt.Sprintf("\n%s", m.status)))
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

// sshIntoInstanceCmd executes an SSH command to connect to the given IP.
func sshIntoInstanceCmd(publicIP string, keyName string) tea.Cmd {
	return tea.ExecProcess(exec.Command("ssh", "-i", "/home/jhagofsk/.ssh/"+keyName+".pem", "ec2-user@"+publicIP), nil)
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

func main() {
	// Initialize AWS session
	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}

	// Create EC2 service client
	svc := ec2.New(sess)

	// Initialize the spinner model
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63")) // Purple color for spinner

	// Create initial model
	m := model{
		ec2Svc:  svc,
		status:  "Loading instances...",
		spinner: s,
	}

	// Start the Bubble Tea program
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
}
