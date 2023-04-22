/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"archive/tar"
	"bytes"
	"compress/lzw"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/InfinityBotList/ibl/helpers"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/spf13/cobra"
)

func mkBackup() {
	cleanup := func() {
		fmt.Println("Cleaning up...")

		// delete all files in work directory
		err := os.RemoveAll("work")

		if err != nil {
			fmt.Println("Error cleaning up:", err)
		}
	}

	cleanup()

	// create a work directory
	err := os.Mkdir("work", 0755)

	if err != nil {
		fmt.Println("Error creating work directory:", err)
		cleanup()
		return
	}

	var passFile = "/db/bakkey"

	if os.Getenv("ALT_BAK_KEY") != "" {
		passFile = os.Getenv("ALT_BAK_KEY")
	}

	var outFolder = "/db/backups"

	if os.Getenv("ALT_BAK_OUT") != "" {
		outFolder = os.Getenv("ALT_BAK_OUT")
	}

	// Read the password from the file
	pass, err := os.ReadFile(passFile)

	if err != nil {
		cleanup()
		fmt.Println(err)
		return
	}

	// Encrypt sample data
	salt := crypto.RandString(8)
	passHashed := helpers.GenKey(string(pass), salt)

	// Create a new backup using pg_dump
	backupCmd := exec.Command("pg_dump", "-Fc", "--no-owner", "-d", "infinity", "-f", "work/schema.sql")

	backupCmd.Env = helpers.GetEnv()

	err = backupCmd.Run()

	if err != nil {
		fmt.Println(err)
		cleanup()
		return
	}

	file, err := os.ReadFile("work/schema.sql")

	if err != nil {
		fmt.Println(err)
		cleanup()
		return
	}

	// Cleanup work now
	cleanup()

	// Encrypt the file
	encFile, err := helpers.Encrypt(passHashed, file)

	if err != nil {
		fmt.Println(err)
		cleanup()
		return
	}

	// Get current datetime formatted
	t := time.Now().Format("2006-01-02-15:04:05")

	// Create a tar file as a io.Writer, NOT a file
	tarFile := bytes.NewBuffer([]byte{})

	if err != nil {
		fmt.Println("Failed to create tar file:", err)
		cleanup()
		return
	}

	tarWriter := tar.NewWriter(tarFile)

	// Write data buf to tar file
	err = helpers.TarAddBuf(tarWriter, bytes.NewBuffer(encFile), "data")

	if err != nil {
		fmt.Println("Failed to write data to tar file:", err)
		cleanup()
		return
	}

	// Write salt buf to tar file
	err = helpers.TarAddBuf(tarWriter, bytes.NewBuffer([]byte(salt)), "salt")

	if err != nil {
		fmt.Println("Failed to write salt to tar file:", err)
		cleanup()
		return
	}

	// Close tar file
	tarWriter.Close()

	// Compress
	compressed, err := os.Create(outFolder + "/backup-" + t + ".iblbackup")

	if err != nil {
		fmt.Println("Failed to create compressed file:", err)
		cleanup()
		return
	}

	defer compressed.Close()

	w := lzw.NewWriter(compressed, lzw.LSB, 8)

	_, err = io.Copy(w, tarFile)

	if err != nil {
		fmt.Println("Failed to compress file:", err)
		cleanup()
		return
	}

	w.Close()
}

// backupdbCmd represents the backupdb command
var backupdbCmd = &cobra.Command{
	Use:   "backupdb",
	Short: "Database backup commands",
	Long:  `Database backup commands`,
}

var backupdbNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new database backup",
	Long:  `Create a new database backup`,
	Run: func(cmd *cobra.Command, args []string) {
		mkBackup()
	},
}

var longRunningBackupCmd = &cobra.Command{
	Use:   "long",
	Short: "Create a new database backup every 8 hours",
	Long:  `Create a new database backup every 8 hours`,
	Run: func(cmd *cobra.Command, args []string) {
		for {
			fmt.Println("Making backup at", time.Now())
			mkBackup()
			fmt.Println("Sleeping for 8 hours...")
			time.Sleep(8 * time.Hour)
		}
	},
}

var decrBackup = &cobra.Command{
	Use:   "decr file output",
	Short: "Decrypt and decompress a backup file",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// Read the password from the file
		passFile := "/certs/bakkey"

		if os.Getenv("ALT_BAK_KEY") != "" {
			passFile = os.Getenv("ALT_BAK_KEY")
		}

		// Read the password from the file
		pass, err := os.ReadFile(passFile)

		if err != nil {
			fmt.Println(err)
			return
		}

		// Open the file
		fmt.Println("Opening", args[0])
		file, err := os.Open(args[0])

		if err != nil {
			fmt.Println(err)
			return
		}

		// Extract seed file using lzw to buffer
		tarBuf := bytes.NewBuffer([]byte{})
		r := lzw.NewReader(file, lzw.LSB, 8)

		_, err = io.Copy(tarBuf, r)

		if err != nil {
			fmt.Println("Failed to decompress backup file:", err)
			return
		}

		// Get size of decompressed file
		tarSize := tarBuf.Len()

		fmt.Println("Decompressed size: ", tarSize, "bytes")

		// Extract tar file to map of buffers
		tarReader := tar.NewReader(tarBuf)

		files := make(map[string]*bytes.Buffer)

		for {
			// Read next file from tar header
			header, err := tarReader.Next()

			if err == io.EOF {
				break
			}

			if err != nil {
				fmt.Println("Failed to read tar file:", err)
				return
			}

			// Read file into buffer
			buf := bytes.NewBuffer([]byte{})

			_, err = io.Copy(buf, tarReader)

			if err != nil {
				fmt.Println("Failed to read tar file:", err)
				return
			}

			// Save file to map
			files[header.Name] = buf
		}

		fmt.Println("Got map keys:", helpers.MapKeys(files))

		// Decrypt data
		ePassHashed := helpers.GenKey(string(pass), files["salt"].String())
		data, err := helpers.Decrypt([]byte(ePassHashed), files["data"].Bytes())

		if err != nil {
			fmt.Println("Failed to decrypt data:", err)
			return
		}

		// Write data to file
		err = os.WriteFile(args[1], data, 0644)

		if err != nil {
			fmt.Println("Failed to write data to file:", err)
			return
		}
	},
}

func init() {
	backupdbCmd.AddCommand(backupdbNewCmd)
	backupdbCmd.AddCommand(longRunningBackupCmd)
	backupdbCmd.AddCommand(decrBackup)
	rootCmd.AddCommand(backupdbCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// backupdbCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// backupdbCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
