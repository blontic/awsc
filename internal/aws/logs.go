package aws

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/blontic/swa/internal/config"
	"github.com/blontic/swa/internal/ui"
)

type LogsManager struct {
	client *cloudwatchlogs.Client
	region string
}

type LogsManagerOptions struct {
	Client *cloudwatchlogs.Client
	Region string
}

func NewLogsManager(ctx context.Context, opts ...LogsManagerOptions) (*LogsManager, error) {
	if len(opts) > 0 && opts[0].Client != nil {
		return &LogsManager{
			client: opts[0].Client,
			region: opts[0].Region,
		}, nil
	}

	cfg, err := config.LoadSWAConfigWithProfile(ctx)
	if err != nil {
		return nil, err
	}

	return &LogsManager{
		client: cloudwatchlogs.NewFromConfig(cfg),
		region: cfg.Region,
	}, nil
}

func (m *LogsManager) reloadClient(ctx context.Context) error {
	cfg, err := config.LoadSWAConfigWithProfile(ctx)
	if err != nil {
		return err
	}
	m.client = cloudwatchlogs.NewFromConfig(cfg)
	m.region = cfg.Region
	return nil
}

func (m *LogsManager) RunTail(ctx context.Context, groupName string, since string, follow bool) error {
	logGroups, err := m.listAllLogGroups(ctx)
	if err != nil {
		if IsAuthError(err) {
			if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
				if reloadErr := m.reloadClient(ctx); reloadErr != nil {
					return reloadErr
				}
				// Retry after successful re-auth and client reload
				return m.RunTail(ctx, groupName, since, follow)
			}
			return err
		}
		return err
	}

	var selectedGroup *types.LogGroup
	if groupName != "" {
		for _, group := range logGroups {
			if *group.LogGroupName == groupName {
				selectedGroup = &group
				break
			}
		}
		if selectedGroup == nil {
			fmt.Fprintf(os.Stderr, "Log group '%s' not found. Available log groups:\n\n", groupName)
		}
	}

	if selectedGroup == nil {
		if len(logGroups) == 0 {
			fmt.Println("No log groups found")
			return nil
		}

		groupNames := make([]string, len(logGroups))
		for i, group := range logGroups {
			groupNames[i] = *group.LogGroupName
		}

		selectedIndex, err := ui.RunSelector("Select log group:", groupNames)
		if err != nil {
			return err
		}
		if selectedIndex == -1 {
			return nil // User quit
		}

		selectedGroup = &logGroups[selectedIndex]
	}

	fmt.Printf("Selected: %s\n\n", *selectedGroup.LogGroupName)

	return m.tailLogs(ctx, *selectedGroup.LogGroupName, since, follow)
}

func (m *LogsManager) listAllLogGroups(ctx context.Context) ([]types.LogGroup, error) {
	var allGroups []types.LogGroup
	var nextToken *string

	for {
		result, err := m.client.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}

		allGroups = append(allGroups, result.LogGroups...)

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	sort.Slice(allGroups, func(i, j int) bool {
		return *allGroups[i].LogGroupName < *allGroups[j].LogGroupName
	})

	return allGroups, nil
}

func (m *LogsManager) tailLogs(ctx context.Context, groupName string, since string, follow bool) error {
	// Parse since parameter to get start time
	startTime, err := m.parseSince(since)
	if err != nil {
		return fmt.Errorf("invalid since parameter: %v", err)
	}

	// Get initial logs
	if err := m.printLogsFromTime(ctx, groupName, startTime); err != nil {
		return err
	}

	if !follow {
		return nil
	}

	// Follow mode - poll for new logs every 2 seconds
	lastTimestamp := time.Now().UnixMilli()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			newTimestamp, err := m.printNewLogs(ctx, groupName, lastTimestamp)
			if err != nil {
				if IsAuthError(err) {
					if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
						if reloadErr := m.reloadClient(ctx); reloadErr != nil {
							return reloadErr
						}
						// Continue polling after re-auth and client reload
						continue
					}
				}
				return err
			}
			if newTimestamp > lastTimestamp {
				lastTimestamp = newTimestamp
			}
		}
	}
}

func (m *LogsManager) parseSince(since string) (time.Time, error) {
	if since == "" {
		return time.Now().Add(-10 * time.Minute), nil
	}

	// Parse relative time format like "5m", "1h", "2d"
	if len(since) < 2 {
		return time.Time{}, fmt.Errorf("invalid format")
	}

	unit := since[len(since)-1:]
	numberStr := since[:len(since)-1]

	number, err := strconv.Atoi(numberStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid number: %s", numberStr)
	}

	var duration time.Duration
	switch unit {
	case "s":
		duration = time.Duration(number) * time.Second
	case "m":
		duration = time.Duration(number) * time.Minute
	case "h":
		duration = time.Duration(number) * time.Hour
	case "d":
		duration = time.Duration(number) * 24 * time.Hour
	case "w":
		duration = time.Duration(number) * 7 * 24 * time.Hour
	default:
		return time.Time{}, fmt.Errorf("unsupported unit: %s (supported: s, m, h, d, w)", unit)
	}

	return time.Now().Add(-duration), nil
}

func (m *LogsManager) printLogsFromTime(ctx context.Context, groupName string, startTime time.Time) error {
	endTime := time.Now()
	var allEvents []types.FilteredLogEvent
	var nextToken *string

	for {
		result, err := m.client.FilterLogEvents(ctx, &cloudwatchlogs.FilterLogEventsInput{
			LogGroupName: &groupName,
			StartTime:    aws.Int64(startTime.UnixMilli()),
			EndTime:      aws.Int64(endTime.UnixMilli()),
			NextToken:    nextToken,
		})
		if err != nil {
			return err
		}

		allEvents = append(allEvents, result.Events...)

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	if len(allEvents) == 0 {
		fmt.Println("No logs found")
		return nil
	}

	// Sort events by timestamp
	sort.Slice(allEvents, func(i, j int) bool {
		return *allEvents[i].Timestamp < *allEvents[j].Timestamp
	})

	// Print all events
	for _, event := range allEvents {
		timestamp := time.UnixMilli(*event.Timestamp)
		fmt.Printf("%s %s\n", timestamp.Format("2006-01-02 15:04:05"), *event.Message)
	}

	return nil
}

func (m *LogsManager) printNewLogs(ctx context.Context, groupName string, lastTimestamp int64) (int64, error) {
	now := time.Now()
	var allEvents []types.FilteredLogEvent
	var nextToken *string

	for {
		result, err := m.client.FilterLogEvents(ctx, &cloudwatchlogs.FilterLogEventsInput{
			LogGroupName: &groupName,
			StartTime:    aws.Int64(lastTimestamp + 1),
			EndTime:      aws.Int64(now.UnixMilli()),
			NextToken:    nextToken,
		})
		if err != nil {
			return lastTimestamp, err
		}

		allEvents = append(allEvents, result.Events...)

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	sort.Slice(allEvents, func(i, j int) bool {
		return *allEvents[i].Timestamp < *allEvents[j].Timestamp
	})

	newLastTimestamp := lastTimestamp
	for _, event := range allEvents {
		timestamp := time.UnixMilli(*event.Timestamp)
		fmt.Printf("%s %s\n", timestamp.Format("2006-01-02 15:04:05"), *event.Message)
		if *event.Timestamp > newLastTimestamp {
			newLastTimestamp = *event.Timestamp
		}
	}

	return newLastTimestamp, nil
}
