package funneleditor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/InfinityBotList/ibl/internal/api"
	"github.com/InfinityBotList/ibl/internal/config"
	"github.com/InfinityBotList/ibl/internal/input"
	"github.com/InfinityBotList/ibl/internal/ui"
	"github.com/InfinityBotList/ibl/internal/views"
	"github.com/InfinityBotList/ibl/types"
	"github.com/InfinityBotList/ibl/types/popltypes"
	"github.com/infinitybotlist/eureka/crypto"
)

type FunnelCommand = func(popltypes.TestAuth, *types.FunnelList) error

type funnelAction struct {
	Char   string
	Name   string
	Action FunnelCommand
}

var funnelActions = []funnelAction{
	{
		Char:   "P",
		Name:   "Change webserver port",
		Action: portMan,
	},
	{
		Char:   "D",
		Name:   "Set domain",
		Action: setDomain,
	},
	{
		Char:   "N",
		Name:   "New funnel",
		Action: newFunnel,
	},
	{
		Char:   "R",
		Name:   "Remove funnel",
		Action: deleteFunnel,
	},
	{
		Char:   "E",
		Name:   "Open funnel editor",
		Action: editor,
	},
	{
		Char:   "Q",
		Name:   "Save And Quit",
		Action: saveAndQuit,
	},
}

func ManageConsole(user popltypes.TestAuth, funnels types.FunnelList) {
	for {
		fmt.Println(`
Welcome to IBL Funnels!

Funnels are a way to ingest webhooks v2 data from the web (and its *scary* out there!)
to a service hosted locally on your machine, *hopefully* firewalled and bound to 127.0.0.1 
only!!!

To start out, set a port, the domain at which the *funnel* will be hosted, then start creating 
funnels to your services!
					`)

		fmt.Println("")

		// Print current settings
		fmt.Println("Current settings:")
		fmt.Println("Port:", funnels.Port)
		fmt.Println("Domain:", funnels.Domain)
		fmt.Println("Funnels:")

		for i, funnel := range funnels.Funnels {
			fmt.Print(ui.BoldText("Funnel", i+1))
			fmt.Println(funnel.String())
			fmt.Println("")
		}

		fmt.Println("")
		fmt.Println("")

		for _, action := range funnelActions {
			fmt.Println(action.Char, "-", action.Name)
		}

		fmt.Println("")

		keyInput := input.GetInput("Select an option?", func(s string) bool {
			for _, action := range funnelActions {
				if action.Char == s {
					return true
				}
			}

			fmt.Print(ui.RedText("Invalid option"))
			return false
		})

		var flag = false
		for _, action := range funnelActions {
			if action.Char == keyInput {
				flag = true
				err := action.Action(user, &funnels)

				if err != nil {
					fmt.Print(ui.RedText("Error:", err))
					time.Sleep(5 * time.Second)
				}
			}
		}

		if !flag {
			fmt.Print(ui.RedText("Invalid option"))
			continue
		}
	}
}

func portMan(_ popltypes.TestAuth, funnels *types.FunnelList) error {
	port := input.GetInput("What port should the webserver run on?", func(s string) bool {
		// Check if port is a number
		_, err := strconv.Atoi(s)

		if err != nil {
			fmt.Fprint(os.Stderr, ui.RedText("Invalid port number"))
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

func setDomain(_ popltypes.TestAuth, funnels *types.FunnelList) error {
	domain := input.GetInput("What domain/IP will the webhook be accessible from?", func(s string) bool {
		if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
			return true
		}

		fmt.Print(ui.RedText("Invalid domain. Must start with http:// or https://"))
		return false
	})

	funnels.Domain = domain

	return nil
}

func newFunnel(u popltypes.TestAuth, funnels *types.FunnelList) error {
	if funnels.Port == 0 || funnels.Domain == "" {
		return errors.New("please set a port and webhook domain ('P' and 'D') before adding a funnel")
	}

	entity, err := views.UserEntitySelector(u, []types.TargetType{
		types.TargetTypeBot,
		// TODO: Add teams, once supported, here
	})

	if err != nil {
		return err
	}

	forwardTo := input.GetInput("Forward to?", func(s string) bool {
		if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
			return true
		}

		fmt.Print(ui.RedText("Invalid domain. Must start with http:// or https://"))
		return false
	})

	// Fetch entity
	switch entity.TargetType {
	case types.TargetTypeBot:
		fmt.Print("Adding ", ui.BoldText(entity.Name+" ["+entity.ID+"]"))
		fmt.Print(ui.BlueText("Updating webhook configuration for this bot..."))

		endpointId := crypto.RandString(32)
		webhookSecret := crypto.RandString(128)

		fmt.Print(ui.BlueText("Domain: " + funnels.Domain + "/funnel?id=" + endpointId))

		tBool := true

		pw := popltypes.PatchBotWebhook{
			WebhookURL:    funnels.Domain + "/?id=" + endpointId,
			WebhookSecret: webhookSecret,
			WebhooksV2:    &tBool,
		}

		// /users/{uid}/bots/{bid}/webhook
		resp, err := api.NewReq().Patch("users/" + u.TargetID + "/bots/" + entity.ID + "/webhook").Auth(u.Token).Json(pw).Do()

		if err != nil {
			return errors.New("error occurred while updating webhook: " + err.Error())
		}

		if resp.Response.StatusCode != 204 {
			body, err := resp.Body()

			if err != nil {
				return errors.New("error occurred while parsing error when updating webhook: " + err.Error())
			}

			return errors.New("error occurred while updating webhook: " + string(body))
		}

		// Add to funnels
		funnels.Funnels = append(funnels.Funnels, types.WebhookFunnel{
			TargetType:    entity.TargetType,
			TargetID:      entity.ID,
			WebhookSecret: webhookSecret,
			EndpointID:    endpointId,
			Forward:       forwardTo,
		})

		return nil
	default:
		return errors.New("target type does not support webhook funnels")
	}
}

func deleteFunnel(_ popltypes.TestAuth, funnels *types.FunnelList) error {
	for i, funnel := range funnels.Funnels {
		fmt.Print(ui.BoldText("Funnel", i+1))
		fmt.Println(funnel.String())
		fmt.Println("")
	}

	index := input.GetInput("Which funnel would you like to delete?", func(s string) bool {
		choice, err := strconv.Atoi(s)

		if err != nil {
			fmt.Print(ui.RedText("Invalid option"))
			return false
		}

		// Check if choice is in range
		if choice < 1 || choice > len(funnels.Funnels) {
			fmt.Print(ui.RedText("Choice out of range, please try again"))
			return false
		}

		return true
	})

	choice, err := strconv.Atoi(index)

	if err != nil {
		return errors.New("invalid choice")
	}

	// Remove from funnels
	funnels.Funnels = append(funnels.Funnels[:choice-1], funnels.Funnels[choice:]...)

	return nil
}

func editor(_ popltypes.TestAuth, funnels *types.FunnelList) error {
	err := config.WriteConfig("funnels", funnels)

	if err != nil {
		fmt.Print(ui.RedText("Config save error: " + err.Error()))
		time.Sleep(3 * time.Second)
	}

	cfgFile := config.ConfigFile()

	// Get editor to use using EDITOR env var defaulting to nano if unset
	editor := os.Getenv("EDITOR")

	if editor == "" {
		editor = "nano"
	}

	// Open editor
	cmd := exec.Command(editor, cfgFile+"/funnels")

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()

	if err != nil {
		return errors.New("error opening editor: " + err.Error())
	}

	// Reload config
	err = config.LoadConfig("funnels", funnels)

	if err != nil {
		return errors.New("error reloading config: " + err.Error())
	}

	return nil
}

func saveAndQuit(_ popltypes.TestAuth, funnels *types.FunnelList) error {
	err := config.WriteConfig("funnels", funnels)

	if err != nil {
		fmt.Print(ui.RedText("Config save error: " + err.Error()))
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}

	os.Exit(0)
	return nil
}
