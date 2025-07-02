package main

import "github.com/charmbracelet/bubbles/key"

type listKeyMap struct {
	details key.Binding
	start   key.Binding
	stop    key.Binding
	ssh     key.Binding
	refresh key.Binding
	logs    key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		details: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "details"),
		),
		stop: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "stop"),
		),
		start: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "start"),
		),
		ssh: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "ssh"),
		),
		refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		logs: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "logs"),
		),
	}
}
