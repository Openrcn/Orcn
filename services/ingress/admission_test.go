package ingress

import (
	"errors"
	"testing"
)

func testAdmissionConfig() Config {
	return Config{}
}

func TestAdmissionIsAggregatePerRoute(t *testing.T) {
	c := NewAdmissionController(testAdmissionConfig())
	release, err := c.Acquire("llama.acme", "model_inference", AdmissionPolicy{MaxConcurrentRequests: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	if _, err := c.Acquire("llama.acme", "model_inference", AdmissionPolicy{MaxConcurrentRequests: 1}); !errors.Is(err, ErrConcurrencyLimited) {
		t.Fatalf("got %v, want concurrency limit", err)
	}
	otherRelease, err := c.Acquire("embeddings.acme", "model_inference", AdmissionPolicy{MaxConcurrentRequests: 1})
	if err != nil {
		t.Fatalf("a different deployment must have its own budget: %v", err)
	}
	otherRelease()
}

func TestAdmissionRateLimit(t *testing.T) {
	c := NewAdmissionController(testAdmissionConfig())
	release, err := c.Acquire("llama.acme", "model_inference", AdmissionPolicy{RequestsPerMinute: 1, BurstRequests: 1})
	if err != nil {
		t.Fatal(err)
	}
	release()
	if _, err := c.Acquire("llama.acme", "model_inference", AdmissionPolicy{RequestsPerMinute: 1, BurstRequests: 1}); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("got %v, want rate limit", err)
	}
}

func TestAdmissionOptionalGlobalLimit(t *testing.T) {
	cfg := testAdmissionConfig()
	c := NewAdmissionController(cfg)
	release, err := c.Acquire("ui.acme", "workload", AdmissionPolicy{RequestsPerMinute: 100, BurstRequests: 100})
	if err != nil {
		t.Fatal(err)
	}
	release()
	if _, err := c.Acquire("docs.acme", "workload", AdmissionPolicy{RequestsPerMinute: 100, BurstRequests: 100}); err != nil {
		t.Fatalf("got %v, want nil global limit", err)
	}
}
