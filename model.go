package main

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// appState defines the current state of the application.
type appState int

const (
	stateMenu appState = iota
	stateEC2
	stateECS
)

// model represents the state of our TUI application.
type model struct {
	ec2Model    ec2Model
	ecsModel    ecsModel
	spinner     spinner.Model
	status      string
	err         error
	state       appState
	menuCursor  int
	menuChoices []string
	keys        *listKeyMap
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
	sshExitMsg               struct{ err error }
	errMsg                   error
)

// Init initializes the model and starts fetching data based on the initial state.
func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}
