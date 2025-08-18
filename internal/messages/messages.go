package messages

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sfn"
)

// messages are used to pass data between commands and the Update function.
type (
	InstancesFetchedMsg      []*ec2.Instance
	EcsClustersFetchedMsg    []*ecs.Cluster
	EcsServicesFetchedMsg    []*ecs.Service
	EcrRepositoriesFetchedMsg []*ecr.Repository
	EcrImagesFetchedMsg      []*ecr.ImageDetail
	EcrImageActionMsg        string
	EcsServiceDetailsMsg     *ecs.Service
	EcsServiceActionMsg      string
	EcsServiceLogsFetchedMsg string
	SfnStateMachinesFetchedMsg []*sfn.StateMachineListItem
	SfnExecutionsFetchedMsg    []*sfn.ExecutionListItem
	SfnExecutionHistoryFetchedMsg []*sfn.HistoryEvent
	InstanceActionMsg          string
	InstanceDetailsMsg       *ec2.Instance
	SshExitMsg               struct{ Err error }
	ErrMsg                   error
)
