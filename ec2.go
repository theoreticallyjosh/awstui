package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type ec2Model struct {
	ec2Svc         *ec2.EC2
	instanceList   list.Model
	status         string
	err            error
	spinner        spinner.Model
	confirming     bool
	action         string
	actionID       *string
	showDetails    bool
	detailInstance *ec2.Instance
	keys           *listKeyMap
}

func (m ec2Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchInstancesCmd(m.ec2Svc))
}

func (m ec2Model) Update(msg tea.Msg) (ec2Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.instanceList.SetSize(msg.Width-2*h, msg.Height-2*v)
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
		case key.Matches(msg, m.keys.details):
			if m.instanceList.SelectedItem() != nil {
				selectedItem := m.instanceList.SelectedItem().(ec2InstanceItem)
				selectedInstance := selectedItem.instance
				m.status = "Fetching instance details..."
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, fetchInstanceDetailsCmd(m.ec2Svc, selectedInstance.InstanceId))
			}
		case key.Matches(msg, m.keys.ssh):
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
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case instancesFetchedMsg:
		listItems := make([]list.Item, len(msg))
		for i, instance := range msg {
			listItems[i] = ec2InstanceItem{instance: instance}
		}
		m.instanceList.SetItems(listItems)
		m.status = "Ready"
		m.err = nil
		return m, nil
	case instanceActionMsg:
		m.status = fmt.Sprintf("Instance %s %s. Refreshing...", *m.actionID, msg)
		m.err = nil
		m.action = ""
		m.actionID = nil
		return m, tea.Batch(m.spinner.Tick, fetchInstancesCmd(m.ec2Svc))
	case instanceDetailsMsg:
		m.detailInstance = msg
		m.showDetails = true
		m.status = "Ready"
		m.err = nil
		return m, nil
	case sshExitMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("SSH command failed: %s", msg.err)
			m.status = "SSH Failed"
		} else {
			m.status = "SSH session ended."
			m.err = nil
		}
		return m, tea.Batch(m.spinner.Tick, fetchInstancesCmd(m.ec2Svc))
	case errMsg:
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
			return detailStyle.Render(
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
					"\nPress 'esc' or 'backspace' to go back." +
					"\n" + statusStyle.Render(fmt.Sprintf("Status: %s", m.status)),
			)
		}
		return statusStyle.Render("No details available.\n")
	}

	var s string
	if len(m.instanceList.Items()) == 0 && m.status == "Ready" {
		s = statusStyle.Render("No EC2 instances found in this region.\n")
	} else {
		s = m.instanceList.View()
	}

	if m.status != "Ready" && m.status != "Error" {
		s += statusStyle.Render(fmt.Sprintf("\n%s %s", m.spinner.View(), m.status))
	} else if m.confirming {
		s += confirmStyle.Render(fmt.Sprintf("\n%s", m.status))
	} else {
		s += statusStyle.Render(fmt.Sprintf("\nStatus: %s", m.status))
	}
	return s
}
