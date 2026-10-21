package local

import (
	"context"
	"fmt"
	"strings"

	"orcn/core"
	"orcn/models"
	"orcn/pkg/util"
)

func (c *Client) CreateDeployment(name, instanceTypeID string, spec *core.JobSpec) (string, error) {
	machine, err := c.getMachineByID(instanceTypeID)
	if err != nil {
		return "", err
	}
	jobName := util.SanitizeJobName(name)
	if err := c.registerJob(machine, jobName, spec); err != nil {
		return "", err
	}
	return util.JobKey(machine.ID, jobName), nil
}

func (c *Client) StartDeployment(providerJobID string) error {
	return fmt.Errorf("use StartDeploymentWithSpec")
}

func (c *Client) StopDeployment(providerJobID string) error {
	return fmt.Errorf("use StopDeploymentWithSpec")
}

func (c *Client) StartDeploymentWithSpec(providerJobID string, spec *core.JobSpec) error {
	machineID, jobName := util.SplitJobKey(providerJobID)
	machine, err := c.getMachineByID(machineID)
	if err != nil {
		return err
	}
	return c.registerJob(machine, jobName, spec)
}

func (c *Client) StopDeploymentWithSpec(providerJobID string, spec *core.JobSpec) error {
	machineID, jobName := util.SplitJobKey(providerJobID)
	machine, err := c.getMachineByID(machineID)
	if err != nil {
		return err
	}
	return c.unregisterFromMachine(machine, jobName)
}

func (c *Client) UpdateTimeout(deploymentID string, timeoutMinutes int) error {
	return nil
}

func (c *Client) GetNodeInfo(providerJobID string) (*core.NodeInfo, error) {
	machineID, jobName := util.SplitJobKey(providerJobID)
	machine, err := c.getMachineByID(machineID)
	if err != nil {
		return &core.NodeInfo{Status: models.InfraPending}, nil
	}

	status, err := c.fetchDeploymentHealth(machine, jobName)
	if err != nil {
		return &core.NodeInfo{Status: models.InfraPending}, nil
	}

	info := &core.NodeInfo{Status: status}

	if base := c.proxyBase(machine); base != "" {
		if job, err := c.fetchRegistration(machine, jobName); err == nil && job != nil {
			info.Endpoints = buildJobEndpoints(base, job)
		}
	}

	return info, nil
}

func (c *Client) GetNodeMetrics(deploymentID string) (*core.NodeMetrics, error) {
	return &core.NodeMetrics{}, nil
}

func (c *Client) proxyBase(machine *MachineConfig) string {
	return machine.ProxyEndpoint
}

func (c *Client) fetchRegistration(machine *MachineConfig, jobName string) (*core.JobSpec, error) {
	ctx := context.Background()
	client := NewNodeAgentClient(machine.Endpoint, machine.Token)
	workloads, err := client.GetRegistrations(ctx)
	if err != nil {
		return nil, err
	}
	job, ok := workloads[jobName]
	if !ok {
		return nil, nil
	}
	return &job, nil
}

func buildJobEndpoints(proxyBase string, job *core.JobSpec) []core.Endpoint {
	var endpoints []core.Endpoint
	for _, container := range job.Containers {
		for _, expose := range container.Args.Expose {
			if !expose.IsPublic {
				continue
			}
			proto := expose.Protocol
			if proto == "" {
				proto = "http"
			}
			endpoints = append(endpoints, core.Endpoint{
				Port:        expose.Port,
				BaseURL:     proxyBase,
				Protocol:    proto,
				ContainerID: container.ID,
			})
		}
	}
	return endpoints
}

func (c *Client) fetchDeploymentHealth(machine *MachineConfig, jobName string) (string, error) {
	ctx := context.Background()
	client := NewNodeAgentClient(machine.Endpoint, machine.Token)
	data, err := client.GetHealth(ctx, util.URLQueryEscape(jobName))
	if err != nil {
		return "", err
	}

	raw, _ := data["status"].(string)
	switch strings.ToLower(raw) {
	case "healthy", "degraded":
		return models.InfraRunning, nil
	case "down", "unhealthy":
		return models.InfraRunning, nil
	case "unknown":
		return models.InfraPending, nil
	default:
		return models.InfraPending, nil
	}
}

func (c *Client) registerJob(machine *MachineConfig, jobName string, spec *core.JobSpec) error {
	ctx := context.Background()
	client := NewNodeAgentClient(machine.Endpoint, machine.Token)
	return client.RegisterJob(ctx, jobName, spec)
}

func (c *Client) unregisterFromMachine(machine *MachineConfig, jobName string) error {
	ctx := context.Background()
	client := NewNodeAgentClient(machine.Endpoint, machine.Token)
	return client.UnregisterJob(ctx, jobName)
}