package models

import (
	"fmt"
	"time"

	"github.com/theoreticallyjosh/awstui/internal/commands"
	"github.com/theoreticallyjosh/awstui/internal/keys"
	"github.com/theoreticallyjosh/awstui/internal/messages"
	"github.com/theoreticallyjosh/awstui/internal/styles"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type sfnState int

const (
	sfnStateList sfnState = iota
	sfnStateExecutions
	sfnStateExecutionDetails
)

type sfnModel struct {
	parent               *Model
	sfnSvc               *sfn.SFN
	sfnList              list.Model
	executionList        list.Model
	executionHistoryList list.Model
	status               string
	err                  error
	keys                 *keys.ListKeyMap
	state                sfnState
	header               []string
	selectedStateMachine *sfn.StateMachineListItem
	selectedExecution    *sfn.ExecutionListItem
}

func (m sfnModel) Init() tea.Cmd {
	return tea.Batch(m.parent.spinner.Tick, commands.FetchSFNStateMachinesCmd(m.sfnSvc))
}

func (m sfnModel) Update(msg tea.Msg) (sfnModel, tea.Cmd) {
	m.header = []string{"Step Functions"}
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.sfnList.SetSize(msg.Width, msg.Height)
		m.executionList.SetSize(msg.Width, msg.Height)
		m.executionHistoryList.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))):
			if m.state == sfnStateExecutions {
				m.state = sfnStateList
				m.status = "Ready"
				m.err = nil
				m.executionList.SetItems([]list.Item{})
				return m, nil
			} else {
				m.state = sfnStateExecutions
				m.status = "Ready"
				m.err = nil
				return m, nil
			}
		}
		switch m.state {
		case sfnStateList:
			if m.sfnList.FilterState() == list.Filtering {
				break
			}
			switch {
			case key.Matches(msg, m.keys.Refresh):
				m.status = styles.StatusStyle.Render("Refreshing state machines...")
				m.err = nil
				return m, tea.Batch(m.parent.spinner.Tick, commands.FetchSFNStateMachinesCmd(m.sfnSvc))
			case key.Matches(msg, m.keys.Choose):
				if m.sfnList.SelectedItem() != nil {
					selectedItem := m.sfnList.SelectedItem().(sfnStateMachineItem)
					m.selectedStateMachine = selectedItem.stateMachine
					m.state = sfnStateExecutions
					m.status = fmt.Sprintf("Loading executions for %s...", aws.StringValue(selectedItem.stateMachine.Name))
					return m, tea.Batch(m.parent.spinner.Tick, commands.FetchSFNExecutionsCmd(m.sfnSvc, selectedItem.stateMachine.StateMachineArn))
				}
			}
		case sfnStateExecutions:
			if m.executionList.FilterState() == list.Filtering {
				break
			}
			switch {
			case key.Matches(msg, m.keys.Refresh):
				m.status = styles.StatusStyle.Render("Refreshing executions...")
				m.err = nil
				return m, tea.Batch(m.parent.spinner.Tick, commands.FetchSFNExecutionsCmd(m.sfnSvc, m.selectedStateMachine.StateMachineArn))
			case key.Matches(msg, m.keys.Choose):
				if m.executionList.SelectedItem() != nil {
					selectedItem := m.executionList.SelectedItem().(sfnExecutionItem)
					m.selectedExecution = selectedItem.execution
					m.state = sfnStateExecutionDetails
					m.status = fmt.Sprintf("Loading execution history for %s...", aws.StringValue(selectedItem.execution.Name))
					return m, tea.Batch(m.parent.spinner.Tick, commands.FetchSFNExecutionHistoryCmd(m.sfnSvc, selectedItem.execution.ExecutionArn))
				}
			}
		case sfnStateExecutionDetails:
			if m.executionHistoryList.FilterState() == list.Filtering {
				break
			}
			switch {
			case key.Matches(msg, m.keys.Refresh):
				m.status = styles.StatusStyle.Render("Refreshing execution history...")
				m.err = nil
				return m, tea.Batch(m.parent.spinner.Tick, commands.FetchSFNExecutionHistoryCmd(m.sfnSvc, m.selectedExecution.ExecutionArn))
			}
		}
	case messages.SfnStateMachinesFetchedMsg:
		listItems := make([]list.Item, len(msg))
		for i, sm := range msg {
			listItems[i] = sfnStateMachineItem{stateMachine: sm}
		}
		m.sfnList.SetItems(listItems)
		m.status = "Ready"
		m.err = nil
		return m, nil
	case messages.SfnExecutionsFetchedMsg:
		m.header = append(m.header, aws.StringValue(m.selectedStateMachine.Name), "Executions")
		listItems := make([]list.Item, len(msg))
		for i, ex := range msg {
			listItems[i] = sfnExecutionItem{execution: ex}
		}
		m.executionList.SetItems(listItems)
		m.status = "Ready"
		m.err = nil
		return m, nil
	case messages.SfnExecutionHistoryFetchedMsg:
		m.header = append(m.header, aws.StringValue(m.selectedStateMachine.Name), "Executions", aws.StringValue(m.selectedExecution.Name), "History")
		eventsMap := make(map[int64]*sfn.HistoryEvent)
		for _, e := range msg {
			if e.Id != nil {
				eventsMap[*e.Id] = e
			}
		}
		listItems := []list.Item{}
		for i := 1; i < len(eventsMap); i++ {
			event := eventsMap[int64(i)]
			stateName := GetStateName(event, msg, eventsMap)
			listItems = append(listItems, sfnExecutionHistoryItem{event: &sfnHistoryState{ID: event.Id, Step: &stateName, Type: event.Type, Timestamp: event.Timestamp}})
		}
		m.executionHistoryList.SetItems(listItems)
		m.status = "Ready"
		m.err = nil
		return m, nil
	case messages.ErrMsg:
		m.err = msg
		m.status = "Error"
		return m, nil
	}
	if m.state == sfnStateList {
		m.sfnList, cmd = m.sfnList.Update(msg)
	} else if m.state == sfnStateExecutions {
		m.header = append(m.header, aws.StringValue(m.selectedStateMachine.Name), "Executions")
		m.executionList, cmd = m.executionList.Update(msg)
	} else if m.state == sfnStateExecutionDetails {
		m.header = append(m.header, aws.StringValue(m.selectedStateMachine.Name), "Executions", aws.StringValue(m.selectedExecution.Name), "History")
		m.executionHistoryList, cmd = m.executionHistoryList.Update(msg)
	}
	return m, cmd
}

func (m sfnModel) View() string {
	var s string
	switch m.state {
	case sfnStateList:
		if len(m.sfnList.Items()) == 0 && m.status == "Ready" {
			s = styles.StatusStyle.Render("No Step Functions state machines found in this region.\n")
		} else {
			s = m.sfnList.View()
		}
	case sfnStateExecutions:
		if len(m.executionList.Items()) == 0 && m.status == "Ready" {
			s = styles.StatusStyle.Render("No executions found for this state machine.\n")
		} else {
			s = m.executionList.View()
		}
	case sfnStateExecutionDetails:
		if len(m.executionHistoryList.Items()) == 0 && m.status == "Ready" {
			s = styles.StatusStyle.Render("No execution history found for this execution.\n")
		} else {
			s = m.executionHistoryList.View()
		}
	}

	return s
}

type sfnExecutionHistoryItem struct {
	event *sfnHistoryState
}

func (i sfnExecutionHistoryItem) FilterValue() string {
	return aws.StringValue(i.event.Step) + " " + aws.StringValue(i.event.Type)
}

func (i sfnExecutionHistoryItem) Title() string {
	return aws.StringValue(i.event.Step)
}

func (i sfnExecutionHistoryItem) Description() string {
	return fmt.Sprintf("ID: %d | Type: %s | Timestamp %s", aws.Int64Value(i.event.ID), aws.StringValue(i.event.Type), i.event.Timestamp.Local().Format("2006-01-02 15:04:05"))
}

type sfnHistoryState struct {
	ID        *int64
	Type      *string
	Step      *string
	Timestamp *time.Time
}

func GetStateName(event *sfn.HistoryEvent, events messages.SfnExecutionHistoryFetchedMsg, eventsMap map[int64]*sfn.HistoryEvent) string {
	if aws.Int64Value(event.Id) == 1 {
		return "State Machine"
	}
	switch aws.StringValue(event.Type) {
	// Direct state events
	case sfn.HistoryEventTypeTaskStateEntered:
		return aws.StringValue(event.StateEnteredEventDetails.Name)
	case sfn.HistoryEventTypeTaskStateExited:
		return aws.StringValue(event.StateExitedEventDetails.Name)
	case sfn.HistoryEventTypePassStateEntered:
		return aws.StringValue(event.StateEnteredEventDetails.Name)
	case sfn.HistoryEventTypePassStateExited:
		return aws.StringValue(event.StateExitedEventDetails.Name)
	case sfn.HistoryEventTypeChoiceStateEntered:
		return aws.StringValue(event.StateEnteredEventDetails.Name)
	case sfn.HistoryEventTypeChoiceStateExited:
		return aws.StringValue(event.StateExitedEventDetails.Name)
	case sfn.HistoryEventTypeParallelStateEntered:
		return aws.StringValue(event.StateEnteredEventDetails.Name)
	case sfn.HistoryEventTypeParallelStateExited:
		return aws.StringValue(event.StateExitedEventDetails.Name)
	case sfn.HistoryEventTypeMapStateEntered:
		return aws.StringValue(event.StateEnteredEventDetails.Name)
	case sfn.HistoryEventTypeMapStateExited:
		return aws.StringValue(event.StateExitedEventDetails.Name)

	// Task and Lambda failures/successes
	default:
		return walkBackToStateEntered(event, eventsMap)
	}

	return ""
}

// walkBackToStateEntered scans backward in the event list until it finds the latest TaskStateEntered
func walkBackToStateEntered(event *sfn.HistoryEvent, history map[int64]*sfn.HistoryEvent) string {
	if event.Id == nil {
		return ""
	}
	for i := aws.Int64Value(event.PreviousEventId); i != 0; {
		cur := history[i]
		if aws.StringValue(cur.Type) == sfn.HistoryEventTypeTaskStateEntered {
			return aws.StringValue(cur.StateEnteredEventDetails.Name)
		}
		i = aws.Int64Value(cur.PreviousEventId)
	}
	// for i := len(history) - 1; i >= 0; i-- {
	// 	e := history[i]
	// 	if e.Id != nil && *e.Id < *event.Id && aws.StringValue(e.Type) == sfn.HistoryEventTypeTaskStateEntered {
	// 		return aws.StringValue(e.StateEnteredEventDetails.Name)
	// 	}
	// }
	return ""
}
