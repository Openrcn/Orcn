package local

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"orcn/core"
	"orcn/pkg/util"
)

const (
	logChannelBuffer = 256
	logTailLines     = 200
	logScanMaxToken  = 1024 * 1024

	logBackoffMin = time.Second
	logBackoffMax = 30 * time.Second
)

type localLogAdapter struct {
	client *Client

	machine  *MachineConfig
	jobName  string
	nodeName string

	events chan core.LogEvent

	cancel    context.CancelFunc
	wg        sync.WaitGroup
	closeOnce sync.Once
}

func (c *Client) NewLogAdapter(providerJobID string) (core.LogAdapter, error) {
	if providerJobID == "" {
		return nil, fmt.Errorf("provider job id is required")
	}

	machineID, jobName := util.SplitJobKey(providerJobID)
	if machineID == "" || jobName == "" {
		return nil, fmt.Errorf("invalid provider job id %q", providerJobID)
	}

	machine, err := c.getMachineByID(machineID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	adapter := &localLogAdapter{
		client:   c,
		machine:  machine,
		jobName:  jobName,
		nodeName: nodeLabel(machine),
		events:   make(chan core.LogEvent, logChannelBuffer),
		cancel:   cancel,
	}

	job, err := c.fetchRegistration(machine, jobName)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load registration for %s: %w", jobName, err)
	}

	containers := containerIDs(job)
	if len(containers) == 0 {
		adapter.emit(ctx, core.LogEvent{
			Node:      adapter.nodeName,
			Container: "system",
			Timestamp: time.Now().UTC(),
			Log:       "no containers registered for this workload",
			Source:    core.LogSourceSystem,
		})
		close(adapter.events)
		return adapter, nil
	}

	adapter.wg.Add(len(containers))
	for _, containerID := range containers {
		go adapter.stream(ctx, containerID)
	}

	go func() {
		adapter.wg.Wait()
		close(adapter.events)
	}()

	return adapter, nil
}

func (a *localLogAdapter) Events() <-chan core.LogEvent {
	return a.events
}

func (a *localLogAdapter) Close() error {
	a.closeOnce.Do(a.cancel)
	return nil
}

func (a *localLogAdapter) emit(ctx context.Context, ev core.LogEvent) {
	select {
	case a.events <- ev:
	case <-ctx.Done():
	}
}

func (a *localLogAdapter) emitSystem(ctx context.Context, message string) {
	a.emit(ctx, core.LogEvent{
		Node:      a.nodeName,
		Container: "system",
		Timestamp: time.Now().UTC(),
		Log:       message,
		Source:    core.LogSourceSystem,
	})
}

func (a *localLogAdapter) stream(ctx context.Context, containerID string) {
	defer a.wg.Done()

	backoff := logBackoffMin
	for {
		if ctx.Err() != nil {
			return
		}

		err := a.connectAndRead(ctx, containerID)
		if ctx.Err() != nil {
			return
		}

		if err != nil {
			a.emitSystem(ctx, fmt.Sprintf("log stream interrupted: %v; reconnecting in %s", err, backoff))
		}

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
		if backoff < logBackoffMax {
			backoff *= 2
			if backoff > logBackoffMax {
				backoff = logBackoffMax
			}
		}
	}
}

// connectAndRead opens the node-agent log stream for a container and emits each
// line until the stream ends or the context is cancelled.
func (a *localLogAdapter) connectAndRead(ctx context.Context, containerID string) error {
	client := NewNodeAgentClient(a.machine.Endpoint, a.machine.Token)
	respBody, err := client.GetLogs(ctx, containerID, logTailLines)
	if err != nil {
		return err
	}
	defer respBody.Close()

	scanner := bufio.NewScanner(respBody)
	scanner.Buffer(make([]byte, 0, 64*1024), logScanMaxToken)
	for scanner.Scan() {
		a.handleLine(ctx, containerID, scanner.Text())
	}
	return scanner.Err()
}

func (a *localLogAdapter) handleLine(ctx context.Context, containerID, raw string) {
	line := cleanLine(raw)
	if line == "" {
		return
	}

	ts, message := splitTimestamp(line)
	a.emit(ctx, core.LogEvent{
		Node:      a.nodeName,
		Container: containerID,
		Timestamp: ts,
		Log:       message,
		Source:    core.LogSourceApp,
	})
}

func containerIDs(job *core.JobSpec) []string {
	if job == nil {
		return nil
	}
	ids := make([]string, 0, len(job.Containers))
	for _, container := range job.Containers {
		if container.ID != "" {
			ids = append(ids, container.ID)
		}
	}
	return ids
}

func nodeLabel(machine *MachineConfig) string {
	if machine == nil {
		return "local"
	}
	if machine.Name != "" {
		return machine.Name
	}
	if len(machine.ID) > 8 {
		return machine.ID[:8]
	}
	return machine.ID
}

func splitTimestamp(line string) (time.Time, string) {
	if i := strings.IndexByte(line, ' '); i > 0 {
		if ts, err := time.Parse(time.RFC3339Nano, line[:i]); err == nil {
			return ts.UTC(), strings.TrimLeft(line[i+1:], " ")
		}
	}
	return time.Now().UTC(), line
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

func cleanLine(s string) string {
	s = ansiPattern.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\r", "")
	return strings.TrimSpace(s)
}
