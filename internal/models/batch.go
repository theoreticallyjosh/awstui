package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/theoreticallyjosh/awstui/internal/commands"
	"github.com/theoreticallyjosh/awstui/internal/keys"
	"github.com/theoreticallyjosh/awstui/internal/messages"
	"github.com/theoreticallyjosh/awstui/internal/styles"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
)

type batchState int

const (
	batchStateJobQueueList batchState = iota
	batchStateJobList
	batchStateJobDetails
	batchStateJobLogs
)

type batchModel struct {
	parent            *Model
	batchSvc          *batch.Batch
	cloudwatchlogsSvc *cloudwatchlogs.CloudWatchLogs
	jobQueueList      list.Model
	jobList           list.Model
	status            string
	err               error
	keys              *keys.ListKeyMap
	paginator         paginator.Model
	state             batchState
	header            []string
	detailJobQueue    *batch.JobQueueDetail
	detailJob         *batch.JobDetail
	jobLogs           string
	confirming        bool
	action            string
	actionID          *string
	getLogs           bool
}

// item delegates
type batchJobQueueItem struct {
	jobQueue *batch.JobQueueDetail
}

func (i batchJobQueueItem) Title() string {
	return aws.StringValue(i.jobQueue.JobQueueName)
}

func (i batchJobQueueItem) Description() string {
	return fmt.Sprintf("Status: %s", aws.StringValue(i.jobQueue.Status))
}

func (i batchJobQueueItem) FilterValue() string {
	return aws.StringValue(i.jobQueue.JobQueueName)
}

type batchJobItem struct {
	job *batch.JobSummary
}

func (i batchJobItem) Title() string {
	return aws.StringValue(i.job.JobName)
}

func (i batchJobItem) Description() string {
	return fmt.Sprintf("ID: %s | Status: %s", aws.StringValue(i.job.JobId), aws.StringValue(i.job.Status))
}

func (i batchJobItem) FilterValue() string {
	return aws.StringValue(i.job.JobName)
}

func (m batchModel) Init() tea.Cmd {
	return tea.Batch(m.parent.spinner.Tick, commands.FetchBatchJobQueuesCmd(m.batchSvc))
}

func (m batchModel) Update(msg tea.Msg) (batchModel, tea.Cmd) {
	var cmd tea.Cmd
	m.header = []string{"Batch Job Queues"}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.paginator.PerPage = msg.Height - 2
		m.jobQueueList.SetSize(msg.Width, msg.Height)
		m.jobList.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))):
			if m.state == batchStateJobList {
				m.state = batchStateJobQueueList
				m.status = "Ready"
				m.err = nil
				m.jobList.SetItems([]list.Item{})
				return m, nil
			} else if m.state == batchStateJobDetails {
				m.state = batchStateJobList
				m.status = "Ready"
				m.err = nil
				m.detailJob = nil
				return m, nil
			} else if m.state == batchStateJobLogs {
				m.state = batchStateJobList
				m.status = "Ready"
				m.err = nil
				m.jobLogs = ""
				return m, nil
			}
		}
		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.status = fmt.Sprintf("%sing job %s...", m.action, *m.actionID)
				m.err = nil
				if m.action == "stop" {
					reason := "Terminated by user"
					return m, tea.Batch(m.parent.spinner.Tick, commands.StopBatchJobCmd(m.batchSvc, m.actionID, &reason))
				}
			case "n", "N":
				m.confirming = false
				m.status = "Ready"
				m.action = ""
				m.state = batchStateJobList
			}
			return m, nil
		}

		switch m.state {
		case batchStateJobQueueList:
			if m.jobQueueList.FilterState() == list.Filtering {
				break
			}
			switch {
			case key.Matches(msg, m.keys.Refresh):
				m.status = "Refreshing Batch job queues..."
				m.err = nil
				return m, tea.Batch(m.parent.spinner.Tick, commands.FetchBatchJobQueuesCmd(m.batchSvc))
			case key.Matches(msg, m.keys.Choose):
				if m.jobQueueList.SelectedItem() != nil {
					selectedItem := m.jobQueueList.SelectedItem().(batchJobQueueItem)
					m.detailJobQueue = selectedItem.jobQueue
					m.state = batchStateJobList
					m.status = fmt.Sprintf("Loading jobs for job queue %s...", aws.StringValue(m.detailJobQueue.JobQueueName))
					return m, tea.Batch(m.parent.spinner.Tick, commands.FetchBatchJobsCmd(m.batchSvc, m.detailJobQueue.JobQueueName))
				}
			}
		case batchStateJobList:
			m.header = append(m.header, aws.StringValue(m.detailJobQueue.JobQueueName), "Jobs")
			if m.jobList.FilterState() == list.Filtering {
				break
			}
			switch {
			case key.Matches(msg, m.keys.Refresh):
				m.status = fmt.Sprintf("Refreshing jobs for job queue %s...", aws.StringValue(m.detailJobQueue.JobQueueName))
				m.err = nil
				return m, tea.Batch(m.parent.spinner.Tick, commands.FetchBatchJobsCmd(m.batchSvc, m.detailJobQueue.JobQueueName))
			case key.Matches(msg, m.keys.Stop):
				if m.jobList.SelectedItem() != nil {
					selectedItem := m.jobList.SelectedItem().(batchJobItem)
					selectedJob := selectedItem.job
					m.confirming = true
					m.action = "stop"
					m.actionID = selectedJob.JobId
					m.status = fmt.Sprintf("Confirm stopping job %s (%s)? (y/N)",
						aws.StringValue(selectedJob.JobName), aws.StringValue(selectedJob.JobId))
				}
			case key.Matches(msg, m.keys.Logs):
				if m.jobList.SelectedItem() != nil {
					m.getLogs = true
					selectedItem := m.jobList.SelectedItem().(batchJobItem)
					m.status = fmt.Sprintf("Fetching logs for job %s...", aws.StringValue(selectedItem.job.JobName))
					return m, tea.Batch(m.parent.spinner.Tick, commands.FetchBatchJobDetailsCmd(m.batchSvc, selectedItem.job.JobId))
				}
			case key.Matches(msg, m.keys.Details):
				if m.jobList.SelectedItem() != nil {
					selectedItem := m.jobList.SelectedItem().(batchJobItem)
					m.status = fmt.Sprintf("Fetching details for job %s...", aws.StringValue(selectedItem.job.JobName))
					return m, tea.Batch(m.parent.spinner.Tick, commands.FetchBatchJobDetailsCmd(m.batchSvc, selectedItem.job.JobId))
				}

			}
		case batchStateJobDetails:
			// No key handling in these states for now
		case batchStateJobLogs:
			m.header = append(m.header, aws.StringValue(m.detailJobQueue.JobQueueName), m.jobList.SelectedItem().FilterValue(), "Logs")
			m.paginator, cmd = m.paginator.Update(msg)
			return m, cmd
		}

	case messages.BatchJobQueuesFetchedMsg:
		listItems := make([]list.Item, len(msg))
		for i, jobQueue := range msg {
			listItems[i] = batchJobQueueItem{jobQueue: jobQueue}
		}
		m.jobQueueList.SetItems(listItems)
		m.status = "Ready"
		m.err = nil
		return m, nil
	case messages.BatchJobsFetchedMsg:
		m.header = append(m.header, m.jobQueueList.SelectedItem().FilterValue(), "Jobs")
		listItems := make([]list.Item, len(msg))
		for i, job := range msg {
			listItems[i] = batchJobItem{job: job}
		}
		m.jobList.SetItems(listItems)
		m.status = "Ready"
		m.err = nil
		return m, nil
	case messages.BatchJobDetailsMsg:
		m.detailJob = msg
		if msg.Container == nil {
			m.status = "No details found"
			m.err = fmt.Errorf("container is nil")
			return m, nil
		}
		m.state = batchStateJobDetails
		if m.getLogs {
			m.getLogs = false
			m.state = batchStateJobLogs
			return m, tea.Batch(m.parent.spinner.Tick, commands.FetchBatchJobLogsCmd(m.cloudwatchlogsSvc, m.detailJob.Container.LogStreamName))
		}
		m.status = "Ready"
		return m, nil
	case messages.BatchJobLogsFetchedMsg:
		m.header = append(m.header, aws.StringValue(m.detailJobQueue.JobQueueName), m.jobList.SelectedItem().FilterValue(), "Logs")
		m.jobLogs = string(msg)
		m.paginator.SetTotalPages(len(strings.Split(m.jobLogs, "\n")))
		m.status = "Ready"
		m.err = nil
		return m, nil
	case messages.BatchJobActionMsg:
		m.status = fmt.Sprintf("Job %s %s. Refreshing...", *m.actionID, msg)
		m.err = nil
		m.action = ""
		m.actionID = nil
		m.confirming = false
		return m, tea.Batch(m.parent.spinner.Tick, commands.FetchBatchJobsCmd(m.batchSvc, m.detailJobQueue.JobQueueName))
	case messages.ErrMsg:
		m.err = msg
		m.status = "Error"
		m.confirming = false
		m.action = ""
		m.detailJob = nil
		m.jobLogs = ""
		return m, nil
	}

	if m.state == batchStateJobQueueList {
		m.jobQueueList, cmd = m.jobQueueList.Update(msg)
	} else if m.state == batchStateJobList {
		m.jobList, cmd = m.jobList.Update(msg)
	}
	return m, cmd
}

func (m batchModel) View() string {
	var s string
	switch m.state {
	case batchStateJobQueueList:
		if len(m.jobQueueList.Items()) == 0 && m.status == "Ready" {
			s = "No Batch job queues found in this region.\n"
		} else {
			s = m.jobQueueList.View()
		}
	case batchStateJobList:
		if len(m.jobList.Items()) == 0 && m.status == "Ready" {
			s = "No Batch jobs found in this job queue.\n"
		} else {
			s = m.jobList.View()
		}
	case batchStateJobDetails:
		if m.detailJob != nil {
			s += "\n" + styles.DetailStyle.Render(
				fmt.Sprintf("Job Name:      %s\n", aws.StringValue(m.detailJob.JobName))+
					fmt.Sprintf("Job ID:        %s\n", aws.StringValue(m.detailJob.JobId))+
					fmt.Sprintf("Status:        %s\n", aws.StringValue(m.detailJob.Status))+
					fmt.Sprintf("Created At:    %s\n", time.Unix(aws.Int64Value(m.detailJob.CreatedAt)/1000, 0).Format(time.RFC822))+
					fmt.Sprintf("Stopped At:    %s\n", time.Unix(aws.Int64Value(m.detailJob.StoppedAt)/1000, 0).Format(time.RFC822))+"\nPress 'esc' or 'backspace' to go back."+"\n",
			)
		} else {
			s = styles.StatusStyle.Render("No job details available.\n")
		}
	case batchStateJobLogs:
		if m.jobLogs == "" && m.status == "Ready" {
			s += styles.StatusStyle.Render("No logs found for this job.\n")
		} else {
			lines := strings.Split(m.jobLogs, "\n")
			start, end := m.paginator.GetSliceBounds(len(lines))
			for _, item := range lines[start:end] {
				s += item + "\n"
			}
		}
		s += m.paginator.View()
		s += "\n" + styles.HelpStyle.Render("Press 'esc' or 'backspace' to go back.")
	}

	return s
}
