package telemetry

import (
	"context"
	"time"

	"orcn/services/node-agent/worker"
)

// Collector is an autonomous component that samples node telemetry and populates an in-memory buffer.
type Collector struct {
	loop    *worker.Loop
	sampler *Sampler
	buffer  *Buffer
}

// NewCollector creates a new Telemetry Collector component.
func NewCollector(interval time.Duration, bufferLimit int) *Collector {
	c := &Collector{
		sampler: NewSampler(),
		buffer:  NewBuffer(bufferLimit),
	}
	c.loop = worker.NewLoop("Telemetry_Collector", interval, c.sample)
	return c
}

// New is an alias for NewCollector for backwards compatibility.
func New(interval time.Duration, bufferLimit int) *Collector {
	return NewCollector(interval, bufferLimit)
}

func (c *Collector) Name() string {
	return c.loop.Name()
}

func (c *Collector) Start(ctx context.Context) error {
	return c.loop.Start(ctx)
}

func (c *Collector) Stop(ctx context.Context) error {
	return c.loop.Stop(ctx)
}

// Buffer returns access to the in-memory telemetry buffer.
func (c *Collector) Buffer() *Buffer {
	return c.buffer
}

func (c *Collector) sample(ctx context.Context) error {
	sample := c.sampler.Sample(ctx)
	host := c.sampler.SampleHost(ctx, sample)
	c.buffer.Push(sample, host)
	return nil
}
