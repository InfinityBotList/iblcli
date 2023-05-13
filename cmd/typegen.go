/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/InfinityBotList/ibl/helpers"
	"github.com/spf13/cobra"
)

type NginxFile struct {
	Name string `json:"name"`
}

type FileList []NginxFile

// Does not include all keys sent by API, only ones we need
type PermDetailMap struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

type PermissionResponse struct {
	Perms []PermDetailMap `json:"perms"`
}

func addTypings(remoteDir, localDir string) error {
	fmt.Println("=>", localDir, "( "+remoteDir+" )")

	res, err := helpers.NewReq().Get("json/" + remoteDir).Do()

	if err != nil {
		fmt.Println("Error getting response:", err)
		return err
	}

	var list FileList

	err = res.JsonOk(&list)

	if err != nil {
		fmt.Println("Error unmarshalling response:", err)
		return err
	}

	os.MkdirAll("src/utils/generated/"+localDir, 0755)

	for i, file := range list {
		fmt.Println("["+strconv.Itoa(i+1)+"/"+strconv.Itoa(len(list))+"] Downloading", file.Name)

		res, err := helpers.NewReq().Get(remoteDir + "/" + file.Name).Do()

		if err != nil {
			fmt.Println("Error downloading file:", err)
			return err
		}

		bytes, err := res.BodyOk()

		if err != nil {
			fmt.Println("Error reading file")
			return err
		}

		f, err := os.Create("src/utils/generated/" + localDir + "/" + file.Name)

		if err != nil {
			fmt.Println("Error creating file:", err)
			return err
		}

		_, err = f.Write(bytes)

		if err != nil {
			fmt.Println("Error writing file:", err)
			return err
		}
	}

	return nil
}

// typegenCmd represents the typegen command
var typegenCmd = &cobra.Command{
	Use:   "typegen",
	Short: "Generate typings for the frontend of IBL",
	Long:  `Generate typings for the frontend of IBL.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Remove any existing src/utils/generated folder if it exists
		os.RemoveAll("src/utils/generated")

		os.MkdirAll("src/utils/generated", 0755)

		helpers.ClientURL = "https://cdn.infinitybots.gg"

		// Find list of all bindings
		res, err := helpers.NewReq().Get("json/dev/bindings").Do()

		if err != nil {
			fmt.Println("Error getting response:", err)
			return
		}

		var blist FileList

		err = res.JsonOk(&blist)

		if err != nil {
			fmt.Println("Error unmarshalling response:", err)
		}

		for _, binding := range blist {
			err := addTypings("dev/bindings/"+binding.Name, binding.Name)

			if err != nil {
				fmt.Println("Error with "+binding.Name, err)
				return
			}
		}

		helpers.ClientURL = helpers.APIUrl

		// Team Permissions
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

		// Write to teamPerms.ts
		f, err := os.Create("src/utils/generated/teamPerms.ts")

		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}

		_, err = f.WriteString(teamPermEnumStr)

		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(typegenCmd)
}
