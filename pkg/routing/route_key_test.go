package routing

import "testing"

func TestDeploymentRouteKey(t *testing.T) {
	got, err := DeploymentRouteKey("Llama Chat", "Acme Labs")
	if err != nil {
		t.Fatal(err)
	}
	if got != "llama-chat.acme-labs" {
		t.Fatalf("got %q", got)
	}
}

func TestDeploymentRouteKeyRejectsEmptyLabels(t *testing.T) {
	if _, err := DeploymentRouteKey("!!!", "default"); err == nil {
		t.Fatal("expected invalid deployment label error")
	}
}
