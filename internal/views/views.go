// Views include core UI components common to iblcli
package views

import (
	"fmt"
	"strconv"

	"github.com/InfinityBotList/ibl/internal/api"
	"github.com/InfinityBotList/ibl/internal/config"
	"github.com/InfinityBotList/ibl/internal/input"
	"github.com/InfinityBotList/ibl/internal/login"
	"github.com/InfinityBotList/ibl/internal/ui"
	"github.com/InfinityBotList/ibl/types"
	"github.com/InfinityBotList/ibl/types/popltypes"
	"github.com/infinitybotlist/eureka/dovewing"
	"github.com/pkg/errors"
)

func UserEntitySelector(auth popltypes.TestAuth, filter func(e types.Entity) bool) (*types.Entity, error) {
	if auth.AuthType != types.TargetTypeUser {
		return nil, errors.New("not logged in as a user")
	}

	res, err := api.NewReq().Get("users/" + auth.TargetID).Do()

	if err != nil {
		return nil, errors.Wrap(err, "error getting user")
	}

	var user popltypes.User

	err = res.JsonOk(&user)

	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch user")
	}

	// Create a set of entities
	var rawEntities []types.Entity

	// #1 - Bots
	for _, bot := range user.UserBots {
		rawEntities = append(rawEntities, types.Entity{
			ID:         bot.User.ID,
			Name:       bot.User.Username,
			TargetType: types.TargetTypeBot,
		})
	}

	// #2 - Teams (not all apis support teams yet!)
	for _, team := range user.UserTeams {
		rawEntities = append(rawEntities, types.Entity{
			ID:         team.ID,
			Name:       team.Name,
			TargetType: types.TargetTypeTeam,
		})
	}

	// Filter the entities
	var entities []types.Entity

	for _, entity := range rawEntities {
		if filter(entity) {
			entities = append(entities, entity)
		}
	}

	// Next ask the user to select a entity
	for i, entity := range entities {
		fmt.Printf("%d. %s [%s] (%s)\n", i+1, entity.Name, entity.ID, entity.TargetType)
	}

	// Get the user's selection
	selected := input.GetInput("Choose an option", func(s string) bool {
		// Convert choice to string
		choice, err := strconv.Atoi(s)

		if err != nil {
			fmt.Print(ui.RedText("Invalid choice, please try again:" + err.Error()))
			return false
		}

		// Check if choice is in range
		if choice < 1 || choice > len(entities) {
			fmt.Print(ui.RedText("Choice out of range, please try again"))
			return false
		}

		return true
	})

	// Convert choice to int
	choice, err := strconv.Atoi(selected)

	if err != nil {
		return nil, errors.Wrap(err, "error converting choice to int")
	}

	// Return the selected entity
	return &entities[choice-1], nil
}

func AccountSwitcher(authType string) (*popltypes.TestAuth, error) {
	forceRetryLogin := false
	for {
		var a *popltypes.TestAuth
		err := config.LoadConfig("auth@"+authType, &a)

		if err != nil || a == nil || a.TargetID == "" || forceRetryLogin {
			if !forceRetryLogin {
				fmt.Print(ui.RedText("You are not logged in on IBL CLI yet (as a " + authType + ")! In order to continue, you must login here..."))
			}

			auth, err := login.LoginUser(authType)

			if err != nil {
				return nil, errors.Wrap(err, "error logging in")
			}

			a = auth
		}

		res, err := api.NewReq().Get("_duser/" + a.TargetID).Do()

		if err != nil {
			return nil, errors.Wrap(err, "error getting user")
		}

		var user dovewing.DiscordUser

		err = res.JsonOk(&user)

		if err != nil {
			return nil, errors.Wrap(err, "error getting username")
		}

		confirm := input.GetInput(fmt.Sprint("You're logged in as ", ui.BoldTextNoLn(user.Username), " ["+authType+"]. ", "Continue [y/n]"), func(s string) bool {
			return s == "y" || s == "n"
		})

		if confirm == "n" {
			forceRetryLogin = true
			continue
		}

		if a == nil {
			return nil, errors.Wrap(err, "unknown login error, auth is nil")

		}

		fmt.Print(ui.GreenText("Excellent! You're logged in!"))
		return a, nil
	}
}
