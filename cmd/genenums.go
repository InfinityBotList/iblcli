/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/InfinityBotList/ibl/internal/api"
	"github.com/InfinityBotList/ibl/internal/devmode"
	"github.com/InfinityBotList/ibl/types"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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
		permRes, err := api.NewReq().Get("teams/meta/permissions").Do()

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

		teamPermEnumStr := "export enum Permission {\n"

		for _, perm := range perms.Perms {
			id := perm.ID
			value := perm.ID

			if value == "*" {
				id = "Owner"
			}

			teamPermEnumStr += "	" + strings.ReplaceAll(cases.Title(language.AmericanEnglish).String(strings.ReplaceAll(id, "_", " ")), " ", "") + " = \"" + value + "\", // " + perm.Name + " => " + perm.Desc + "\n"
		}

		teamPermEnumStr += "}"

		// Save to /silverpelt/cdn/ibl/bindings/popplio/team-perms.ts
		f, err := os.Create("/silverpelt/cdn/ibl/dev/bindings/popplio/team-perms.ts")

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
	if devmode.DevMode().Allows(types.DevModeFull) {
		rootCmd.AddCommand(genEnumsCmd)
	}
}
