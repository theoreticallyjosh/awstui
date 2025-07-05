package models

import (
	"awstui/internal/commands"
	"awstui/internal/keys"
	"awstui/internal/messages"
	"awstui/internal/styles"
	"awstui/internal/utils"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type ec2Model struct {
	parent         *Model
	ec2Svc         *ec2.EC2
	instanceList   list.Model
	status         string
	err            error
	confirming     bool
	action         string
	actionID       *string
	showDetails    bool
	detailInstance *ec2.Instance
	keys           *keys.ListKeyMap
}

func (m ec2Model) Init() tea.Cmd {
	return tea.Batch(m.parent.spinner.Tick, commands.FetchInstancesCmd(m.ec2Svc))
}

func (m ec2Model) Update(msg tea.Msg) (ec2Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := styles.AppStyle.GetFrameSize()
		m.instanceList.SetSize(msg.Width-3*h, msg.Height-3*v)
		return m, nil
	case tea.KeyMsg:
		if m.instanceList.FilterState() == list.Filtering {
			break
		}
		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.status = fmt.Sprintf("%sing instance %s...", m.action, *m.actionID)
				m.err = nil
				if m.action == "stop" {
					return m, tea.Batch(m.parent.spinner.Tick, commands.StopInstanceCmd(m.ec2Svc, m.actionID))
				} else if m.action == "start" {
					return m, tea.Batch(m.parent.spinner.Tick, commands.StartInstanceCmd(m.ec2Svc, m.actionID))
				}
			case "n", "N":
				m.confirming = false
				m.status = "Action cancelled."
				m.action = ""
				m.actionID = nil
			}
			return m, nil
		}

		if m.showDetails {
			switch msg.String() {
			case "esc", "backspace":
				m.showDetails = false
				m.detailInstance = nil
				m.status = "Ready"
				m.err = nil
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Refresh):
			m.status = "Refreshing instances..."
			m.err = nil
			return m, tea.Batch(m.parent.spinner.Tick, commands.FetchInstancesCmd(m.ec2Svc))
		case key.Matches(msg, m.keys.Stop):
			if m.instanceList.SelectedItem() != nil {
				selectedItem := m.instanceList.SelectedItem().(ec2InstanceItem)
				selectedInstance := selectedItem.instance
				if *selectedInstance.State.Name == ec2.InstanceStateNameRunning {
					m.confirming = true
					m.action = "stop"
					m.actionID = selectedInstance.InstanceId
					m.status = fmt.Sprintf("Confirm stopping instance %s (%s)? (y/N)",
						utils.GetInstanceName(selectedInstance), *selectedInstance.InstanceId)
				} else {
					m.status = fmt.Sprintf("Instance %s is not running. Cannot stop.", utils.GetInstanceName(selectedInstance))
				}
			}
		case key.Matches(msg, m.keys.Start):
			if m.instanceList.SelectedItem() != nil {
				selectedItem := m.instanceList.SelectedItem().(ec2InstanceItem)
				selectedInstance := selectedItem.instance
				if *selectedInstance.State.Name == ec2.InstanceStateNameStopped {
					m.confirming = true
					m.action = "start"
					m.actionID = selectedInstance.InstanceId
					m.status = fmt.Sprintf("Confirm starting instance %s (%s)? (y/N)",
						utils.GetInstanceName(selectedInstance), *selectedInstance.InstanceId)
				} else {
					m.status = fmt.Sprintf("Instance %s is not stopped. Cannot start.", utils.GetInstanceName(selectedInstance))
				}
			}
		case key.Matches(msg, m.keys.Details):
			if m.instanceList.SelectedItem() != nil {
				selectedItem := m.instanceList.SelectedItem().(ec2InstanceItem)
				selectedInstance := selectedItem.instance
				m.status = "Fetching instance details..."
				m.err = nil
				return m, tea.Batch(m.parent.spinner.Tick, commands.FetchInstanceDetailsCmd(m.ec2Svc, selectedInstance.InstanceId))
			}
		case key.Matches(msg, m.keys.Ssh):
			if m.instanceList.SelectedItem() != nil {
				selectedItem := m.instanceList.SelectedItem().(ec2InstanceItem)
				selectedInstance := selectedItem.instance
				publicIP := aws.StringValue(selectedInstance.PublicIpAddress)
				if publicIP == "" {
					m.status = "Selected instance has no public IP address for SSH."
					m.err = fmt.Errorf("no public IP for SSH")
				} else {
					m.status = fmt.Sprintf("Attempting to SSH into %s (%s)...", utils.GetInstanceName(selectedInstance), publicIP)
					m.err = nil
					return m, tea.Sequence(tea.ClearScreen, commands.SshIntoInstanceCmd(publicIP, aws.StringValue(selectedInstance.KeyName)))
				}
			}
		}
	case messages.InstancesFetchedMsg:
		listItems := make([]list.Item, len(msg))
		for i, instance := range msg {
			listItems[i] = ec2InstanceItem{instance: instance}
		}
		m.instanceList.SetItems(listItems)
		m.status = "Ready"
		m.err = nil
		return m, nil
	case messages.InstanceActionMsg:
		m.status = fmt.Sprintf("Instance %s %s. Refreshing...", *m.actionID, msg)
		m.err = nil
		m.action = ""
		m.actionID = nil
		return m, tea.Batch(m.parent.spinner.Tick, commands.FetchInstancesCmd(m.ec2Svc))
	case messages.InstanceDetailsMsg:
		m.detailInstance = msg
		m.showDetails = true
		m.status = "Ready"
		m.err = nil
		return m, nil
	case messages.SshExitMsg:
		if msg.Err != nil {
			m.err = fmt.Errorf("SSH command failed: %s", msg.Err)
			m.status = "SSH Failed"
		} else {
			m.status = "SSH session ended."
			m.err = nil
		}
		return m, tea.Batch(m.parent.spinner.Tick, commands.FetchInstancesCmd(m.ec2Svc))
	case messages.ErrMsg:
		m.err = msg
		m.status = "Error"
		m.confirming = false
		m.action = ""
		m.actionID = nil
		m.showDetails = false
		m.detailInstance = nil
		return m, nil
	}
	m.instanceList, cmd = m.instanceList.Update(msg)
	return m, cmd
}

func (m ec2Model) View() string {
	if m.showDetails {
		if m.detailInstance != nil {
			return styles.DetailStyle.Render(
				fmt.Sprintf("Instance ID:   %s\n", aws.StringValue(m.detailInstance.InstanceId)) +
					fmt.Sprintf("Name:          %s\n", utils.GetInstanceName(m.detailInstance)) +
					fmt.Sprintf("State:         %s\n", aws.StringValue(m.detailInstance.State.Name)) +
					fmt.Sprintf("Type:          %s\n", aws.StringValue(m.detailInstance.InstanceType)) +
					fmt.Sprintf("Launch Time:   %s\n", aws.TimeValue(m.detailInstance.LaunchTime).Format(time.RFC822)) +
					fmt.Sprintf("Public IP:     %s\n", aws.StringValue(m.detailInstance.PublicIpAddress)) +
					fmt.Sprintf("Private IP:    %s\n", aws.StringValue(m.detailInstance.PrivateIpAddress)) +
					fmt.Sprintf("Availability Zone: %s\n", aws.StringValue(m.detailInstance.Placement.AvailabilityZone)) +
					fmt.Sprintf("VPC ID:        %s\n", aws.StringValue(m.detailInstance.VpcId)) +
					fmt.Sprintf("Subnet ID:     %s\n", aws.StringValue(m.detailInstance.SubnetId)) +
					"\nPress 'esc' or 'backspace' to go back." +
					"\n" + styles.StatusStyle.Render(fmt.Sprintf("Status: %s", m.status)),
			)
		}
		return styles.StatusStyle.Render("No details available.\n")
	}

	var s string
	if len(m.instanceList.Items()) == 0 && m.status == "Ready" {
		s = styles.StatusStyle.Render("No EC2 instances found in this region.\n")
	} else {
		s = m.instanceList.View()
	}

	return s
}
