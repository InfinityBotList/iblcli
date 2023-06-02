package types

import "github.com/infinitybotlist/eureka/dovewing"

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
	Domain  string          `json:"domain"`
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

// funnel entities
type FunnelBot struct {
	User       *dovewing.DiscordUser `json:"user"`
	BotID      string                `json:"bot_id"`
	WebhooksV2 bool                  `json:"webhooks_v2"`
	Owner      *dovewing.DiscordUser `json:"owner"`
	TeamOwner  *struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Members []struct {
			User  *dovewing.DiscordUser `json:"user"`
			Perms []TeamPermission      `json:"perms"`
		} `json:"members"`
	} `json:"team_owner"`
}

type FunnelServer struct {
	ServerID   string `json:"bot_id"`
	WebhooksV2 bool   `json:"webhooks_v2"`
	TeamOwner  *struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Members []struct {
			User  *dovewing.DiscordUser `json:"user"`
			Perms []TeamPermission      `json:"perms"`
		} `json:"members"`
	} `json:"team_owner"`
}

// webhooks.go

type PatchBotWebhook struct {
	WebhookURL    string `json:"webhook_url"`
	WebhookSecret string `json:"webhook_secret"`
	WebhooksV2    *bool  `json:"webhooks_v2"`
	Clear         bool   `json:"clear"`
}
