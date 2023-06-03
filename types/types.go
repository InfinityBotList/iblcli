package types

type TestAuth struct {
	AuthType TargetType `yaml:"auth_type" json:"auth_type"`
	TargetID string     `yaml:"target_id" json:"target_id"`
	Token    string     `yaml:"token" json:"token"`
}

type TargetType string

const (
	TargetTypeUser   TargetType = "user"
	TargetTypeBot    TargetType = "bot"
	TargetTypeServer TargetType = "server"
)

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

// Auth data
type AuthData struct {
	TargetType TargetType `json:"target_type" yaml:"target_type"`
	ID         string     `json:"id" yaml:"id"`
	Authorized bool       `json:"authorized" yaml:"authorized"`
}
