package types

type TestAuth struct {
	AuthType TargetType `json:"auth_type"`
	TargetID string     `json:"target_id"`
	Token    string     `json:"token"`
}

type TargetType int

const (
	TargetTypeUser TargetType = iota
	TargetTypeBot
	TargetTypeServer
)

type NotifyMethod int

const (
	NotifyMethodWebhook NotifyMethod = iota
	NotifyMethodHTTP
)

type AuthData struct {
	TargetType TargetType `json:"target_type"`
	ID         string     `json:"id"`
	Authorized bool       `json:"authorized"`
}

type WebhookSecret struct {
	Secret string `json:"secret"`
}

type WebhookState struct {
	HTTP        bool `json:"http"`
	WebhookHMAC bool `json:"webhook_hmac_auth"`
	SecretSet   bool `json:"webhook_secret_set"`
}
