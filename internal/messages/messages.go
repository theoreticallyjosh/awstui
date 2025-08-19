package messages

import (
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sfn"
)

// messages are used to pass data between commands and the Update function.
type (
	InstancesFetchedMsg []*ec2.Instance
	InstanceActionMsg   string
	InstanceDetailsMsg  *ec2.Instance

	EcsClustersFetchedMsg    []*ecs.Cluster
	EcsServicesFetchedMsg    []*ecs.Service
	EcsServiceDetailsMsg     *ecs.Service
	EcsServiceActionMsg      string
	EcsServiceLogsFetchedMsg string

	EcrRepositoriesFetchedMsg []*ecr.Repository
	EcrImagesFetchedMsg       []*ecr.ImageDetail
	EcrImageActionMsg         string

	SfnStateMachinesFetchedMsg    []*sfn.StateMachineListItem
	SfnExecutionsFetchedMsg       []*sfn.ExecutionListItem
	SfnExecutionHistoryFetchedMsg []*sfn.HistoryEvent

	BatchJobQueuesFetchedMsg []*batch.JobQueueDetail
	BatchJobsFetchedMsg      []*batch.JobSummary
	BatchJobDetailsMsg       *batch.JobDetail
	BatchJobActionMsg        string
	BatchJobLogsFetchedMsg   string

	SshExitMsg struct{ Err error }
	ErrMsg     error
)
