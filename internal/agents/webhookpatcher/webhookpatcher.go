package webhookpatcher

import (
	"errors"
	"fmt"

	"github.com/InfinityBotList/ibl/internal/api"
	"github.com/InfinityBotList/ibl/internal/ui"
	"github.com/InfinityBotList/ibl/types"
	"github.com/InfinityBotList/ibl/types/popltypes"
)

func PatchWebhooks(funnels *types.FunnelList, u popltypes.TestAuth) error {
	for _, funnel := range funnels.Funnels {
		fmt.Print(ui.BlueText("Updating webhook configuration for", funnel.TargetType, funnel.TargetID))

		url := funnels.Domain + "/funnel?id=" + funnel.EndpointID

		fmt.Print(ui.BlueText("Domain: " + url))

		tBool := true

		switch funnel.TargetType {
		case types.TargetTypeBot:
			pw := popltypes.PatchBotWebhook{
				WebhookURL:    url,
				WebhookSecret: funnel.WebhookSecret,
				WebhooksV2:    &tBool,
			}

			// /users/{uid}/bots/{bid}/webhook
			resp, err := api.NewReq().Patch("users/" + u.TargetID + "/bots/" + funnel.TargetID + "/webhook").Auth(u.Token).Json(pw).Do()

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
		case types.TargetTypeTeam:
			pw := popltypes.PatchTeamWebhook{
				WebhookURL:    url,
				WebhookSecret: funnel.WebhookSecret,
			}

			// /users/{uid}/teams/{tid}/webhook
			resp, err := api.NewReq().Patch("users/" + u.TargetID + "/teams/" + funnel.TargetID + "/webhook").Auth(u.Token).Json(pw).Do()

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
		default:
			return errors.New("target type does not support webhook funnels")
		}
	}

	return nil
}
