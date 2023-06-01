package funnel

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/InfinityBotList/ibl/helpers"
	"github.com/InfinityBotList/ibl/types"
)

type FunnelCommand = func(*types.FunnelList) error

type funnelAction struct {
	Name   string
	Action FunnelCommand
}

var funnelActions = map[string]funnelAction{
	"P": {
		Name:   "Change webserver port",
		Action: portMan,
	},
	"N": {
		Name:   "New funnel",
		Action: newFunnel,
	},
	"Q": {
		Name: "Quit",
		Action: func(funnels *types.FunnelList) error {
			os.Exit(0)
			return nil
		},
	},
}

func ManageConsole(user types.TestAuth, funnels types.FunnelList) {
	for {
		fmt.Println("")
		fmt.Println("")

		for key, action := range funnelActions {
			fmt.Println(key, "-", action.Name)
		}

		fmt.Println("")

		keyInput := helpers.GetInput("Select an option?", func(s string) bool {
			_, ok := funnelActions[s]

			if !ok {
				fmt.Print(helpers.RedText("Invalid option"))
				return false
			}

			return true
		})

		action, ok := funnelActions[keyInput]

		if !ok {
			fmt.Print(helpers.RedText("Invalid option"))
			continue
		}

		err := action.Action(&funnels)

		if err != nil {
			fmt.Print(helpers.RedText("Invalid option"))
			time.Sleep(5 * time.Second)
		}

		err = helpers.WriteConfig("funnels", funnels)

		if err != nil {
			fmt.Print(helpers.RedText("Config save error: " + err.Error()))
			time.Sleep(5 * time.Second)
		}
	}
}

func portMan(funnels *types.FunnelList) error {
	port := helpers.GetInput("What port should the webserver run on?", func(s string) bool {
		// Check if port is a number
		_, err := strconv.Atoi(s)

		if err != nil {
			fmt.Fprint(os.Stderr, helpers.RedText("Invalid port number"))
			return false
		}

		return true
	})

	portNum, err := strconv.Atoi(port)

	if err != nil {
		return errors.New("invalid port number")
	}

	funnels.Port = portNum

	return nil
}

func newFunnel(funnels *types.FunnelList) error {
	return nil
}
