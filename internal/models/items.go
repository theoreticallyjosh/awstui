package models

import (
	"awstui/internal/styles"
	"awstui/internal/utils"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/charmbracelet/bubbles/list"
)

type item interface {
	list.Item
	Title() string
	Description() string
}

type resourceItem struct {
	title, desc string
}

func (i resourceItem) Title() string       { return styles.TitleStyle.Render(i.title) }
func (i resourceItem) Description() string { return styles.DescriptionStyle.Render(i.desc) }
func (i resourceItem) FilterValue() string { return i.title }

// EC2 Instance Item
type ec2InstanceItem struct {
	instance *ec2.Instance
}

func (i ec2InstanceItem) Title() string {
	return styles.TitleStyle.Render(getInstanceName(i.instance))
}
func (i ec2InstanceItem) Description() string {
	return styles.DescriptionStyle.Render(fmt.Sprintf("ID: %s | State: %s | Type: %s",
		aws.StringValue(i.instance.InstanceId),
		aws.StringValue(i.instance.State.Name),
		aws.StringValue(i.instance.InstanceType),
	))
}
func (i ec2InstanceItem) FilterValue() string { return getInstanceName(i.instance) }

func getInstanceName(instance *ec2.Instance) string {
	for _, tag := range instance.Tags {
		if aws.StringValue(tag.Key) == "Name" {
			return aws.StringValue(tag.Value)
		}
	}
	return aws.StringValue(instance.InstanceId)
}

// ECS Cluster Item
type ecsClusterItem struct {
	cluster *ecs.Cluster
}

func (i ecsClusterItem) Title() string {
	return styles.TitleStyle.Render(aws.StringValue(i.cluster.ClusterName))
}
func (i ecsClusterItem) Description() string {
	return styles.DescriptionStyle.Render(fmt.Sprintf("ARN: %s", aws.StringValue(i.cluster.ClusterArn)))
}
func (i ecsClusterItem) FilterValue() string {
	return aws.StringValue(i.cluster.ClusterName)
}

// ECS Service Item
type ecsServiceItem struct {
	service *ecs.Service
}

func (i ecsServiceItem) Title() string {
	return styles.TitleStyle.Render(aws.StringValue(i.service.ServiceName))
}
func (i ecsServiceItem) Description() string {
	return styles.DescriptionStyle.Render(fmt.Sprintf("Status: %s | Desired: %d | Running: %d",
		aws.StringValue(i.service.Status),
		aws.Int64Value(i.service.DesiredCount),
		aws.Int64Value(i.service.RunningCount),
	))
}
func (i ecsServiceItem) FilterValue() string {
	return aws.StringValue(i.service.ServiceName)
}

// ECR Repository Item
type ecrRepositoryItem struct {
	repository *ecr.Repository
}

func (i ecrRepositoryItem) Title() string {
	return styles.TitleStyle.Render(aws.StringValue(i.repository.RepositoryName))
}

func (i ecrRepositoryItem) Description() string {
	return styles.DescriptionStyle.Render(fmt.Sprintf("URI: %s", aws.StringValue(i.repository.RepositoryUri)))
}

func (i ecrRepositoryItem) FilterValue() string {
	return aws.StringValue(i.repository.RepositoryName)
}

// ECR Image Item
type ecrImageItem struct {
	image *ecr.ImageDetail
}

func (i ecrImageItem) Title() string {
	if len(i.image.ImageTags) > 0 {
		return styles.TitleStyle.Render(utils.ArrayToCSV(i.image.ImageTags))
	}
	return aws.StringValue(i.image.ImageDigest)
}

func (i ecrImageItem) Description() string {
	return styles.DescriptionStyle.Render(fmt.Sprintf("Digest: %s | Pushed: %s",
		aws.StringValue(i.image.ImageDigest),
		aws.TimeValue(i.image.ImagePushedAt).Format(time.RFC822),
	))
}

func (i ecrImageItem) FilterValue() string {
	if len(i.image.ImageTags) > 0 {
		return utils.ArrayToCSV(i.image.ImageTags)
	}
	return aws.StringValue(i.image.ImageDigest)
}
