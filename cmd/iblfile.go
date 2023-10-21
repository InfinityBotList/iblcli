/*
Copyright © 2022 Infinity Bot List
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/InfinityBotList/ibl/internal/iblfile"
	"github.com/spf13/cobra"
)

var iblFileCmd = &cobra.Command{
	Use:   "file",
	Short: "IBL file information",
	Long:  "Retrieve information about an IBL file",
}

var iblFileUpgrade = &cobra.Command{
	Use:   "upgrade <input file> <output file>",
	Short: "Upgrade a file protocol version where possible",
	Long:  `Upgrade a file protocol version where possible. This does not upgrade format versions. To upgrade format version, use a more specific convert command provided by the format.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// Open input file
		inputFile, err := os.Open(args[0])

		if err != nil {
			fmt.Println("ERROR: Failed to open input file:", err)
			os.Exit(1)
		}

		// Open input file
		sections, err := iblfile.RawDataParse(inputFile)

		if err != nil {
			fmt.Println("ERROR: Failed to parse input file:", err)
			os.Exit(1)
		}

		metaBuf, ok := sections["meta"]

		if !ok {
			// All current protocols have a metadata section
			fmt.Println("ERROR: No metadata section found")
			os.Exit(1)
		}

		metaBytes := metaBuf.Bytes()

		var data struct {
			Protocol string `json:"p"`
		}

		err = json.NewDecoder(bytes.NewBuffer(metaBytes)).Decode(&data)

		if err != nil {
			fmt.Println("ERROR: Failed to decode metadata:", err)
			os.Exit(1)
		}

		switch data.Protocol {
		case "frostpaw-rev4-e1":
			type EncryptionData struct {
				// Public key to encrypt data with
				PEM []byte `json:"p"`

				// Encrypted OEAP keys
				Keys [][]byte `json:"k"`

				// Encryption nonce
				Nonce string `json:"n"`
			}

			type Meta struct {
				CreatedAt time.Time `json:"c"`
				Protocol  string    `json:"p"`

				// Format version
				//
				// This can be used to create breaking changes to a file type without changing the entire protocol
				FormatVersion string `json:"v,omitempty"`

				// Encryption data, if a section is encrypted
				// This is a map that maps each section to its encryption data
				EncryptionData map[string]*EncryptionData `json:"e,omitempty"`

				// Type of the file
				Type string `json:"t"`
			}

			var meta Meta

			err = json.NewDecoder(bytes.NewBuffer(metaBytes)).Decode(&meta)

			if err != nil {
				fmt.Println("ERROR: Failed to decode metadata:", err)
				os.Exit(1)
			}

			// New: Type namespacing
			fmt.Println("INFO: Namespacing types")
			var renamesMap = map[string]string{
				"backup": "db.backup",
				"seed":   "db.seed",
			}

			if newName, ok := renamesMap[meta.Type]; ok {
				meta.Type = newName
			}

			// Update the protocol to next version
			meta.Protocol = "frostpaw-rev5-e1"

			var bufNew = bytes.NewBuffer([]byte{})

			err = json.NewEncoder(bufNew).Encode(meta)

			if err != nil {
				fmt.Println("ERROR: Failed to encode metadata:", err)
				os.Exit(1)
			}

			sections["meta"] = bufNew
		default:
			fmt.Println("ERROR: Unsupported protocol version:", data.Protocol)
		}

		// Open output file
		outputFile, err := os.Create(args[1])

		if err != nil {
			fmt.Println("ERROR: Failed to open output file:", err)
			os.Exit(1)
		}

		newFile := iblfile.New()

		for name, buf := range sections {
			err = newFile.WriteSection(buf, name)

			if err != nil {
				fmt.Println("ERROR: Failed to write section:", err)
				os.Exit(1)
			}
		}

		err = newFile.WriteOutput(outputFile)

		if err != nil {
			fmt.Println("ERROR: Failed to write output file:", err)
			os.Exit(1)
		}
	},
}

func init() {
	iblFileCmd.AddCommand(iblFileUpgrade)
	rootCmd.AddCommand(iblFileCmd)
}
