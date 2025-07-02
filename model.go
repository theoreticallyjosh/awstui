package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// appState defines the current state of the application.
type appState int

const (
	stateMenu appState = iota
	stateEC2Instances
	stateECSClusters
	stateECSServices
	stateECSServiceDetails
	stateECSServiceConfirmAction
	stateECSServiceLogs
)

// model represents the state of our TUI application.
type model struct {
	ec2Svc                  *ec2.EC2
	ecsSvc                  *ecs.ECS
	cloudwatchlogsSvc       *cloudwatchlogs.CloudWatchLogs
	instanceList            list.Model
	clusterList             list.Model
	serviceList             list.Model
	status                  string
	err                     error
	spinner                 spinner.Model
	confirming              bool
	action                  string
	actionID                *string
	showDetails             bool
	detailInstance          *ec2.Instance
	detailCluster           *ecs.Cluster
	detailService           *ecs.Service
	ecsServiceActionService *ecs.Service
	serviceLogs             string
	keys                    *listKeyMap
	state                   appState
	menuCursor              int
	menuChoices             []string
}

// messages are used to pass data between commands and the Update function.
type (
	instancesFetchedMsg      []*ec2.Instance
	ecsClustersFetchedMsg    []*ecs.Cluster
	ecsServicesFetchedMsg    []*ecs.Service
	ecsServiceDetailsMsg     *ecs.Service
	ecsServiceActionMsg      string
	ecsServiceLogsFetchedMsg string
	instanceActionMsg        string
	instanceDetailsMsg       *ec2.Instance
	sshFinishedMsg           error
	errMsg                   error
)

// Init initializes the model and starts fetching data based on the initial state.
func (m model) Init() tea.Cmd {
	switch m.state {
	case stateEC2Instances:
		return tea.Batch(m.spinner.Tick, fetchInstancesCmd(m.ec2Svc))
	case stateECSClusters:
		return tea.Batch(m.spinner.Tick, fetchECSClustersCmd(m.ecsSvc))
	case stateECSServices:
		if m.detailCluster != nil {
			return tea.Batch(m.spinner.Tick, fetchECSServicesCmd(m.ecsSvc, aws.StringValue(m.detailCluster.ClusterArn)))
		}
	}
	return nil
}
