/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/InfinityBotList/ibl/internal/api"
	"github.com/InfinityBotList/ibl/internal/devmode"
	"github.com/InfinityBotList/ibl/types"
	"github.com/spf13/cobra"
)

type NginxFile struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type FileList []NginxFile

func addTypings(path, remoteDir, localDir string, filter func(string) bool) error {
	fmt.Println("=>", localDir, "( "+remoteDir+" )")

	res, err := api.NewReq().Get("json/" + remoteDir).Do()

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

	// Go through every file and filter
	for i := 0; i < len(list); i++ {
		if !filter(list[i].Name) || list[i].Type != "file" {
			list = append(list[:i], list[i+1:]...)
			i--
		}
	}

	os.MkdirAll(path+"/"+localDir, 0755)

	for i, file := range list {
		fmt.Println("["+strconv.Itoa(i+1)+"/"+strconv.Itoa(len(list))+"] Downloading", file.Name)

		res, err := api.NewReq().Get(remoteDir + "/" + file.Name).Do()

		if err != nil {
			fmt.Println("Error downloading file:", err)
			return err
		}

		bytes, err := res.BodyOk()

		if err != nil {
			fmt.Println("Error reading file")
			return err
		}

		f, err := os.Create(path + "/" + localDir + "/" + file.Name)

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
		var path = "src/utils/generated"

		if os.Getenv("IBL_PATH") != "" {
			path = os.Getenv("IBL_PATH")
		}

		// Remove any existing src/utils/generated folder if it exists
		os.RemoveAll(path)

		os.MkdirAll(path, 0755)

		api.ClientURL = "https://cdn.infinitybots.gg"

		// Find list of all bindings
		res, err := api.NewReq().Get("json/dev/bindings").Do()

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
			err := addTypings(path, "dev/bindings/"+binding.Name, binding.Name, func(name string) bool {
				return strings.HasSuffix(name, ".ts")
			})

			if err != nil {
				fmt.Println("Error with "+binding.Name, err)
				return
			}
		}

		api.ClientURL = api.APIUrl
	},
}

// goTypegenCmd represents the gotypegen command
var goTypeGen = &cobra.Command{
	Use:   "gotypegen",
	Short: "Download all popplio go typings",
	Long:  `Download all popplio go typings`,
	Run: func(cmd *cobra.Command, args []string) {
		var path = "types/popltypes"

		if os.Getenv("IBL_PATH") != "" {
			path = os.Getenv("IBL_PATH")
		}

		// Remove any existing src/utils/generated folder if it exists
		os.RemoveAll(path)

		os.MkdirAll(path, 0755)

		api.ClientURL = "https://cdn.infinitybots.gg"

		err := addTypings(path, "dev/bindings/popplio/go/types", "", func(name string) bool {
			return strings.HasSuffix(name, ".go")
		})

		if err != nil {
			fmt.Println("Error downloading types", err)
			return
		}

		api.ClientURL = api.APIUrl
	},
}

func init() {
	if devmode.DevMode().Allows(types.DevModeLocal) {
		rootCmd.AddCommand(typegenCmd)
		rootCmd.AddCommand(goTypeGen)
	}
}
