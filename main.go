package main

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	fmt.Print("\033[H\033[2J")
	// Initialize AWS session
	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}

	// Create AWS service clients
	ec2Svc := ec2.New(sess)
	ecsSvc := ecs.New(sess)
	cloudwatchlogsSvc := cloudwatchlogs.New(sess)

	// Initialize the spinner model
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(tokyoNightCyan)

	var listkeys = newListKeyMap()

	// Initialize the EC2 list model
	ec2List := list.New([]list.Item{}, itemDelegate{}, 0, 20)
	ec2List.Title = "EC2 Instances"
	ec2List.SetShowStatusBar(false)
	ec2List.SetFilteringEnabled(true)
	ec2List.Styles.Title = headerStyle
	ec2List.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(tokyoNightGreen)
	ec2List.Styles.FilterCursor = lipgloss.NewStyle().Foreground(tokyoNightGreen)
	ec2List.Styles.NoItems = statusStyle.UnsetPaddingLeft()
	ec2List.SetStatusBarItemName("instance", "instances")
	ec2List.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.details,
			listkeys.start,
			listkeys.stop,
			listkeys.ssh,
			listkeys.refresh,
		}
	}

	// Initialize the ECS cluster list model
	ecsClusterList := list.New([]list.Item{}, itemDelegate{}, 0, 20)
	ecsClusterList.Title = "ECS Clusters"
	ecsClusterList.SetShowStatusBar(false)
	ecsClusterList.SetFilteringEnabled(true)
	ecsClusterList.Styles.Title = headerStyle
	ecsClusterList.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(tokyoNightGreen)
	ecsClusterList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(tokyoNightGreen)
	ecsClusterList.Styles.NoItems = statusStyle.UnsetPaddingLeft()
	ecsClusterList.SetStatusBarItemName("cluster", "clusters")
	ecsClusterList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.refresh,
		}
	}

	// Initialize the ECS service list model
	ecsServiceList := list.New([]list.Item{}, itemDelegate{}, 0, 20)
	ecsServiceList.Title = "ECS Services"
	ecsServiceList.SetShowStatusBar(false)
	ecsServiceList.SetFilteringEnabled(true)
	ecsServiceList.Styles.Title = headerStyle
	ecsServiceList.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(tokyoNightGreen)
	ecsServiceList.Styles.FilterCursor = lipgloss.NewStyle().Foreground(tokyoNightGreen)
	ecsServiceList.Styles.NoItems = statusStyle.UnsetPaddingLeft()
	ecsServiceList.SetStatusBarItemName("service", "services")
	ecsServiceList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listkeys.details,
			listkeys.stop,
			listkeys.refresh,
			listkeys.logs,
		}
	}

	// Create initial model
	m := model{
		ec2Svc:            ec2Svc,
		ecsSvc:            ecsSvc,
		cloudwatchlogsSvc: cloudwatchlogsSvc,
		status:            "Select an option.",
		spinner:           s,
		instanceList:      ec2List,
		clusterList:       ecsClusterList,
		serviceList:       ecsServiceList,
		keys:              listkeys,
		state:             stateMenu,
		menuChoices:       []string{"EC2 Instances", "ECS Clusters"},
		menuCursor:        0,
	}

	// Start the Bubble Tea program
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
}
