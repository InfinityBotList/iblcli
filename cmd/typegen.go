/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/InfinityBotList/ibl/helpers"
	"github.com/InfinityBotList/ibl/types"
	"github.com/spf13/cobra"
)

type NginxFile struct {
	Name string `json:"name"`
}

type FileList []NginxFile

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

	var path = "src/utils/generated"

	if os.Getenv("IBL_PATH") != "" {
		path = os.Getenv("IBL_PATH")
	}

	os.MkdirAll(path+"/"+localDir, 0755)

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
	},
}

func init() {
	if DevMode == types.DevModeLocal {
		rootCmd.AddCommand(typegenCmd)
	}
}
