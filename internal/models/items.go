package models

import (
	"awstui/internal/styles"
	"awstui/internal/utils"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ec2InstanceItem represents a single EC2 instance in the list.
type ec2InstanceItem struct {
	instance *ec2.Instance
}

// FilterValue implements list.Item.
func (i ec2InstanceItem) FilterValue() string {
	return utils.GetInstanceName(i.instance) + " " + aws.StringValue(i.instance.InstanceId) + " " + aws.StringValue(i.instance.State.Name)
}

// Title implements list.DefaultItem.
func (i ec2InstanceItem) Title() string {
	return styles.TitleStyle.Render(fmt.Sprintf("%s (%s)", utils.GetInstanceName(i.instance), aws.StringValue(i.instance.InstanceId)))
}

// Description implements list.DefaultItem.
func (i ec2InstanceItem) Description() string {
	return styles.DescriptionStyle.Render(fmt.Sprintf("Type: %s | State: %s | Public IP: %s",
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
	return styles.TitleStyle.Render(fmt.Sprintf("%s", aws.StringValue(i.cluster.ClusterName)))
}

// Description implements list.DefaultItem for ECS clusters.
func (i ecsClusterItem) Description() string {
	return styles.DescriptionStyle.Render(fmt.Sprintf("Status: %s | Services: %d | Tasks: %d | Container Instances: %d",
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
	return styles.TitleStyle.Render(fmt.Sprintf("%s", aws.StringValue(i.service.ServiceName)))
}

// Description implements list.DefaultItem for ECS services.
func (i ecsServiceItem) Description() string {
	return styles.DescriptionStyle.Render(fmt.Sprintf("Status: %s | Desired: %d | Running: %d | Pending: %d",
		aws.StringValue(i.service.Status),
		aws.Int64Value(i.service.DesiredCount),
		aws.Int64Value(i.service.RunningCount),
		aws.Int64Value(i.service.PendingCount),
	))
}

// ItemDelegate customizes how each item in the list is rendered.
type ItemDelegate struct {
}

func (d ItemDelegate) Height() int                               { return 2 }
func (d ItemDelegate) Spacing() int                              { return 1 }
func (d ItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d ItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title, description string
	var style lipgloss.Style

	if index == m.Index() {
		style = styles.SelectedItemStyle
	} else {
		style = styles.UnselectedItemStyle
	}

	switch i := item.(type) {
	case ec2InstanceItem:
		title = i.Title()
		description = i.Description()
	case ecsClusterItem:
		title = i.Title()
		description = i.Description()
	case ecsServiceItem:
		title = i.Title()
		description = i.Description()
	default:
		return
	}

	str := fmt.Sprintf("%s\n%s", title, description)
	fmt.Fprint(w, style.Render(str))
}
