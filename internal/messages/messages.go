package messages

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
)

// messages are used to pass data between commands and the Update function.
type (
	InstancesFetchedMsg      []*ec2.Instance
	EcsClustersFetchedMsg    []*ecs.Cluster
	EcsServicesFetchedMsg    []*ecs.Service
	EcsServiceDetailsMsg     *ecs.Service
	EcsServiceActionMsg      string
	EcsServiceLogsFetchedMsg string
	InstanceActionMsg        string
	InstanceDetailsMsg       *ec2.Instance
	SshExitMsg               struct{ Err error }
	ErrMsg                   error
)
