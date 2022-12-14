/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// cfgCmd represents the cfg command
var cfgCmd = &cobra.Command{
	Use:   "cfg",
	Short: "Configuration system",
	Long:  `Configure the "ibl" client (authentication and other settings).`,
}

var loginCmd = &cobra.Command{
	Use:     "login TYPE ID TOKEN",
	Short:   "Login to the IBL API",
	Long:    `Login to the IBL API using a bot or user token.`,
	Aliases: []string{"auth", "a", "l"},
	Args:    cobra.ExactArgs(3),
	Run:     func(cmd *cobra.Command, args []string) {},
}

func init() {
	// login
	cfgCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(cfgCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// cfgCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// cfgCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
