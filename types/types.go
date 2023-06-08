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

// Auth data
type AuthData struct {
	TargetType TargetType `json:"target_type" yaml:"target_type"`
	ID         string     `json:"id" yaml:"id"`
	Authorized bool       `json:"authorized" yaml:"authorized"`
}
