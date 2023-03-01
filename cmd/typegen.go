/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
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

// typegenCmd represents the typegen command
var typegenCmd = &cobra.Command{
	Use:   "typegen",
	Short: "Generate typings for the frontend of IBL",
	Long:  `Generate typings for the frontend of IBL.`,
	Run: func(cmd *cobra.Command, args []string) {
		cleanup := func() {
			fmt.Println("Cleaning up...")

			// delete all files in work directory
			err := os.RemoveAll("work")

			if err != nil {
				fmt.Println("Error cleaning up:", err)
			}
		}

		defer cleanup()

		// create a work directory
		err := os.Mkdir("work", 0755)

		if err != nil {
			fmt.Println("Error creating work directory:", err)
			return
		}

		fmt.Println("Downloading RPC")
		resp, err := http.Get("https://devel.infinitybots.xyz/json/apiBindings/")

		if err != nil {
			fmt.Println("Error downloading RPC:", err)
			return
		}

		if resp.StatusCode != http.StatusOK {
			fmt.Println("Error downloading RPC:", resp.Status)
			return
		}

		var list FileList

		err = json.NewDecoder(resp.Body).Decode(&list)

		if err != nil {
			fmt.Println("Error decoding JSON:", err)
			return
		}

		for i, file := range list {
			fmt.Println("["+strconv.Itoa(i+1)+"/"+strconv.Itoa(len(list))+"] Downloading", file.Name)

			resp, err := http.Get("https://devel.infinitybots.xyz/apiBindings/" + file.Name)

			if err != nil {
				fmt.Println("Error downloading file:", err)
				return
			}

			if resp.StatusCode != http.StatusOK {
				fmt.Println("Error downloading file:", resp.Status)
				return
			}

			f, err := os.Create("work/" + file.Name)

			if err != nil {
				fmt.Println("Error creating file:", err)
				return
			}

			bytes, err := io.ReadAll(resp.Body)

			if err != nil {
				fmt.Println("Error reading body:", err)
				return
			}

			_, err = f.Write(bytes)

			if err != nil {
				fmt.Println("Error writing file:", err)
				return
			}
		}

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
		f, err := os.Create("work/teamPerms.ts")

		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}

		_, err = f.WriteString(teamPermEnumStr)

		if err != nil {
			fmt.Println("Error writing file:", err)
			return
		}

		// Check for src/utils/generated
		if _, err := os.Stat("src/utils/generated"); errors.Is(err, fs.ErrExist) {
			fmt.Println("Copying files to src/utils/generated")

			// copy all files in work to src/utils/generated
			fileList, err := os.ReadDir("work")

			if err != nil {
				fmt.Println("Error reading work directory:", err)
				return
			}

			for i, file := range fileList {
				fmt.Println("["+strconv.Itoa(i+1)+"/"+strconv.Itoa(len(list))+"] Copying", file.Name())

				// Read file
				b, err := os.ReadFile("work/" + file.Name())

				if err != nil {
					fmt.Println("Error opening file:", err)
					return
				}

				err = os.WriteFile("src/utils/generated/"+file.Name(), b, 0755)

				if err != nil {
					fmt.Println("Error creating file:", err)
					return
				}
			}
		} else {
			fmt.Println("No src/utils/generated directory, exiting")
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(typegenCmd)
}
