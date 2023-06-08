package views

import (
	"fmt"
	"os"

	"github.com/InfinityBotList/ibl/internal/api"
	"github.com/InfinityBotList/ibl/internal/config"
	"github.com/InfinityBotList/ibl/internal/input"
	"github.com/InfinityBotList/ibl/internal/login"
	"github.com/InfinityBotList/ibl/internal/ui"
	"github.com/InfinityBotList/ibl/types"
	"github.com/infinitybotlist/eureka/dovewing"
)

func UserBotSelector() *dovewing.DiscordUser {
	return nil
}

func AccountSwitcher() types.TestAuth {
	var auth types.TestAuth

	var flag bool = true
	for flag {
		var a *types.TestAuth
		err := config.LoadConfig("auth@user", &a)

		if err != nil {
			fmt.Print(ui.RedText("You are not logged in on IBL CLI yet! A login is required for proper configuration and setup..."))
			os.Setenv("REQUIRED_AUTH_METHOD", "user")
			login.LoginUser()
		} else {
			res, err := api.NewReq().Get("_duser/" + a.TargetID).Do()

			if err != nil {
				fmt.Print(ui.RedText("Error getting user: " + err.Error() + ", exiting..."))
				os.Exit(1)
			}

			var user dovewing.DiscordUser

			err = res.JsonOk(&user)

			if err != nil {
				fmt.Print(ui.RedText("Error getting username: " + err.Error() + ", reauthenticating..."))
				os.Setenv("REQUIRED_AUTH_METHOD", "user")
				login.LoginUser()
			}

			confirm := input.GetInput(fmt.Sprint("You're logged in as ", ui.BoldText(user.Username), "Continue [y/n]"), func(s string) bool {
				return s == "y" || s == "n"
			})

			if confirm == "n" {
				os.Setenv("REQUIRED_AUTH_METHOD", "user")
				login.LoginUser()
			}
		}

		fmt.Print(ui.GreenText("Excellent! You're logged in!"))
		flag = false

		auth = *a
	}

	return auth
}
