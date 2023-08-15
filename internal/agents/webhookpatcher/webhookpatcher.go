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

		pw := popltypes.PatchWebhook{
			WebhookURL:    url,
			WebhookSecret: funnel.WebhookSecret,
		}

		// /users/{uid}/bots/{bid}/webhook
		resp, err := api.NewReq().Patch("users/" + u.TargetID + "/webhooks/" + funnel.TargetID + "?target_type=" + funnel.TargetType).Auth(u.Token).Json(pw).Do()

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
	}

	return nil
}
