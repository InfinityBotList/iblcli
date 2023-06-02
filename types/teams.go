package types

type TeamPermission string // TeamPermission is a permission that a team can have

/*
TODO:

- Add more permissions
- Arcadia task to ensure all owners have a team_member entry with the OWNER permission
*/

const (
	TeamPermissionUndefined TeamPermission = ""

	// Bot permissions
	TeamPermissionEditBotSettings      TeamPermission = "EDIT_BOT_SETTINGS"
	TeamPermissionAddNewBots           TeamPermission = "ADD_NEW_BOTS"
	TeamPermissionResubmitBots         TeamPermission = "RESUBMIT_BOTS"
	TeamPermissionCertifyBots          TeamPermission = "CERTIFY_BOTS"
	TeamPermissionResetBotTokens       TeamPermission = "RESET_BOT_TOKEN"
	TeamPermissionEditBotWebhooks      TeamPermission = "EDIT_BOT_WEBHOOKS"
	TeamPermissionTestBotWebhooks      TeamPermission = "TEST_BOT_WEBHOOKS"
	TeamPermissionGetBotWebhookLogs    TeamPermission = "GET_BOT_WEBHOOK_LOGS"
	TeamPermissionRetryBotWebhookLogs  TeamPermission = "RETRY_BOT_WEBHOOK_LOGS"
	TeamPermissionDeleteBotWebhookLogs TeamPermission = "DELETE_BOT_WEBHOOK_LOGS"
	TeamPermissionSetBotVanity         TeamPermission = "SET_BOT_VANITY"
	TeamPermissionDeleteBots           TeamPermission = "DELETE_BOTS"

	// Server permissions
	TeamPermissionEditServerSettings      TeamPermission = "EDIT_SERVER_SETTINGS"
	TeamPermissionAddNewServers           TeamPermission = "ADD_NEW_SERVERS"
	TeamPermissionCertifyServers          TeamPermission = "CERTIFY_SERVERS"
	TeamPermissionResetServerTokens       TeamPermission = "RESET_SERVER_TOKEN"
	TeamPermissionEditServerWebhooks      TeamPermission = "EDIT_SERVER_WEBHOOKS"
	TeamPermissionTestServerWebhooks      TeamPermission = "TEST_SERVER_WEBHOOKS"
	TeamPermissionGetServerWebhookLogs    TeamPermission = "GET_SERVER_WEBHOOK_LOGS"
	TeamPermissionRetryServerWebhookLogs  TeamPermission = "RETRY_SERVER_WEBHOOK_LOGS"
	TeamPermissionDeleteServerWebhookLogs TeamPermission = "DELETE_SERVER_WEBHOOK_LOGS"
	TeamPermissionSetServerVanity         TeamPermission = "SET_SERVER_VANITY"
	TeamPermissionDeleteServers           TeamPermission = "DELETE_SERVERS"

	// Common permissions
	TeamPermissionEditTeamInfo              TeamPermission = "EDIT_TEAM_INFO"
	TeamPermissionAddTeamMembers            TeamPermission = "ADD_TEAM_MEMBERS"
	TeamPermissionRemoveTeamMembers         TeamPermission = "REMOVE_TEAM_MEMBERS"
	TeamPermissionEditTeamMemberPermissions TeamPermission = "EDIT_TEAM_MEMBER_PERMISSIONS"

	// Owner permission
	TeamPermissionOwner TeamPermission = "OWNER"
)
