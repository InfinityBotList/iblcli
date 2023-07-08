package funnelserver

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/InfinityBotList/ibl/internal/agents/webhookpatcher"
	"github.com/InfinityBotList/ibl/internal/ui"
	"github.com/InfinityBotList/ibl/types"
	"github.com/InfinityBotList/ibl/types/popltypes"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type secret struct {
	Raw string
}

func (s secret) Sign(data []byte) string {
	h := hmac.New(sha512.New, []byte(s.Raw))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

const webhookProtocol = "splashtail"

type funnelCache struct {
	action string // port or exec
	data   string // port number or exec command
}

var fc = make(map[string]funnelCache)

func StartServer(funnels *types.FunnelList, u popltypes.TestAuth) {
	err := webhookpatcher.PatchWebhooks(funnels, u)

	if err != nil {
		fmt.Print(ui.RedText("Error occurred while patching webhooks: " + err.Error()))
	}

	for _, funnel := range funnels.Funnels {
		forwardSplit := strings.Split(funnel.Forward, ":")

		if len(forwardSplit) < 2 {
			fmt.Println(ui.RedText("Invalid forward: " + funnel.Forward))
			os.Exit(1)
		}

		action := forwardSplit[0]
		data := strings.Join(forwardSplit[1:], ":")

		if action != "port" && action != "exec" {
			fmt.Println(ui.RedText("Invalid forward: " + funnel.Forward))
			os.Exit(1)
		}

		fc[funnel.EndpointID] = funnelCache{
			action: action,
			data:   data,
		}
	}

	mux := chi.NewMux()

	mux.Use(
		middleware.Logger,
		middleware.Recoverer,
		middleware.RealIP,
	)

	mux.Post("/funnel", func(w http.ResponseWriter, r *http.Request) {
		// Get 'id'
		id := r.URL.Query().Get("id")

		// Get funnel
		var funnel *types.WebhookFunnel

		for _, f := range funnels.Funnels {
			if f.EndpointID == id {
				funnel = &f
				break
			}
		}

		if funnel == nil {
			w.WriteHeader(http.StatusGone)
			w.Write([]byte("Funnel not found"))
			return
		}

		// Check x-webhook-protocol (Step #1) of https://spider.infinitybots.gg/docs#overview--parsing-webhooks
		if r.Header.Get("x-webhook-protocol") != webhookProtocol {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Invalid protocol version!"))
			return
		}

		// Ensure nonce exists
		nonce := r.Header.Get("x-webhook-nonce")

		if nonce == "" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("No nonce provided?"))
			return
		}

		sig := r.Header.Get("x-webhook-signature")

		if sig == "" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("No signature provided?"))
			return
		}

		// Check that body is not empty
		if r.Body == nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("No body provided?"))
			return
		}

		// Read body into bytes
		body, err := io.ReadAll(r.Body)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Could not read body"))
			return
		}

		tok1 := secret{Raw: funnel.WebhookSecret}.Sign(body)
		finalToken := secret{Raw: nonce}.Sign([]byte(tok1))

		if sig != finalToken {
			fmt.Println("expected:", finalToken, "| got:", sig)
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Invalid signature"))
			return
		}

		hkS := sha256.New()
		hkS.Write([]byte(funnel.WebhookSecret + nonce))

		// Decrypt request body with hashed
		c, err := aes.NewCipher(hkS.Sum(nil))

		if err != nil {
			fmt.Println(err, "cipher creation failed")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Could not create cipher"))
			return
		}

		gcm, err := cipher.NewGCM(c)

		if err != nil {
			fmt.Println(err, "gcm creation failed")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Could not create cipher gcm"))
			return
		}

		nonceSize := gcm.NonceSize()

		// Hex decode body
		body, err = hex.DecodeString(string(body))

		if err != nil {
			fmt.Println(err, "hex decode failed")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Could not decode body"))
			return
		}

		if len(body) < nonceSize {
			fmt.Println("body too small for nonce size")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Body too small"))
			return
		}

		nonceData, cipherText := body[:nonceSize], body[nonceSize:]
		plaintext, err := gcm.Open(nil, nonceData, cipherText, nil)

		if err != nil {
			fmt.Println(err, "gcm open failed")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Could not open gcm"))
			return
		}

		// Check funnel forward
		fcData, ok := fc[funnel.EndpointID]

		if !ok {
			fmt.Println(ui.RedText("Funnel forward not found"))
			w.WriteHeader(http.StatusInsufficientStorage)
			w.Write([]byte("Funnel forward not found"))
		}

		switch {
		case fcData.action == "port":
			// Timeout here must be 10 seconds
			client := http.Client{
				Timeout: 10 * time.Second,
			}

			resp, err := client.Post("http://localhost:"+fcData.data, "application/json", bytes.NewReader(plaintext))

			if err != nil {
				fmt.Println(ui.RedText("Funnel forward failed: " + err.Error()))
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			switch {
			case resp.StatusCode >= 200 && resp.StatusCode < 300:
				w.WriteHeader(http.StatusNoContent)
				return
			case resp.StatusCode >= 300 && resp.StatusCode < 400:
				fmt.Print(ui.RedText("Unexpected redirction:", resp.StatusCode, funnel.EndpointID))
				w.WriteHeader(http.StatusBadGateway)
				return
			case resp.StatusCode >= 400 && resp.StatusCode < 500:
				w.WriteHeader(http.StatusBadRequest)

				// Try to read body?
				body, err := io.ReadAll(resp.Body)

				if err != nil {
					fmt.Print(ui.RedText("Client response (unknown body):", resp.StatusCode, funnel.EndpointID))
					w.Write([]byte("Bad request"))
					return
				}

				fmt.Print(ui.RedText("Client response:", resp.StatusCode, funnel.EndpointID, "\nBody:", string(body)))

				w.Write(body)
				return
			case resp.StatusCode >= 500 && resp.StatusCode < 600:
				w.WriteHeader(http.StatusInternalServerError)

				// Try to read body?
				body, err := io.ReadAll(resp.Body)

				if err != nil {
					fmt.Print(ui.RedText("Client response (unknown body):", resp.StatusCode, funnel.EndpointID))
					w.Write([]byte("Internal server error"))
					return
				}

				fmt.Print(ui.RedText("Client response:", resp.StatusCode, funnel.EndpointID, "\nBody:", string(body)))

				w.Write(body)
				return
			}

			fmt.Println(ui.RedText("Unhandled status code:", resp.StatusCode, funnel.EndpointID))
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Unhandled status code: " + strconv.Itoa(resp.StatusCode)))
			return
		case fcData.action == "exec":
			// Timeout here must be 10 seconds
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, fcData.data, "")

			cmd.Env = append(os.Environ(), "DATA="+hex.EncodeToString(plaintext))

			out, err := cmd.CombinedOutput()

			if ctx.Err() != context.DeadlineExceeded {
				// Command completed before timeout
				if err != nil {
					fmt.Println(ui.RedText("Funnel forward failed:", err.Error()))
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
					return
				}

				fmt.Println(string(out))
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Command timed out
			fmt.Println(ui.RedText("Funnel forward timed out:", err.Error()))
			w.WriteHeader(http.StatusGatewayTimeout)
			w.Write([]byte(err.Error() + "\n" + string(out)))
			return
		}
	})

	fmt.Println("Starting funnel server on port", funnels.Port)

	http.ListenAndServe(":"+strconv.Itoa(funnels.Port), mux)
}
