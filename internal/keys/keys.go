package keys

import "github.com/charmbracelet/bubbles/key"

type ListKeyMap struct {
	Details        key.Binding
	Start          key.Binding
	Stop           key.Binding
	Ssh            key.Binding
	Refresh        key.Binding
	Logs           key.Binding
	ForceDeploy    key.Binding
	Scale          key.Binding
	Pull           key.Binding
	Push           key.Binding
	Choose         key.Binding
	StartExecution key.Binding
}

func NewListKeyMap() *ListKeyMap {
	return &ListKeyMap{
		Details: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "details"),
		),
		Stop: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "stop"),
		),
		Start: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "start"),
		),
		Ssh: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "ssh"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Logs: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "logs"),
		),
		ForceDeploy: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "force deploy"),
		),
		Scale: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "scale"),
		),
		Pull: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "pull"),
		),
		Push: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "push"),
		),
		Choose: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "choose"),
		),
		StartExecution: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "execute"),
		),
	}
}
