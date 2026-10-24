// Package routing contains the public-route identity rules shared by the
// control plane and ingress. Keeping this outside an HTTP handler prevents the
// API and ingress from silently deriving different hostnames.
package routing

import (
	"fmt"
	"regexp"
	"strings"
)

var invalidLabelChar = regexp.MustCompile(`[^a-z0-9-]+`)
var repeatedDash = regexp.MustCompile(`-+`)

// SlugLabel returns a DNS-safe lower-case label. Route labels are intentionally
// limited to 63 bytes, the DNS label limit.
func SlugLabel(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = invalidLabelChar.ReplaceAllString(slug, "-")
	slug = repeatedDash.ReplaceAllString(slug, "-")
	return strings.Trim(slug, "-")
}

// DeploymentRouteKey returns the public hostname without the configured base
// domain, for example "llama-chat.default".
func DeploymentRouteKey(deploymentName, organizationName string) (string, error) {
	deployment := SlugLabel(deploymentName)
	organization := SlugLabel(organizationName)
	if deployment == "" || organization == "" {
		return "", fmt.Errorf("deployment and organization names must contain a DNS-safe character")
	}
	if len(deployment) > 63 || len(organization) > 63 {
		return "", fmt.Errorf("deployment and organization route labels must be at most 63 characters")
	}
	return deployment + "." + organization, nil
}
