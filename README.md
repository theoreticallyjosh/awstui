# awstui

A terminal user interface (TUI) for interacting with AWS services, built with Go and Bubble Tea.

**DISCLAIMER**: This was my first attempt at "vibe coding". I wanted to see if it was as good/bad people are claiming it to be. I made some manually tweaks a well.

## Features

- Navigate and manage AWS resources (e.g., EC2, ECS, ECR).
- Interactive and responsive TUI experience.
- Lightweight and fast.

### EC2

- [x] List instances
- [x] View details
- [x] Start instance
- [x] Stop instance
- [x] SSH into instance

### ECS

- [x] List clusters
- [x] List services
- [x] View service details
- [x] View Logs
- [x] Trigger service redeployment
- [x] Stop service (scale to 0)
- [ ] Scale service

### ECR

- [x] View private repositories
- [x] View images
- [x] Pull image
- [x] Push image

### Step Functions

- [x] View
- [ ] Start New Execution
- [x] View Executions
- [x] View Execution Steps

### Batch

- [x] View Queues
- [x] View Batch Jobs
- [x] View Batch Job Logs
- [ ] Execute Batch Jobs

## Installation

To install `awstui`, make sure you have Go installed (version 1.16 or higher is recommended).

```bash
go install github.com/theoreticallyjosh/awstui@latest
```

## Configuration

Default path for the global config file:

- Linux: `~/.config/awstui/config.yml`
- MacOS: `~/Library/Application\ Support/awstui/config.yml`
- Windows: `%LOCALAPPDATA%\awstui\config.yml`

### AWS

To use awstui, you will need to configure your AWS credentials. You can follow the same steps required of [aws-cli](https://github.com/aws/aws-cli#configuration).

### Theme

You can set the color theme in the awstui config.yml file:

```
theme: <bubble_tint_id>

```

A list of available themes/tints can be found [here](https://github.com/lrstanley/bubbletint/blob/master/DEFAULT_TINTS.md).

## Usage

After installation, you can run `awstui` from your terminal:

```bash
awstui
```

## Screenshots

![Demo](demo.gif "Demo")

## Contributing

Contributions are welcome! Please feel free to open issues or submit pull requests.

## License

This project is licensed under the [MIT License](LICENSE).
