/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/InfinityBotList/ibl/helpers"
	"github.com/InfinityBotList/ibl/types"
	"github.com/spf13/cobra"
)

// Does not include all keys sent by API, only ones we need
type PermDetailMap struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

type PermissionResponse struct {
	Perms []PermDetailMap `json:"perms"`
}

// typegenCmd represents the typegen command
var genEnumsCmd = &cobra.Command{
	Use:   "genenums",
	Short: "Generate enums for teams and other objects",
	Long:  `Generate enums for teams and other objects`,
	Run: func(cmd *cobra.Command, args []string) {
		permRes, err := helpers.NewReq().Get("teams/meta/permissions").Do()

		if err != nil {
			fmt.Println("Error getting permissions:", err)
			return
		}

		var perms PermissionResponse

		err = permRes.JsonOk(&perms)

		if err != nil {
			fmt.Println("Error getting permissions:", err)
			return
		}

		teamPermEnumStr := "export enum TeamPermissions {\n"

		for _, perm := range perms.Perms {
			teamPermEnumStr += "	" + strings.ReplaceAll(perm.Name, " ", "") + " = \"" + perm.ID + "\", // " + perm.Desc + "\n"
		}

		teamPermEnumStr += "}"

		// Save to /iblcdn/public/dev/bindings/popplio/team-perms.ts
		f, err := os.Create("/iblcdn/public/dev/bindings/popplio/team-perms.ts")

		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}

		_, err = f.WriteString(teamPermEnumStr)

		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
	},
}

func init() {
	if helpers.DevMode().Allows(types.DevModeFull) {
		rootCmd.AddCommand(genEnumsCmd)
	}
}
