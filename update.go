package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles incoming messages and updates the model's state.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.instanceList.SetSize(msg.Width-2*h, msg.Height-2*v)
		m.clusterList.SetSize(msg.Width-2*h, msg.Height-2*v)
		m.serviceList.SetSize(msg.Width-2*h, msg.Height-2*v)
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

		// Handle confirmation for ECS service actions
		if m.state == stateECSServiceConfirmAction {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.status = fmt.Sprintf("%sing service %s...", m.action, aws.StringValue(m.ecsServiceActionService.ServiceName))
				m.err = nil
				if m.action == "stop" {
					return m, tea.Batch(m.spinner.Tick, stopECSServiceCmd(m.ecsSvc, aws.StringValue(m.detailCluster.ClusterArn), aws.StringValue(m.ecsServiceActionService.ServiceArn)))
				}
			case "n", "N":
				m.confirming = false
				m.status = "Action cancelled."
				m.action = ""
				m.ecsServiceActionService = nil
			}
			return m, nil
		}

		// Handle keys specific to EC2, ECS clusters, ECS services, or ECS service details
		switch m.state {
		case stateEC2Instances:
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
		case stateECSServices:
			if m.serviceList.FilterState() == list.Filtering {
				break
			}
			switch {
			case key.Matches(msg, m.keys.refresh):
				m.status = fmt.Sprintf("Refreshing services for cluster %s...", aws.StringValue(m.detailCluster.ClusterName))
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, fetchECSServicesCmd(m.ecsSvc, aws.StringValue(m.detailCluster.ClusterArn)))
			case key.Matches(msg, m.keys.details):
				if m.serviceList.SelectedItem() != nil {
					selectedItem := m.serviceList.SelectedItem().(ecsServiceItem)
					m.detailService = selectedItem.service
					m.state = stateECSServiceDetails
					m.status = "Showing service details."
					m.err = nil
				}
			case key.Matches(msg, m.keys.stop):
				if m.serviceList.SelectedItem() != nil {
					selectedItem := m.serviceList.SelectedItem().(ecsServiceItem)
					selectedService := selectedItem.service
					if aws.Int64Value(selectedService.DesiredCount) > 0 {
						m.confirming = true
						m.action = "stop"
						m.ecsServiceActionService = selectedService
						m.state = stateECSServiceConfirmAction
						m.status = fmt.Sprintf("Confirm stopping service %s (Desired: %d)? (y/N)",
							aws.StringValue(selectedService.ServiceName), aws.Int64Value(selectedService.DesiredCount))
					} else {
						m.status = fmt.Sprintf("Service %s is already stopped (Desired: 0).", aws.StringValue(selectedService.ServiceName))
					}
				}
			case key.Matches(msg, m.keys.logs):
				if m.serviceList.SelectedItem() != nil {
					selectedItem := m.serviceList.SelectedItem().(ecsServiceItem)
					m.detailService = selectedItem.service
					m.state = stateECSServiceLogs
					m.status = fmt.Sprintf("Fetching logs for service %s...", aws.StringValue(selectedItem.service.ServiceName))
					return m, tea.Batch(m.spinner.Tick, fetchECSServiceLogsCmd(m.ecsSvc, m.cloudwatchlogsSvc, selectedItem.service))
				}
			}
		case stateECSServiceDetails:
		case stateECSServiceLogs:
		}

		// Global keys (e.g., quit, back to menu/previous view)
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q/ctrl+c", "quit"))):
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("backspace", "esc"), key.WithHelp("backspace/esc", "back"))):
			if m.state == stateEC2Instances {
				m.state = stateMenu
				m.status = "Select an option."
				m.err = nil
				m.confirming = false
				m.showDetails = false
				m.detailInstance = nil
				return m, nil
			} else if m.state == stateECSClusters {
				m.state = stateMenu
				m.status = "Select an option."
				m.err = nil
				m.confirming = false
				m.detailCluster = nil
				m.clusterList.SetItems([]list.Item{})
				return m, nil
			} else if m.state == stateECSServices {
				m.state = stateECSClusters
				m.status = "Ready"
				m.err = nil
				m.serviceList.SetItems([]list.Item{})
				return m, nil
			} else if m.state == stateECSServiceDetails {
				m.state = stateECSServices
				m.status = "Ready"
				m.err = nil
				m.detailService = nil
				return m, nil
			} else if m.state == stateECSServiceConfirmAction {
				m.state = stateECSServices
				m.confirming = false
				m.action = ""
				m.ecsServiceActionService = nil
				m.status = "Action cancelled."
				return m, nil
			} else if m.state == stateECSServiceLogs {
				m.state = stateECSServices
				m.status = "Ready"
				m.err = nil
				m.serviceLogs = ""
				return m, nil
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

	case ecsClustersFetchedMsg:
		listItems := make([]list.Item, len(msg))
		for i, cluster := range msg {
			listItems[i] = ecsClusterItem{cluster: cluster}
		}
		m.clusterList.SetItems(listItems)
		m.status = "Ready"
		m.err = nil
		return m, nil

	case ecsServicesFetchedMsg:
		listItems := make([]list.Item, len(msg))
		for i, service := range msg {
			listItems[i] = ecsServiceItem{service: service}
		}
		m.serviceList.SetItems(listItems)
		m.status = "Ready"
		m.err = nil
		return m, nil

	case ecsServiceDetailsMsg:
		m.detailService = msg
		m.state = stateECSServiceDetails
		m.status = "Ready"
		m.err = nil
		return m, nil

	case ecsServiceActionMsg:
		m.status = fmt.Sprintf("Service %s %s. Refreshing...", aws.StringValue(m.ecsServiceActionService.ServiceName), msg)
		m.err = nil
		m.action = ""
		m.ecsServiceActionService = nil
		m.confirming = false
		return m, tea.Batch(m.spinner.Tick, fetchECSServicesCmd(m.ecsSvc, aws.StringValue(m.detailCluster.ClusterArn)))

	case ecsServiceLogsFetchedMsg:
		m.serviceLogs = string(msg)
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
		m.detailService = nil
		m.serviceLogs = ""
		return m, nil
	}

	if m.state == stateEC2Instances {
		m.instanceList, cmd = m.instanceList.Update(msg)
	} else if m.state == stateECSClusters {
		m.clusterList, cmd = m.clusterList.Update(msg)
	} else if m.state == stateECSServices {
		m.serviceList, cmd = m.serviceList.Update(msg)
	}
	return m, cmd
}
