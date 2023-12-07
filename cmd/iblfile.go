/*
Copyright © 2022 Infinity Bot List
*/
package cmd

import (
	"bytes"
	"compress/lzw"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/infinitybotlist/iblfile"
	"github.com/spf13/cobra"
)

var iblFileCmd = &cobra.Command{
	Use:   "file",
	Short: "IBL file information",
	Long:  "Retrieve information about an IBL file",
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Gets info about a ibl file",
	Long:  `Gets info about a ibl file`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		showPubKey := cmd.Flag("show-pubkey").Value.String() == "true"

		filename := args[0]

		f, err := os.Open(filename)

		if err != nil {
			fmt.Println("ERROR: Failed to open file:", err)
			os.Exit(1)
		}

		defer f.Close()

		sections, meta, err := iblfile.ParseData(f)

		if err != nil {
			fmt.Println("ERROR:", err)
			os.Exit(1)
		}

		fmt.Println("\n== Extra Info ==")
		if len(meta.EncryptionData) > 0 {
			fmt.Println("File contains encrypted sections")

			for sectionName, enc := range meta.EncryptionData {
				fmt.Println("\n=> Encrypted section '" + sectionName + "'")

				if showPubKey {
					fmt.Print("Public Key:\n")
					fmt.Print(string(enc.PEM))
				}
			}
		} else {
			fmt.Println("File is not encrypted")
		}

		format, err := iblfile.GetFormat(meta.Type)

		if err != nil {
			fmt.Println("WARNING: Unknown/unregistered format:", meta.Type, "due to error: ", err)
			os.Exit(1)
		}

		if format.GetExtended != nil {
			extendedMeta, err := format.GetExtended(sections, meta)

			if err != nil {
				fmt.Println("ERROR:", err)
				os.Exit(1)
			}

			fmt.Println("\n== Extended Info ==")

			for k, v := range extendedMeta {
				// If v is a struct marshal it into newline seperated key: value format
				fmt.Println(k+":", v)
			}
		}
	},
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

		// LZW migration. Older versions of the tool used LZW compression for files but this actually worsens file sizes and wastes CPU
		//
		// Try to lzw decompress the file, if so we can upgrade the file by simply decompressing it
		// This is a hacky way to do it, but it works
		var buf bytes.Buffer

		r := lzw.NewReader(inputFile, lzw.LSB, 8)

		_, err = io.Copy(&buf, r)

		// Upgrade to frostpaw-rev6
		if err == nil {
			fmt.Println("INFO: Detected LZW compressed file, upgrading frostpaw-rev5-e1 to frostpaw-rev6")
			sections, err := iblfile.RawDataParse(&buf)

			if err != nil {
				fmt.Println("ERROR: Failed to parse input file:", err)
				os.Exit(1)
			}

			type EncryptionData struct {
				// Public key to encrypt data with
				PEM []byte `json:"p"`

				// Encrypted OEAP keys
				Keys [][]byte `json:"k"`

				// Encryption nonce
				Nonce string `json:"n"`

				// Whether or not symmetric encryption is being used
				//
				// If this option is set, then a `privKey` section MUST be present (e.g. using an AutoEncrypted file)
				Symmetric bool `json:"s"`
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

			err = json.NewDecoder(bytes.NewBuffer(sections["meta"].Bytes())).Decode(&meta)

			if err != nil {
				fmt.Println("ERROR: Failed to decode metadata:", err)
				os.Exit(1)
			}

			meta.Protocol = "frostpaw-rev6"

			var bufNew = bytes.NewBuffer([]byte{})

			err = json.NewEncoder(bufNew).Encode(meta)

			if err != nil {
				fmt.Println("ERROR: Failed to encode metadata:", err)
				os.Exit(1)
			}

			sections["meta"] = bufNew

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

			os.Exit(0)
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
	infoCmd.PersistentFlags().Bool("show-pubkey", false, "Whether or not to show the public key for the encrypted data")

	iblFileCmd.AddCommand(infoCmd)
	iblFileCmd.AddCommand(iblFileUpgrade)
	rootCmd.AddCommand(iblFileCmd)
}
