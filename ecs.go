package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type ecsState int

const (
	ecsStateClusterList ecsState = iota
	ecsStateServiceList
	ecsStateServiceDetails
	ecsStateServiceConfirmAction
	ecsStateServiceLogs
)

type ecsModel struct {
	ecsSvc                  *ecs.ECS
	cloudwatchlogsSvc       *cloudwatchlogs.CloudWatchLogs
	clusterList             list.Model
	serviceList             list.Model
	status                  string
	err                     error
	spinner                 spinner.Model
	confirming              bool
	action                  string
	detailCluster           *ecs.Cluster
	detailService           *ecs.Service
	ecsServiceActionService *ecs.Service
	serviceLogs             string
	keys                    *listKeyMap
	state                   ecsState
}

func (m ecsModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchECSClustersCmd(m.ecsSvc))
}

func (m ecsModel) Update(msg tea.Msg) (ecsModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.clusterList.SetSize(msg.Width-2*h, msg.Height-2*v)
		m.serviceList.SetSize(msg.Width-2*h, msg.Height-2*v)
		return m, nil
	case tea.KeyMsg:
		if m.state == ecsStateServiceConfirmAction {
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

		switch m.state {
		case ecsStateClusterList:
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
					m.state = ecsStateServiceList
					m.status = fmt.Sprintf("Loading services for cluster %s...", aws.StringValue(m.detailCluster.ClusterName))
					return m, tea.Batch(m.spinner.Tick, fetchECSServicesCmd(m.ecsSvc, aws.StringValue(m.detailCluster.ClusterArn)))
				}
			}
		case ecsStateServiceList:
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
					m.state = ecsStateServiceDetails
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
						m.state = ecsStateServiceConfirmAction
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
					m.state = ecsStateServiceLogs
					m.status = fmt.Sprintf("Fetching logs for service %s...", aws.StringValue(selectedItem.service.ServiceName))
					return m, tea.Batch(m.spinner.Tick, fetchECSServiceLogsCmd(m.ecsSvc, m.cloudwatchlogsSvc, selectedItem.service))
				}
			}
		case ecsStateServiceDetails, ecsStateServiceLogs:
			// No key handling in these states for now
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("backspace", "esc"), key.WithHelp("backspace/esc", "back"))):
			if m.state == ecsStateServiceList {
				m.state = ecsStateClusterList
				m.status = "Ready"
				m.err = nil
				m.serviceList.SetItems([]list.Item{})
				return m, nil
			} else if m.state == ecsStateServiceDetails {
				m.state = ecsStateServiceList
				m.status = "Ready"
				m.err = nil
				m.detailService = nil
				return m, nil
			} else if m.state == ecsStateServiceConfirmAction {
				m.state = ecsStateServiceList
				m.confirming = false
				m.action = ""
				m.ecsServiceActionService = nil
				m.status = "Action cancelled."
				return m, nil
			} else if m.state == ecsStateServiceLogs {
				m.state = ecsStateServiceList
				m.status = "Ready"
				m.err = nil
				m.serviceLogs = ""
				return m, nil
			}
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
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
		m.state = ecsStateServiceDetails
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
	case errMsg:
		m.err = msg
		m.status = "Error"
		m.confirming = false
		m.action = ""
		m.detailService = nil
		m.serviceLogs = ""
		return m, nil
	}

	if m.state == ecsStateClusterList {
		m.clusterList, cmd = m.clusterList.Update(msg)
	} else if m.state == ecsStateServiceList {
		m.serviceList, cmd = m.serviceList.Update(msg)
	}
	return m, cmd
}

func (m ecsModel) View() string {
	var s string
	switch m.state {
	case ecsStateClusterList:
		if len(m.clusterList.Items()) == 0 && m.status == "Ready" {
			s = statusStyle.Render("No ECS clusters found in this region.\n")
		} else {
			s = m.clusterList.View()
		}
	case ecsStateServiceList:
		s = headerStyle.Render(fmt.Sprintf("ECS Services in Cluster: %s\n", aws.StringValue(m.detailCluster.ClusterName)))
		if len(m.serviceList.Items()) == 0 && m.status == "Ready" {
			s += statusStyle.Render("No ECS services found in this cluster.\n")
		} else {
			s += m.serviceList.View()
		}
	case ecsStateServiceDetails:
		if m.detailService != nil {
			s = headerStyle.Render(fmt.Sprintf("ECS Service Details: %s\n", aws.StringValue(m.detailService.ServiceName)))
			s += "\n" + detailStyle.Render(
				fmt.Sprintf("Service Name:  %s\n", aws.StringValue(m.detailService.ServiceName)) +
					fmt.Sprintf("Service ARN:   %s\n", aws.StringValue(m.detailService.ServiceArn)) +
					fmt.Sprintf("Status:        %s\n", aws.StringValue(m.detailService.Status)) +
					fmt.Sprintf("Desired Count: %d\n", aws.Int64Value(m.detailService.DesiredCount)) +
					fmt.Sprintf("Running Count: %d\n", aws.Int64Value(m.detailService.RunningCount)) +
					fmt.Sprintf("Pending Count: %d\n", aws.Int64Value(m.detailService.PendingCount)) +
					fmt.Sprintf("Launch Type:   %s\n", aws.StringValue(m.detailService.LaunchType)) +
					fmt.Sprintf("Task Definition: %s\n", aws.StringValue(m.detailService.TaskDefinition)) +
					fmt.Sprintf("Created At:    %s\n", aws.TimeValue(m.detailService.CreatedAt).Format(time.RFC822)) +
					"\nPress 'esc' or 'backspace' to go back." +
					"\n" + statusStyle.Render(fmt.Sprintf("Status: %s", m.status)),
			)
		} else {
			s = statusStyle.Render("No service details available.\n")
		}
	case ecsStateServiceConfirmAction:
		s = confirmStyle.Render(fmt.Sprintf("\n%s", m.status))
	case ecsStateServiceLogs:
		s = headerStyle.Render(fmt.Sprintf("Logs for Service: %s\n", aws.StringValue(m.detailService.ServiceName)))
		if m.serviceLogs == "" && m.status == "Ready" {
			s += statusStyle.Render("No logs found for this service.\n")
		} else {
			s += "\n" + detailStyle.Render(m.serviceLogs)
		}
		s += statusStyle.Render("\nPress 'esc' or 'backspace' to go back.")
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
