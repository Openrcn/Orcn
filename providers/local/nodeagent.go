package local

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"orcn/core"
)

type NodeAgentClient struct {
	endpoint string
	token    string
	client   *http.Client
}

func NewNodeAgentClient(endpoint, token string) *NodeAgentClient {
	return &NodeAgentClient{
		endpoint: endpoint,
		token:    token,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (n *NodeAgentClient) withAuth(req *http.Request) *http.Request {
	if n.token != "" && n.token != "[encrypted]" {
		req.Header.Set("Authorization", "Bearer "+n.token)
	}
	return req
}

func (n *NodeAgentClient) doRequest(ctx context.Context, method, path string, body any, timeout time.Duration) ([]byte, error) {
	url := n.endpoint + path
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req = n.withAuth(req)

	client := n.client
	if timeout > 0 {
		client = &http.Client{Timeout: timeout}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node-agent api error: %s - %s", resp.Status, string(b))
	}

	return io.ReadAll(resp.Body)
}

func (n *NodeAgentClient) GetCapabilities(ctx context.Context) (map[string]any, error) {
	b, err := n.doRequest(ctx, "GET", "/capabilities", nil, 0)
	if err != nil {
		return nil, err
	}
	var caps map[string]any
	if err := json.Unmarshal(b, &caps); err != nil {
		return nil, err
	}
	return caps, nil
}

func (n *NodeAgentClient) GetRegistrations(ctx context.Context) (map[string]core.JobSpec, error) {
	b, err := n.doRequest(ctx, "GET", "/registrations", nil, 0)
	if err != nil {
		return nil, err
	}
	var data struct {
		Workloads map[string]core.JobSpec `json:"workloads"`
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return data.Workloads, nil
}

func (n *NodeAgentClient) GetHealth(ctx context.Context, deployment string) (map[string]any, error) {
	path := "/health?deployment=" + deployment
	b, err := n.doRequest(ctx, "GET", path, nil, 0)
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (n *NodeAgentClient) RegisterJob(ctx context.Context, jobName string, spec *core.JobSpec) error {
	job := *spec
	job.JobName = jobName
	_, err := n.doRequest(ctx, "POST", "/register", &job, 60*time.Second)
	return err
}

func (n *NodeAgentClient) UnregisterJob(ctx context.Context, jobName string) error {
	path := "/register/" + jobName
	_, err := n.doRequest(ctx, "DELETE", path, nil, 30*time.Second)
	return err
}

func (n *NodeAgentClient) GetLogs(ctx context.Context, containerID string, tail int) (io.ReadCloser, error) {
	path := fmt.Sprintf("/containers/%s/logs?live=true&tail=%d", containerID, tail)
	url := n.endpoint + path

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req = n.withAuth(req)

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, err
	}

	switch {
	case resp.StatusCode == http.StatusForbidden:
		resp.Body.Close()
		return nil, fmt.Errorf("node-agent rejected the log request (workload not registered)")
	case resp.StatusCode == http.StatusNotFound:
		resp.Body.Close()
		return nil, fmt.Errorf("container %q not found on node-agent", containerID)
	case resp.StatusCode >= 400:
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		return nil, fmt.Errorf("node-agent logs failed: %s: %s", resp.Status, string(b))
	}

	return resp.Body, nil
}