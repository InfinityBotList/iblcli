package types

type TargetType = string

const (
	TargetTypeUser   TargetType = "user"
	TargetTypeBot    TargetType = "bot"
	TargetTypeServer TargetType = "server"

	// Cannot be logged into, but still a target type
	TargetTypeTeam TargetType = "team"
)

// Internal struct defining a iblcli entity
type Entity struct {
	TargetType TargetType
	ID         string
	Name       string
}
