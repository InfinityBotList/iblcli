/*
Copyright © 2023 Infinity Bot List
*/
package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/InfinityBotList/ibl/internal/devmode"
	"github.com/InfinityBotList/ibl/types"
	"github.com/spf13/cobra"
)

// proxyCmd represents the proxy command
var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Starts up nirn-proxy",
	Long:  `Self-explanatory.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check for nirn-proxy in the PATH
		path, err := exec.LookPath("nirn-proxy")

		if err != nil {
			fmt.Println("Error looking for nirn-proxy:", err)
			os.Exit(1)
		}

		if path == "" {
			fmt.Println("nirn-proxy not found in PATH. Try running `go install github.com/germanoeich/nirn-proxy@latest`")
			os.Exit(1)
		}

		if os.Getenv("PORT") == "" {
			os.Setenv("PORT", "3219")
		}

		oip := cmd.Flag("outbound-ip").Value.String()

		if oip != "" {
			os.Setenv("OUTBOUND_IP", oip)
		}

		if os.Getenv("OUTBOUND_IP") == "" {
			// Get the anchor URL from DO
			resp, err := http.Get("http://169.254.169.254/metadata/v1/interfaces/public/0/anchor_ipv4/address")

			if err != nil {
				fmt.Println("Error getting anchor URL:", err)
				os.Exit(1)
			}

			defer resp.Body.Close()

			url, err := io.ReadAll(resp.Body)

			if err != nil {
				fmt.Println("Error reading anchor URL:", err)
				os.Exit(1)
			}

			fmt.Println("Anchor URL:", string(url))

			os.Setenv("OUTBOUND_IP", string(url))
		}

		// Fork nirn-proxy
		proxy := exec.Command(path)

		proxy.Stdout = os.Stdout
		proxy.Stderr = os.Stderr

		err = proxy.Run()

		if err != nil {
			fmt.Println("Error running nirn-proxy:", err)
			os.Exit(1)
		}
	},
}

func init() {
	if devmode.DevMode().Allows(types.DevModeFull) {
		rootCmd.AddCommand(proxyCmd)

		proxyCmd.Flags().StringP("outbound-ip", "o", "", "The outbound IP to use")
	}
}
