package models

import (
	"awstui/internal/commands"
	"awstui/internal/keys"
	"awstui/internal/messages"
	"awstui/internal/styles"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
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
	parent                  *Model
	ecsSvc                  *ecs.ECS
	cloudwatchlogsSvc       *cloudwatchlogs.CloudWatchLogs
	clusterList             list.Model
	serviceList             list.Model
	status                  string
	err                     error
	confirming              bool
	action                  string
	detailCluster           *ecs.Cluster
	detailService           *ecs.Service
	ecsServiceActionService *ecs.Service
	serviceLogs             string
	keys                    *keys.ListKeyMap
	paginator               paginator.Model
	state                   ecsState
	header                  []string
}

func (m ecsModel) Init() tea.Cmd {
	return tea.Batch(m.parent.spinner.Tick, commands.FetchECSClustersCmd(m.ecsSvc))
}

func (m ecsModel) Update(msg tea.Msg) (ecsModel, tea.Cmd) {
	var cmd tea.Cmd
	m.header = []string{"ECS Clusters"}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.paginator.PerPage = msg.Height - 2
		m.clusterList.SetSize(msg.Width, msg.Height)
		m.serviceList.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))):
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
				m.status = "Ready"
				return m, nil
			} else if m.state == ecsStateServiceLogs {
				m.state = ecsStateServiceList
				m.status = "Ready"
				m.err = nil
				m.serviceLogs = ""
				return m, nil
			}
		}
		if m.state == ecsStateServiceConfirmAction {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.status = fmt.Sprintf("%sing service %s...", m.action, aws.StringValue(m.ecsServiceActionService.ServiceName))
				m.err = nil
				if m.action == "stop" {
					return m, tea.Batch(m.parent.spinner.Tick, commands.StopECSServiceCmd(m.ecsSvc, aws.StringValue(m.detailCluster.ClusterArn), aws.StringValue(m.ecsServiceActionService.ServiceArn)))
				} else if m.action == "force-deploy" {
					return m, tea.Batch(m.parent.spinner.Tick, commands.ForceDeployECSServiceCmd(m.ecsSvc, aws.StringValue(m.detailCluster.ClusterArn), aws.StringValue(m.ecsServiceActionService.ServiceArn)))
				}
			case "n", "N":
				m.confirming = false
				m.status = "Ready"
				m.action = ""
				m.state = ecsStateServiceList
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
			case key.Matches(msg, m.keys.Refresh):
				m.status = "Refreshing ECS clusters..."
				m.err = nil
				return m, tea.Batch(m.parent.spinner.Tick, commands.FetchECSClustersCmd(m.ecsSvc))
			case key.Matches(msg, m.keys.Choose):
				if m.clusterList.SelectedItem() != nil {
					selectedItem := m.clusterList.SelectedItem().(ecsClusterItem)
					m.detailCluster = selectedItem.cluster
					m.state = ecsStateServiceList
					m.status = fmt.Sprintf("Loading services for cluster %s...", aws.StringValue(m.detailCluster.ClusterName))
					return m, tea.Batch(m.parent.spinner.Tick, commands.FetchECSServicesCmd(m.ecsSvc, aws.StringValue(m.detailCluster.ClusterArn)))
				}
			}
		case ecsStateServiceList:
			m.header = append(m.header, aws.StringValue(m.detailCluster.ClusterName), "Services")
			if m.serviceList.FilterState() == list.Filtering {
				break
			}
			switch {
			case key.Matches(msg, m.keys.Refresh):
				m.status = fmt.Sprintf("Refreshing services for cluster %s...", aws.StringValue(m.detailCluster.ClusterName))
				m.err = nil
				return m, tea.Batch(m.parent.spinner.Tick, commands.FetchECSServicesCmd(m.ecsSvc, aws.StringValue(m.detailCluster.ClusterArn)))
			case key.Matches(msg, m.keys.Details):
				if m.serviceList.SelectedItem() != nil {
					selectedItem := m.serviceList.SelectedItem().(ecsServiceItem)
					m.detailService = selectedItem.service
					m.state = ecsStateServiceDetails
					m.status = "Showing service details."
					m.err = nil
				}
			case key.Matches(msg, m.keys.Stop):
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
			case key.Matches(msg, m.keys.ForceDeploy):
				if m.serviceList.SelectedItem() != nil {
					selectedItem := m.serviceList.SelectedItem().(ecsServiceItem)
					selectedService := selectedItem.service
					m.confirming = true
					m.action = "force-deploy"
					m.ecsServiceActionService = selectedService
					m.state = ecsStateServiceConfirmAction
					m.status = fmt.Sprintf("Confirm force deployment of service %s? (y/N)",
						aws.StringValue(selectedService.ServiceName))
				}
			case key.Matches(msg, m.keys.Logs):
				if m.serviceList.SelectedItem() != nil {
					selectedItem := m.serviceList.SelectedItem().(ecsServiceItem)
					m.detailService = selectedItem.service
					m.state = ecsStateServiceLogs
					m.status = fmt.Sprintf("Fetching logs for service %s...", aws.StringValue(selectedItem.service.ServiceName))
					return m, tea.Batch(m.parent.spinner.Tick, commands.FetchECSServiceLogsCmd(m.ecsSvc, m.cloudwatchlogsSvc, selectedItem.service))
				}
			}
		case ecsStateServiceDetails:
			// No key handling in these states for now
		case ecsStateServiceLogs:
			m.header = append(m.header, aws.StringValue(m.detailCluster.ClusterName), m.serviceList.SelectedItem().FilterValue(), "Logs")
			m.paginator, cmd = m.paginator.Update(msg)
			return m, cmd
		}

	case messages.EcsClustersFetchedMsg:
		listItems := make([]list.Item, len(msg))
		for i, cluster := range msg {
			listItems[i] = ecsClusterItem{cluster: cluster}
		}
		m.clusterList.SetItems(listItems)
		m.status = "Ready"
		m.err = nil
		return m, nil
	case messages.EcsServicesFetchedMsg:
		m.header = append(m.header, m.clusterList.SelectedItem().FilterValue(), "Services")
		listItems := make([]list.Item, len(msg))
		for i, service := range msg {
			listItems[i] = ecsServiceItem{service: service}
		}
		m.serviceList.SetItems(listItems)
		m.status = "Ready"
		m.err = nil
		return m, nil
	case messages.EcsServiceDetailsMsg:
		m.detailService = msg
		m.state = ecsStateServiceDetails
		m.status = "Ready"
		m.err = nil
		return m, nil
	case messages.EcsServiceActionMsg:
		m.status = fmt.Sprintf("Service %s %s. Refreshing...", aws.StringValue(m.ecsServiceActionService.ServiceName), msg)
		m.err = nil
		m.action = ""
		m.ecsServiceActionService = nil
		m.confirming = false
		return m, tea.Batch(m.parent.spinner.Tick, commands.FetchECSServicesCmd(m.ecsSvc, aws.StringValue(m.detailCluster.ClusterArn)))
	case messages.EcsServiceLogsFetchedMsg:
		m.header = append(m.header, aws.StringValue(m.detailCluster.ClusterName), m.serviceList.SelectedItem().FilterValue(), "Logs")
		m.serviceLogs = string(msg)
		m.paginator.SetTotalPages(len(strings.Split(m.serviceLogs, "\n")))
		m.status = "Ready"
		m.err = nil
		return m, nil
	case messages.ErrMsg:
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
			s = "No ECS clusters found in this region.\n"
		} else {
			s = m.clusterList.View()
		}
	case ecsStateServiceList:
		if len(m.serviceList.Items()) == 0 && m.status == "Ready" {
			s = "No ECS services found in this cluster.\n"
		} else {
			s = m.serviceList.View()
		}
	case ecsStateServiceDetails:
		if m.detailService != nil {
			s += "\n" + styles.DetailStyle.Render(
				fmt.Sprintf("Service Name:  %s\n", aws.StringValue(m.detailService.ServiceName))+
					fmt.Sprintf("Service ARN:   %s\n", aws.StringValue(m.detailService.ServiceArn))+
					fmt.Sprintf("Status:        %s\n", aws.StringValue(m.detailService.Status))+
					fmt.Sprintf("Desired Count: %d\n", aws.Int64Value(m.detailService.DesiredCount))+
					fmt.Sprintf("Running Count: %d\n", aws.Int64Value(m.detailService.RunningCount))+
					fmt.Sprintf("Pending Count: %d\n", aws.Int64Value(m.detailService.PendingCount))+
					fmt.Sprintf("Launch Type:   %s\n", aws.StringValue(m.detailService.LaunchType))+
					fmt.Sprintf("Task Definition: %s\n", aws.StringValue(m.detailService.TaskDefinition))+
					fmt.Sprintf("Created At:    %s\n", aws.TimeValue(m.detailService.CreatedAt).Format(time.RFC822))+
					"\nPress 'esc' or 'backspace' to go back."+
					"\n"+styles.StatusStyle.Render(fmt.Sprintf("Status: %s", m.status)),
			)
		} else {
			s = styles.StatusStyle.Render("No service details available.\n")
		}
	case ecsStateServiceConfirmAction:
		// s = styles.ConfirmStyle.Render(fmt.Sprintf("\n%s", m.status))
	case ecsStateServiceLogs:
		if m.serviceLogs == "" && m.status == "Ready" {
			s += styles.StatusStyle.Render("No logs found for this service.\n")
		} else {
			lines := strings.Split(m.serviceLogs, "\n")
			start, end := m.paginator.GetSliceBounds(len(lines))
			for _, item := range lines[start:end] {
				s += item + "\n"
			}
		}
		s += m.paginator.View()
		s += "\n" + styles.HelpStyle.Render("Press 'esc' or 'backspace' to go back.")
	}

	return s
}
