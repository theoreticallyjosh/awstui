package models

import (
	"awstui/internal/commands"
	"awstui/internal/keys"
	"awstui/internal/messages"
	"awstui/internal/styles"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type ecrState int

const (
	ecrStateRepositoryList ecrState = iota
	ecrStateImageList
)

type ecrModel struct {
	parent             *Model
	ecrSvc             *ecr.ECR
	repositoryList     list.Model
	imageList          list.Model
	status             string
	err                error
	keys               *keys.ListKeyMap
	state              ecrState
	confirming         bool
	action             string
	actionID           *string
	selectedRepository *ecr.Repository
}

func (m ecrModel) Init() tea.Cmd {
	return tea.Batch(m.parent.spinner.Tick, commands.FetchECRRepositoriesCmd(m.ecrSvc))
}

func (m ecrModel) Update(msg tea.Msg) (ecrModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := styles.AppStyle.GetFrameSize()
		m.repositoryList.SetSize(msg.Width-3*h, msg.Height-4*v)
		m.imageList.SetSize(msg.Width-3*h, msg.Height-4*v)
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("backspace", "esc"), key.WithHelp("backspace/esc", "back"))):
			if m.state == ecrStateImageList {
				m.state = ecrStateRepositoryList
				m.status = "Ready"
				m.err = nil
				m.imageList.SetItems([]list.Item{})
				return m, nil
			}
		}
		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.status = fmt.Sprintf("%sing image %s...", m.action, *m.actionID)
				m.err = nil
				if m.action == "pull" {
					return m, tea.Batch(m.parent.spinner.Tick, commands.PullEcrImageCmd(m.ecrSvc, aws.StringValue(m.selectedRepository.RepositoryUri), *m.actionID))
				} else if m.action == "push" {
					return m, tea.Batch(m.parent.spinner.Tick, commands.PushEcrImageCmd(m.ecrSvc, aws.StringValue(m.selectedRepository.RepositoryUri), *m.actionID))
				}
			case "n", "N":
				m.confirming = false
				m.status = "Action cancelled."
				m.action = ""
				m.actionID = nil
			}
			return m, nil
		}
		switch m.state {
		case ecrStateRepositoryList:
			if m.repositoryList.FilterState() == list.Filtering {
				break
			}
			switch {
			case key.Matches(msg, m.keys.Refresh):
				m.status = "Refreshing ECR repositories..."
				m.err = nil
				return m, tea.Batch(m.parent.spinner.Tick, commands.FetchECRRepositoriesCmd(m.ecrSvc))
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select repository"))):
				if m.repositoryList.SelectedItem() != nil {
					selectedItem := m.repositoryList.SelectedItem().(ecrRepositoryItem)
					m.selectedRepository = selectedItem.repository
					m.state = ecrStateImageList
					m.status = fmt.Sprintf("Loading images for repository %s...", aws.StringValue(selectedItem.repository.RepositoryName))
					return m, tea.Batch(m.parent.spinner.Tick, commands.FetchECRImagesCmd(m.ecrSvc, selectedItem.repository.RepositoryName))
				}
			}
		case ecrStateImageList:
			if m.imageList.FilterState() == list.Filtering {
				break
			}
			switch {
			case key.Matches(msg, m.keys.Pull):
				if m.imageList.SelectedItem() != nil {
					selectedItem := m.imageList.SelectedItem().(ecrImageItem)
					m.confirming = true
					m.action = "pull"
					m.actionID = selectedItem.image.ImageTags[0]
					m.status = fmt.Sprintf("Confirm pulling image %s? (y/N)", aws.StringValue(selectedItem.image.ImageTags[0]))
				}
			case key.Matches(msg, m.keys.Push):
				if m.imageList.SelectedItem() != nil {
					selectedItem := m.imageList.SelectedItem().(ecrImageItem)
					m.confirming = true
					m.action = "push"
					m.actionID = selectedItem.image.ImageTags[0]
					m.status = fmt.Sprintf("Confirm pushing image %s? (y/N)", aws.StringValue(selectedItem.image.ImageTags[0]))
				}
			}
		}

	case messages.EcrRepositoriesFetchedMsg:
		listItems := make([]list.Item, len(msg))
		for i, repository := range msg {
			listItems[i] = ecrRepositoryItem{repository: repository}
		}
		m.repositoryList.SetItems(listItems)
		m.status = "Ready"
		m.err = nil
		return m, nil

	case messages.EcrImagesFetchedMsg:
		listItems := make([]list.Item, len(msg))
		for i, image := range msg {
			listItems[i] = ecrImageItem{image: image}
		}
		m.imageList.SetItems(listItems)
		m.status = "Ready"
		m.err = nil
		return m, nil
	case messages.EcrImageActionMsg:
		m.status = fmt.Sprintf("Image %s. Refreshing...", msg)
		m.err = nil
		m.action = ""
		m.actionID = nil
		return m, tea.Batch(m.parent.spinner.Tick, commands.FetchECRImagesCmd(m.ecrSvc, m.selectedRepository.RepositoryName))

	case messages.ErrMsg:
		m.err = msg
		m.status = "Error"
		return m, nil
	}
	if m.state == ecrStateRepositoryList {
		m.repositoryList, cmd = m.repositoryList.Update(msg)
	} else if m.state == ecrStateImageList {
		m.imageList, cmd = m.imageList.Update(msg)
	}
	return m, cmd
}

func (m ecrModel) View() string {
	var s string
	switch m.state {
	case ecrStateRepositoryList:
		if len(m.repositoryList.Items()) == 0 && m.status == "Ready" {
			s = styles.StatusStyle.Render("No ECR repositories found in this region.\n")
		} else {
			s = m.repositoryList.View()
		}
	case ecrStateImageList:
		m.imageList.Title = fmt.Sprintf("Images in Registry: %s", aws.StringValue(m.selectedRepository.RepositoryName))
		if len(m.imageList.Items()) == 0 && m.status == "Ready" {
			s = styles.StatusStyle.Render("No ECR images found in this repository.\n")
		} else {
			s = m.imageList.View()
		}
	}
	return s
}
