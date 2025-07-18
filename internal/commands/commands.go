package commands

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/theoreticallyjosh/awstui/internal/messages"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	tea "github.com/charmbracelet/bubbletea"
)

// FetchECRRepositoriesCmd fetches ECR repositories from AWS.
func FetchECRRepositoriesCmd(svc *ecr.ECR) tea.Cmd {
	return func() tea.Msg {
		result, err := svc.DescribeRepositories(&ecr.DescribeRepositoriesInput{})
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to describe ECR repositories: %w", err))
		}
		return messages.EcrRepositoriesFetchedMsg(result.Repositories)
	}
}

// FetchECRImagesCmd fetches ECR images from a specific repository.
func FetchECRImagesCmd(svc *ecr.ECR, repositoryName *string) tea.Cmd {
	return func() tea.Msg {
		result, err := svc.DescribeImages(&ecr.DescribeImagesInput{
			RepositoryName: repositoryName,
		})
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to describe ECR images: %w", err))
		}
		return messages.EcrImagesFetchedMsg(result.ImageDetails)
	}
}

// PullEcrImageCmd pulls a docker image from ECR.
func PullEcrImageCmd(svc *ecr.ECR, repositoryUri string, imageTag string) tea.Cmd {
	return func() tea.Msg {
		// Get login token
		result, err := svc.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to get ECR authorization token: %w", err))
		}
		// Decode token
		token, err := base64.StdEncoding.DecodeString(aws.StringValue(result.AuthorizationData[0].AuthorizationToken))
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to decode ECR authorization token: %w", err))
		}
		// Login to ECR
		loginCmd := exec.Command("docker", "login", "-u", "AWS", "-p", string(token)[4:], repositoryUri)
		if err := loginCmd.Run(); err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to login to ECR: %w", err))
		}
		// Pull image
		imageName := fmt.Sprintf("%s:%s", repositoryUri, imageTag)
		pullCmd := exec.Command("docker", "pull", imageName)
		if err := pullCmd.Run(); err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to pull docker image: %w", err))
		}
		return messages.EcrImageActionMsg("pulled")
	}
}

// PushEcrImageCmd pushes a docker image to ECR.
func PushEcrImageCmd(svc *ecr.ECR, repositoryUri string, imageTag string) tea.Cmd {
	return func() tea.Msg {
		// Get login token
		result, err := svc.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to get ECR authorization token: %w", err))
		}
		// Decode token
		token, err := base64.StdEncoding.DecodeString(aws.StringValue(result.AuthorizationData[0].AuthorizationToken))
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to decode ECR authorization token: %w", err))
		}
		// Login to ECR
		loginCmd := exec.Command("docker", "login", "-u", "AWS", "-p", string(token)[4:], repositoryUri)
		if err := loginCmd.Run(); err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to login to ECR: %w", err))
		}
		// Push image
		imageName := fmt.Sprintf("%s:%s", repositoryUri, imageTag)
		pushCmd := exec.Command("docker", "push", imageName)
		if err := pushCmd.Run(); err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to push docker image: %w", err))
		}
		return messages.EcrImageActionMsg("pushed")
	}
}

// FetchInstancesCmd fetches EC2 instances from AWS.
func FetchInstancesCmd(svc *ec2.EC2) tea.Cmd {
	return func() tea.Msg {
		result, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{})
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to describe instances: %w", err))
		}
		var instances []*ec2.Instance
		for _, reservation := range result.Reservations {
			for _, instance := range reservation.Instances {
				if *instance.State.Name != ec2.InstanceStateNameTerminated {
					instances = append(instances, instance)
				}
			}
		}
		return messages.InstancesFetchedMsg(instances)
	}
}

// FetchInstanceDetailsCmd fetches details for a specific EC2 instance.
func FetchInstanceDetailsCmd(svc *ec2.EC2, instanceID *string) tea.Cmd {
	return func() tea.Msg {
		result, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
			InstanceIds: []*string{instanceID},
		})
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to describe instance %s details: %w", *instanceID, err))
		}

		if len(result.Reservations) > 0 && len(result.Reservations[0].Instances) > 0 {
			return messages.InstanceDetailsMsg(result.Reservations[0].Instances[0])
		}
		return messages.ErrMsg(fmt.Errorf("instance %s not found", *instanceID))
	}
}

// StopInstanceCmd stops a specific EC2 instance.
func StopInstanceCmd(svc *ec2.EC2, instanceID *string) tea.Cmd {
	return func() tea.Msg {
		_, err := svc.StopInstances(&ec2.StopInstancesInput{
			InstanceIds: []*string{instanceID},
		})
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to stop instance %s: %w", *instanceID, err))
		}
		time.Sleep(2 * time.Second)
		return messages.InstanceActionMsg("stopped")
	}
}

// StartInstanceCmd starts a specific EC2 instance.
func StartInstanceCmd(svc *ec2.EC2, instanceID *string) tea.Cmd {
	return func() tea.Msg {
		_, err := svc.StartInstances(&ec2.StartInstancesInput{
			InstanceIds: []*string{instanceID},
		})
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to start instance %s: %w", *instanceID, err))
		}
		time.Sleep(2 * time.Second)
		return messages.InstanceActionMsg("started")
	}
}

// FetchECSClustersCmd fetches ECS clusters from AWS.
func FetchECSClustersCmd(svc *ecs.ECS) tea.Cmd {
	return func() tea.Msg {
		listResult, err := svc.ListClusters(&ecs.ListClustersInput{})
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to list ECS clusters: %w", err))
		}

		if len(listResult.ClusterArns) == 0 {
			return messages.EcsClustersFetchedMsg([]*ecs.Cluster{})
		}

		describeResult, err := svc.DescribeClusters(&ecs.DescribeClustersInput{
			Clusters: listResult.ClusterArns,
		})
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to describe ECS clusters: %w", err))
		}

		return messages.EcsClustersFetchedMsg(describeResult.Clusters)
	}
}

// FetchECSServicesCmd fetches ECS services for a given cluster ARN.
func FetchECSServicesCmd(svc *ecs.ECS, clusterArn string) tea.Cmd {
	return func() tea.Msg {
		listResult, err := svc.ListServices(&ecs.ListServicesInput{
			Cluster: aws.String(clusterArn),
		})
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to list ECS services for cluster %s: %w", clusterArn, err))
		}

		if len(listResult.ServiceArns) == 0 {
			return messages.EcsServicesFetchedMsg([]*ecs.Service{})
		}

		describeResult, err := svc.DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  aws.String(clusterArn),
			Services: listResult.ServiceArns,
		})
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to describe ECS services for cluster %s: %w", clusterArn, err))
		}

		return messages.EcsServicesFetchedMsg(describeResult.Services)
	}
}

// FetchECSServiceDetailsCmd fetches details for a specific ECS service.
func FetchECSServiceDetailsCmd(svc *ecs.ECS, clusterArn, serviceArn string) tea.Cmd {
	return func() tea.Msg {
		describeResult, err := svc.DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  aws.String(clusterArn),
			Services: []*string{aws.String(serviceArn)},
		})
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to describe ECS service %s details: %w", serviceArn, err))
		}

		if len(describeResult.Services) > 0 {
			return messages.EcsServiceDetailsMsg(describeResult.Services[0])
		}
		return messages.ErrMsg(fmt.Errorf("ECS service %s not found", serviceArn))
	}
}

// StopECSServiceCmd updates the desired count of an ECS service to 0 to stop it.
func StopECSServiceCmd(svc *ecs.ECS, clusterArn, serviceArn string) tea.Cmd {
	return func() tea.Msg {
		_, err := svc.UpdateService(&ecs.UpdateServiceInput{
			Cluster:      aws.String(clusterArn),
			Service:      aws.String(serviceArn),
			DesiredCount: aws.Int64(0),
		})
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to stop ECS service %s: %w", serviceArn, err))
		}
		time.Sleep(2 * time.Second)
		return messages.EcsServiceActionMsg("stopped")
	}
}

// ForceDeployECSServiceCmd forces a new deployment of an ECS service.
func ForceDeployECSServiceCmd(svc *ecs.ECS, clusterArn, serviceArn string) tea.Cmd {
	return func() tea.Msg {
		_, err := svc.UpdateService(&ecs.UpdateServiceInput{
			Cluster:            aws.String(clusterArn),
			Service:            aws.String(serviceArn),
			ForceNewDeployment: aws.Bool(true),
		})
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to force deploy ECS service %s: %w", serviceArn, err))
		}
		time.Sleep(2 * time.Second)
		return messages.EcsServiceActionMsg("force-deployed")
	}
}

// FetchECSServiceLogsCmd fetches logs for a specific ECS service from CloudWatch Logs.
func FetchECSServiceLogsCmd(ecsSvc *ecs.ECS, cloudwatchlogsSvc *cloudwatchlogs.CloudWatchLogs, service *ecs.Service) tea.Cmd {
	return func() tea.Msg {
		var allLogs strings.Builder

		taskDefinitionArn := aws.StringValue(service.TaskDefinition)
		if taskDefinitionArn == "" {
			return messages.ErrMsg(fmt.Errorf("service %s has no associated task definition", aws.StringValue(service.ServiceName)))
		}

		describeTaskDefResult, err := ecsSvc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: aws.String(taskDefinitionArn),
		})
		serviceName := strings.ReplaceAll(aws.StringValue(describeTaskDefResult.TaskDefinition.Family), "application-", "")
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to describe task definition %s for service %s: %w", taskDefinitionArn, aws.StringValue(service.ServiceName), err))
		}

		var logGroupName string
		var logStreamPrefix string

		for _, containerDef := range describeTaskDefResult.TaskDefinition.ContainerDefinitions {
			if containerDef.LogConfiguration != nil && aws.StringValue(containerDef.LogConfiguration.LogDriver) == "awslogs" {
				if group, ok := containerDef.LogConfiguration.Options["awslogs-group"]; ok {
					logGroupName = aws.StringValue(group)
				}
				if prefix, ok := containerDef.LogConfiguration.Options["awslogs-stream-prefix"]; ok {
					logStreamPrefix = aws.StringValue(prefix)
				}
				break
			}
		}

		if logGroupName == "" {
			return messages.ErrMsg(fmt.Errorf("awslogs log group not found in task definition for service %s", aws.StringValue(service.ServiceName)))
		}

		twentyFourHoursAgo := time.Now().Add(-24 * time.Hour).UnixMilli()

		streamNamePrefix := fmt.Sprintf("%s/%s", logStreamPrefix, serviceName)
		logStreamsResult, err := cloudwatchlogsSvc.DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
			LogGroupName: aws.String(logGroupName),
			// LogStreamNamePrefix: aws.String(streamNamePrefix),
			Descending: aws.Bool(true),
			Limit:      aws.Int64(20),
			OrderBy:    aws.String("LastEventTime"),
		})
		if err != nil {
			return messages.ErrMsg(fmt.Errorf("failed to describe log streams for service %s (group: %s, prefix: %s): %w", aws.StringValue(service.ServiceName), logGroupName, streamNamePrefix, err))
		}

		if len(logStreamsResult.LogStreams) == 0 {
			return messages.EcsServiceLogsFetchedMsg(fmt.Sprintf("No log streams found for this service in the last 24 hours. (%s %s)", logGroupName, streamNamePrefix))
		}

		for _, stream := range logStreamsResult.LogStreams {
			strings.Contains(*stream.LogStreamName, serviceName)
			getEventsInput := &cloudwatchlogs.GetLogEventsInput{
				LogGroupName:  aws.String(logGroupName),
				LogStreamName: stream.LogStreamName,
				StartTime:     aws.Int64(twentyFourHoursAgo),
				Limit:         aws.Int64(50),
			}

			eventsResult, err := cloudwatchlogsSvc.GetLogEvents(getEventsInput)
			if len(eventsResult.Events) > 0 {
				allLogs.WriteString(fmt.Sprintf("--- Log Stream: %s ---\n", aws.StringValue(stream.LogStreamName)))
				if err != nil {
					allLogs.WriteString(fmt.Sprintf("Error fetching events from %s: %v\n", aws.StringValue(stream.LogStreamName), err))
					continue
				}

				for _, event := range eventsResult.Events {
					allLogs.WriteString(fmt.Sprintf("[%s] %s\n", time.UnixMilli(aws.Int64Value(event.Timestamp)).Format("15:04:05"), aws.StringValue(event.Message)))
				}
				allLogs.WriteString("\n")
			}
		}

		if allLogs.Len() == 0 {
			return messages.EcsServiceLogsFetchedMsg("No logs found for this service in the last 24 hours.")
		}

		return messages.EcsServiceLogsFetchedMsg(allLogs.String())
	}
}

// SshIntoInstanceCmd executes an SSH command to connect to the given IP.
func SshIntoInstanceCmd(publicIP string, keyName string) tea.Cmd {
	return tea.ExecProcess(exec.Command("ssh", "-i", "~/.ssh/"+keyName+".pem", "ec2-user@"+publicIP), func(err error) tea.Msg {
		return messages.SshExitMsg{Err: err}
	})
}
