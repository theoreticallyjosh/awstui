package models

import (
	"fmt"

	"github.com/theoreticallyjosh/awstui/internal/commands"
	"github.com/theoreticallyjosh/awstui/internal/keys"
	"github.com/theoreticallyjosh/awstui/internal/messages"
	"github.com/theoreticallyjosh/awstui/internal/styles"

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
	header             []string
}

func (m ecrModel) Init() tea.Cmd {
	return tea.Batch(m.parent.spinner.Tick, commands.FetchECRRepositoriesCmd(m.ecrSvc))
}

func (m ecrModel) Update(msg tea.Msg) (ecrModel, tea.Cmd) {
	m.header = []string{"ECR Repositories"}
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.repositoryList.SetSize(msg.Width, msg.Height)
		m.imageList.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case messages.EcrRepositoriesFetchedMsg:
		return m.handleRepositoriesFetched(msg)

	case messages.EcrImagesFetchedMsg:
		return m.handleImagesFetched(msg)

	case messages.EcrImageActionMsg:
		return m.handleImageAction(msg)

	case messages.ErrMsg:
		return m.handleErrorMessage(msg)
	}

	// Update the appropriate list based on current state
	if m.state == ecrStateRepositoryList {
		m.repositoryList, cmd = m.repositoryList.Update(msg)
	} else if m.state == ecrStateImageList {
		m.header = append(m.header, aws.StringValue(m.selectedRepository.RepositoryName), "Images")
		m.imageList, cmd = m.imageList.Update(msg)
	}

	return m, cmd
}

func (m ecrModel) handleKeyMsg(msg tea.KeyMsg) (ecrModel, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))):
		return m.handleEscKey()
	}

	if m.confirming {
		return m.handleConfirmation(msg)
	}

	switch m.state {
	case ecrStateRepositoryList:
		return m.handleRepositoryListKeyMsg(msg)
	case ecrStateImageList:
		return m.handleImageListKeyMsg(msg)
	}

	return m, nil
}

func (m ecrModel) handleEscKey() (ecrModel, tea.Cmd) {
	if m.state == ecrStateImageList {
		m.state = ecrStateRepositoryList
		m.status = "Ready"
		m.err = nil
		m.imageList.SetItems([]list.Item{})
		return m, nil
	}
	return m, nil
}

func (m ecrModel) handleConfirmation(msg tea.KeyMsg) (ecrModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.executeAction()
	case "n", "N":
		m.confirming = false
		m.status = "Action cancelled."
		m.action = ""
		m.actionID = nil
	}
	return m, nil
}

func (m ecrModel) executeAction() (ecrModel, tea.Cmd) {
	m.confirming = false
	m.status = fmt.Sprintf("%sing image %s...", m.action, *m.actionID)
	m.err = nil

	var cmd tea.Cmd
	if m.action == "pull" {
		cmd = commands.PullEcrImageCmd(m.ecrSvc, aws.StringValue(m.selectedRepository.RepositoryUri), *m.actionID)
	} else if m.action == "push" {
		cmd = commands.PushEcrImageCmd(m.ecrSvc, aws.StringValue(m.selectedRepository.RepositoryUri), *m.actionID)
	}
	return m, tea.Batch(m.parent.spinner.Tick, cmd)
}

func (m ecrModel) handleRepositoryListKeyMsg(msg tea.KeyMsg) (ecrModel, tea.Cmd) {
	if m.repositoryList.FilterState() == list.Filtering {
		return m, nil
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

	var cmd tea.Cmd
	m.repositoryList, cmd = m.repositoryList.Update(msg)
	return m, cmd
}

func (m ecrModel) handleImageListKeyMsg(msg tea.KeyMsg) (ecrModel, tea.Cmd) {
	if m.imageList.FilterState() == list.Filtering {
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Pull):
		return m.handlePullAction(msg)
	case key.Matches(msg, m.keys.Push):
		return m.handlePushAction(msg)
	}

	var cmd tea.Cmd
	m.imageList, cmd = m.imageList.Update(msg)
	return m, cmd
}

func (m ecrModel) handlePullAction(msg tea.KeyMsg) (ecrModel, tea.Cmd) {
	if m.imageList.SelectedItem() != nil {
		selectedItem := m.imageList.SelectedItem().(ecrImageItem)
		m.confirming = true
		m.action = "pull"
		m.actionID = selectedItem.image.ImageTags[0]
		m.status = fmt.Sprintf("Confirm pulling image %s? (y/N)", aws.StringValue(selectedItem.image.ImageTags[0]))
	}
	return m, nil
}

func (m ecrModel) handlePushAction(msg tea.KeyMsg) (ecrModel, tea.Cmd) {
	if m.imageList.SelectedItem() != nil {
		selectedItem := m.imageList.SelectedItem().(ecrImageItem)
		m.confirming = true
		m.action = "push"
		m.actionID = selectedItem.image.ImageTags[0]
		m.status = fmt.Sprintf("Confirm pushing image %s? (y/N)", aws.StringValue(selectedItem.image.ImageTags[0]))
	}
	return m, nil
}

func (m ecrModel) handleRepositoriesFetched(msg messages.EcrRepositoriesFetchedMsg) (ecrModel, tea.Cmd) {
	listItems := make([]list.Item, len(msg))
	for i, repository := range msg {
		listItems[i] = ecrRepositoryItem{repository: repository}
	}
	m.repositoryList.SetItems(listItems)
	m.status = "Ready"
	m.err = nil
	return m, nil
}

func (m ecrModel) handleImagesFetched(msg messages.EcrImagesFetchedMsg) (ecrModel, tea.Cmd) {
	m.header = append(m.header, aws.StringValue(m.selectedRepository.RepositoryName), "Images")
	listItems := make([]list.Item, len(msg))
	for i, image := range msg {
		listItems[i] = ecrImageItem{image: image}
	}
	m.imageList.SetItems(listItems)
	m.status = "Ready"
	m.err = nil
	return m, nil
}

func (m ecrModel) handleImageAction(msg messages.EcrImageActionMsg) (ecrModel, tea.Cmd) {
	m.status = fmt.Sprintf("Image %s. Refreshing...", msg)
	m.err = nil
	m.action = ""
	m.actionID = nil
	return m, tea.Batch(m.parent.spinner.Tick, commands.FetchECRImagesCmd(m.ecrSvc, m.selectedRepository.RepositoryName))
}

func (m ecrModel) handleErrorMessage(msg messages.ErrMsg) (ecrModel, tea.Cmd) {
	m.err = msg
	m.status = "Error"
	return m, nil
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
		if len(m.imageList.Items()) == 0 && m.status == "Ready" {
			s = styles.StatusStyle.Render("No ECR images found in this repository.\n")
		} else {
			s = m.imageList.View()
		}
	}
	return s
}
