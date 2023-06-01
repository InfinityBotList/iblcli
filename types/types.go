package types

type TestAuth struct {
	AuthType TargetType `json:"auth_type"`
	TargetID string     `json:"target_id"`
	Token    string     `json:"token"`
}

type TargetType string

const (
	TargetTypeUser   TargetType = "user"
	TargetTypeBot    TargetType = "bot"
	TargetTypeServer TargetType = "server"
)

type WebhookFunnel struct {
	TargetType    TargetType `json:"target_type"`
	TargetID      string     `json:"target_id"`
	WebhookSecret string     `json:"webhook_secret"`
	EndpointID    string     `json:"endpoint_id"`
	Forward       string     `json:"forward"`
}

type FunnelList struct {
	Port    int             `json:"port"`
	Funnels []WebhookFunnel `json:"funnels"`
}

// Auth data
type AuthData struct {
	TargetType TargetType `json:"target_type"`
	ID         string     `json:"id"`
	Authorized bool       `json:"authorized"`
}

// oauth2

type OauthMeta struct {
	ClientID string `json:"client_id"`
	URL      string `json:"url"`
}

type AuthorizeRequest struct {
	ClientID    string `json:"client_id" validate:"required"`
	Code        string `json:"code" validate:"required,min=5"`
	RedirectURI string `json:"redirect_uri" validate:"required"`
	Nonce       string `json:"nonce" validate:"required"` // Just to identify and block older clients from vulns
	Scope       string `json:"scope" validate:"required,oneof=normal ban_exempt external_auth"`
}

type UserLogin struct {
	Token  string `json:"token" description:"The users token"`
	UserID string `json:"user_id" description:"The users ID"`
}

type DevMode string

const (
	DevModeFull  DevMode = "full"
	DevModeLocal DevMode = "local"
	DevModeOff   DevMode = "off"
)

type DevModeCfg struct {
	Mode DevMode `json:"mode"`
}
