package types

type WebhookFunnel struct {
	TargetType    TargetType `yaml:"target_type"`
	TargetID      string     `yaml:"target_id"`
	WebhookSecret string     `yaml:"webhook_secret"`
	EndpointID    string     `yaml:"endpoint_id"`
	Forward       string     `yaml:"forward"`
}

type FunnelList struct {
	Port    int             `yaml:"port"`
	Domain  string          `yaml:"domain"`
	Funnels []WebhookFunnel `yaml:"funnels"`
}
