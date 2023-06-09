package types

import "strings"

type WebhookFunnel struct {
	TargetType TargetType `yaml:"target_type"`
	TargetID   string     `yaml:"target_id"`
	EndpointID string     `yaml:"endpoint_id"`
	Forward    string     `yaml:"forward"`

	// Hidden from String(), sensitive
	WebhookSecret string `yaml:"webhook_secret"`
}

func (w WebhookFunnel) String() string {
	entries := []string{
		"Target Type: " + string(w.TargetType),
		"Target ID: " + w.TargetID,
		"Endpoint ID: " + w.EndpointID,
		"Forward: " + w.Forward,
	}

	return strings.Join(entries, "\n")
}

type FunnelList struct {
	Port    int             `yaml:"port"`
	Domain  string          `yaml:"domain"`
	Funnels []WebhookFunnel `yaml:"funnels"`
}
