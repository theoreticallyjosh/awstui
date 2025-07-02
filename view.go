package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
)

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
	case stateECSServices:
		s.WriteString(headerStyle.Render(fmt.Sprintf("ECS Services in Cluster: %s\n", aws.StringValue(m.detailCluster.ClusterName))))
		if len(m.serviceList.Items()) == 0 && m.status == "Ready" {
			s.WriteString(statusStyle.Render("No ECS services found in this cluster.\n"))
		} else {
			s.WriteString(m.serviceList.View())
		}

		if m.status != "Ready" && m.status != "Error" {
			s.WriteString(statusStyle.Render(fmt.Sprintf("\n%s %s", m.spinner.View(), m.status)))
		} else if m.confirming {
			s.WriteString(confirmStyle.Render(fmt.Sprintf("\n%s", m.status)))
		} else {
			s.WriteString(statusStyle.Render(fmt.Sprintf("\nStatus: %s", m.status)))
		}
	case stateECSServiceDetails:
		if m.detailService != nil {
			s.WriteString(headerStyle.Render(fmt.Sprintf("ECS Service Details: %s\n", aws.StringValue(m.detailService.ServiceName))))
			s.WriteString("\n" + detailStyle.Render(
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
					"\n"+statusStyle.Render(fmt.Sprintf("Status: %s", m.status)),
			))
		} else {
			s.WriteString(statusStyle.Render("No service details available.\n"))
		}
	case stateECSServiceConfirmAction:
		s.WriteString(confirmStyle.Render(fmt.Sprintf("\n%s", m.status)))
	case stateECSServiceLogs:
		s.WriteString(headerStyle.Render(fmt.Sprintf("Logs for Service: %s\n", aws.StringValue(m.detailService.ServiceName))))
		if m.serviceLogs == "" && m.status == "Ready" {
			s.WriteString(statusStyle.Render("No logs found for this service.\n"))
		} else {
			s.WriteString("\n" + detailStyle.Render(m.serviceLogs))
		}
		s.WriteString(statusStyle.Render("\nPress 'esc' or 'backspace' to go back."))
	}

	return appStyle.Render(s.String())
}
