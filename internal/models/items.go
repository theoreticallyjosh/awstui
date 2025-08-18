package models

import (
	"fmt"
	"time"

	"github.com/theoreticallyjosh/awstui/internal/utils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sfn"
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

func (i resourceItem) Title() string       { return i.title }
func (i resourceItem) Description() string { return i.desc }
func (i resourceItem) FilterValue() string { return i.title }

// EC2 Instance Item
type ec2InstanceItem struct {
	instance *ec2.Instance
}

func (i ec2InstanceItem) Title() string {
	return getInstanceName(i.instance)
}
func (i ec2InstanceItem) Description() string {
	return fmt.Sprintf("ID: %s | State: %s | Type: %s",
		aws.StringValue(i.instance.InstanceId),
		aws.StringValue(i.instance.State.Name),
		aws.StringValue(i.instance.InstanceType),
	)
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
	return aws.StringValue(i.cluster.ClusterName)
}
func (i ecsClusterItem) Description() string {
	return fmt.Sprintf("ARN: %s", aws.StringValue(i.cluster.ClusterArn))
}
func (i ecsClusterItem) FilterValue() string {
	return aws.StringValue(i.cluster.ClusterName)
}

// ECS Service Item
type ecsServiceItem struct {
	service *ecs.Service
}

func (i ecsServiceItem) Title() string {
	return aws.StringValue(i.service.ServiceName)
}
func (i ecsServiceItem) Description() string {
	return fmt.Sprintf("Status: %s | Desired: %d | Running: %d",
		aws.StringValue(i.service.Status),
		aws.Int64Value(i.service.DesiredCount),
		aws.Int64Value(i.service.RunningCount),
	)
}
func (i ecsServiceItem) FilterValue() string {
	return aws.StringValue(i.service.ServiceName)
}

// ECR Repository Item
type ecrRepositoryItem struct {
	repository *ecr.Repository
}

func (i ecrRepositoryItem) Title() string {
	return aws.StringValue(i.repository.RepositoryName)
}

func (i ecrRepositoryItem) Description() string {
	return fmt.Sprintf("URI: %s", aws.StringValue(i.repository.RepositoryUri))
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
		return utils.ArrayToCSV(i.image.ImageTags)
	}
	return aws.StringValue(i.image.ImageDigest)
}

func (i ecrImageItem) Description() string {
	return fmt.Sprintf("Digest: %s | Pushed: %s",
		aws.StringValue(i.image.ImageDigest),
		aws.TimeValue(i.image.ImagePushedAt).Format(time.RFC822),
	)
}

func (i ecrImageItem) FilterValue() string {
	if len(i.image.ImageTags) > 0 {
		return utils.ArrayToCSV(i.image.ImageTags)
	}
	return aws.StringValue(i.image.ImageDigest)
}

// SFN State Machine Item
type sfnStateMachineItem struct {
	stateMachine *sfn.StateMachineListItem
}

func (i sfnStateMachineItem) Title() string {
	return aws.StringValue(i.stateMachine.Name)
}

func (i sfnStateMachineItem) Description() string {
	return fmt.Sprintf("ARN: %s", aws.StringValue(i.stateMachine.StateMachineArn))
}

func (i sfnStateMachineItem) FilterValue() string {
	return aws.StringValue(i.stateMachine.Name)
}

// SFN Execution Item
type sfnExecutionItem struct {
	execution *sfn.ExecutionListItem
}

func (i sfnExecutionItem) Title() string {
	return aws.StringValue(i.execution.Name)
}

func (i sfnExecutionItem) Description() string {
	return fmt.Sprintf("Status: %s | Started: %s",
		aws.StringValue(i.execution.Status),
		aws.TimeValue(i.execution.StartDate).Format(time.RFC822),
	)
}

func (i sfnExecutionItem) FilterValue() string {
	return aws.StringValue(i.execution.Name)
}
