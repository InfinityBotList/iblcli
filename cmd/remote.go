/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/InfinityBotList/ibl/helpers"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func devOnlyWarn() {
	fmt.Println("This command is only meant for developers and will NOT work for normal users.")
	time.Sleep(1 * time.Second)
}

// remoteCmd represents the remote command
var remoteCmd = &cobra.Command{
	Use:     "remote",
	Aliases: []string{"remotes", "r", "server"},
	Short:   "Remote Management",
	Long:    ``,
}

// setRemoteCmd represents the set command
var setRemoteCmd = &cobra.Command{
	Use:     "set name hostname",
	Aliases: []string{"s", "add"},
	Short:   "Set a remote server",
	Long:    ``,
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		keyFlag := cmd.Flag("key").Value.String()
		usernameFlag := cmd.Flag("username").Value.String()
		passFlag := cmd.Flag("pass").Value.String()

		devOnlyWarn()

		var pass string

		if passFlag == "true" {
			fmt.Println("Enter the password for the private key:")
			passwd, err := term.ReadPassword(int(syscall.Stdin))

			if err != nil {
				fmt.Println("Error reading password:", err)
				return
			}

			pass = string(passwd)
		}

		configPath, err := helpers.GetConfigDirAndCreate()

		if err != nil {
			fmt.Println("Error getting/creating config directory:", err)
			return
		}

		// Remove configPath + "/remote" if it exists
		if _, err := os.Stat(configPath + "/remote"); err == nil {
			err := os.Remove(configPath + "/remote")
			if err != nil {
				fmt.Println("Error removing old remote config:", err)
				return
			}
		}

		f, err := os.Create(configPath + "/remote")

		if err != nil {
			fmt.Println("Error creating remote config:", err)
			return
		}

		remoteCfg := helpers.ConfigRemote{
			Name:     args[0],
			Hostname: args[1],
			Username: usernameFlag,
			Key:      keyFlag,
			KeyPass:  pass,
		}

		// Attempt a connection to the remote server
		cli, err := remoteCfg.Connect()

		if err != nil {
			fmt.Println("Error connecting to remote server:", err)
			return
		}

		// Try to get the server info
		sv := cli.ServerVersion()

		fmt.Println("Connected to remote server with ssh version:", string(sv))

		bytes, err := json.Marshal(remoteCfg)

		if err != nil {
			fmt.Println("Error marshalling remote config:", err)
			return
		}

		_, err = f.Write(bytes)

		if err != nil {
			fmt.Println("Error writing remote config:", err)
			return
		}

		f.Close()
	},
}

func init() {
	// remote set
	setRemoteCmd.Flags().StringP("key", "k", "", "The private key that will be used to connect to the remote server.")
	setRemoteCmd.MarkFlagRequired("key")

	setRemoteCmd.Flags().StringP("username", "u", "", "The username that will be used to connect to the remote server.")
	setRemoteCmd.MarkFlagRequired("username")

	setRemoteCmd.Flags().BoolP("pass", "p", false, "Whether the private key has a password or not.")

	remoteCmd.AddCommand(setRemoteCmd)
	rootCmd.AddCommand(remoteCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// remoteCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// remoteCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
