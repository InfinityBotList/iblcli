/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"archive/tar"
	"bytes"
	"compress/lzw"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/InfinityBotList/ibl/helpers"

	"github.com/jackc/pgtype"
	"github.com/spf13/cobra"
)

type Meta struct {
	CreatedAt time.Time
	Nonce     string
}

type SourceParsed struct {
	Data  map[string]any
	Table string
}

const seedApiVer = "ravenpaw"

// seedCmd represents the seed command
var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Database Seeding Commands",
	Long:  ``,
}

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Creates a new database seed",
	Long:  `Creates a new database seed`,
	Run: func(cmd *cobra.Command, args []string) {
		cleanup := func() {
			fmt.Println("Cleaning up...")

			// delete all files in work directory
			err := os.RemoveAll("work")

			if err != nil {
				fmt.Println("Error cleaning up:", err)
			}
		}

		// create a work directory
		err := os.Mkdir("work", 0755)

		if err != nil {
			fmt.Println("Error creating work directory")
			cleanup()
			return
		}

		fmt.Println("Creating database backup as schema.sql")

		backupCmd := exec.Command("pg_dump", "-Fc", "--schema-only", "--no-owner", "-d", "infinity", "-f", "work/schema.sql")

		backupCmd.Env = helpers.GetEnv()

		err = backupCmd.Run()

		if err != nil {
			fmt.Println(err)
			cleanup()
			return
		}

		fmt.Println("Copying seed sample data")

		type SeedSource struct {
			Table  string
			Column string
			Value  any
			Desc   string
		}

		var seedDataSrc = []SeedSource{
			{
				Table:  "users",
				Column: "user_id",
				Value:  "563808552288780322",
				Desc:   "Rootsprings User Info",
			},
			{
				Table:  "users",
				Column: "user_id",
				Value:  "728871946456137770",
				Desc:   "Burgerkings User Info",
			},
			{
				Table:  "users",
				Column: "user_id",
				Value:  "510065483693817867",
				Desc:   "Toxic Devs User Info",
			},
			{
				Table:  "bots",
				Column: "bot_id",
				Value:  "721279531939397673",
				Desc:   "Bristlefrost Bot Info",
			},
			{
				Table:  "bots",
				Column: "bot_id",
				Value:  "815553000470478850",
				Desc:   "IBL Bot Bot Info [for reviews]",
			},
			{
				Table:  "reviews",
				Column: "bot_id",
				Value:  "815553000470478850",
				Desc:   "IBL Bot Review",
			},
			{
				Table:  "tickets",
				Column: "id",
				Value:  67021157,
				Desc:   "IBL Ticket (toxics test tickets)",
			},
			{
				Table:  "transcripts",
				Column: "id",
				Value:  67021157,
				Desc:   "IBL Ticket Transcript (toxics test transcript)",
			},
			{
				Table:  "votes",
				Column: "bot_id",
				Value:  "721279531939397673",
				Desc:   "Bristlefrost Votes",
			},
		}

		pool, err := helpers.GetPool()

		if err != nil {
			fmt.Println("Pool connection failed:", err)
			cleanup()
			return
		}

		var parsed = []SourceParsed{}

		for _, source := range seedDataSrc {
			var jsonRow pgtype.JSON
			err := pool.QueryRow(context.Background(), "SELECT row_to_json("+source.Table+") FROM "+source.Table+" WHERE "+source.Column+" = $1", source.Value).Scan(&jsonRow)

			if err != nil {
				fmt.Println("Failed to get data for", source.Desc, err)
				cleanup()
				return
			}

			var data map[string]any
			err = jsonRow.AssignTo(&data)

			if err != nil {
				fmt.Println("Failed to parse data for", source.Desc, err)
				cleanup()
				return
			}

			// Strip tokens from data
			var parsedData = make(map[string]any)
			for k, v := range data {
				/*
					if k == "premium_period_length" {
						continue // Ignore column, is not supported
					}*/
				if k == "webhook" {
					parsedData[k] = "https://testhook.xyz"
				} else if strings.Contains(k, "token") || strings.Contains(k, "web") {
					parsedData[k] = helpers.RandString(128)
				} else if strings.Contains(k, "unique") {
					parsedData[k] = []any{}
				} else {
					parsedData[k] = v
				}
			}

			parsed = append(parsed, SourceParsed{Data: data, Table: source.Table})
		}

		// Create sample.json buffer
		sampleBuf := bytes.NewBuffer([]byte{})

		// Write sample data to buffer
		enc := json.NewEncoder(sampleBuf)

		err = enc.Encode(parsed)

		if err != nil {
			fmt.Println("Failed to write sample data:", err)
			cleanup()
			return
		}

		// Write metadata to buffer
		mdBuf := bytes.NewBuffer([]byte{})

		// Write metadata to md file
		metadata := Meta{
			CreatedAt: time.Now(),
			Nonce:     helpers.RandString(32),
		}

		enc = json.NewEncoder(mdBuf)

		err = enc.Encode(metadata)

		if err != nil {
			fmt.Println("Failed to write metadata:", err)
			cleanup()
			return
		}

		// Create a tar file as a io.Writer, NOT a file
		tarFile := bytes.NewBuffer([]byte{})

		if err != nil {
			fmt.Println("Failed to create tar file:", err)
			cleanup()
			return
		}

		tarWriter := tar.NewWriter(tarFile)

		// Write sample buf to tar file
		err = helpers.TarAddBuf(tarWriter, sampleBuf, "sample")

		if err != nil {
			fmt.Println("Failed to write sample data to tar file:", err)
			cleanup()
			return
		}

		// Write metadata buf to tar file
		err = helpers.TarAddBuf(tarWriter, mdBuf, "md")

		if err != nil {
			fmt.Println("Failed to write metadata to tar file:", err)
			cleanup()
			return
		}

		// Write schema to tar file
		schemaFile, err := os.Open("work/schema.sql")

		if err != nil {
			fmt.Println("Failed to open schema file:", err)
			cleanup()
			return
		}

		// -- convert to bytes.Buffer
		schemaBuf := bytes.NewBuffer([]byte{})

		_, err = schemaBuf.ReadFrom(schemaFile)

		if err != nil {
			fmt.Println("Failed to read schema file:", err)
			cleanup()
			return
		}

		err = helpers.TarAddBuf(tarWriter, schemaBuf, "schema")

		if err != nil {
			fmt.Println("Failed to write schema to tar file:", err)
			cleanup()
			return
		}

		// Write required iblseed api version
		err = helpers.TarAddBuf(tarWriter, bytes.NewBufferString(seedApiVer), "seedapi")

		if err != nil {
			fmt.Println("Failed to write seedapi to tar file:", err)
			cleanup()
			return
		}

		// Close tar file
		tarWriter.Close()

		compressed, err := os.Create("seed.iblseed")

		if err != nil {
			fmt.Println("Failed to create compressed file:", err)
			cleanup()
			return
		}

		defer compressed.Close()

		// Compress
		w := lzw.NewWriter(compressed, lzw.LSB, 8)

		_, err = io.Copy(w, tarFile)

		if err != nil {
			fmt.Println("Failed to compress file:", err)
			cleanup()
			return
		}

		w.Close()
		cleanup()

		// Try to find /iblseeds folder (devel assets server)
		_, err = os.Stat("/iblseeds")

		if err == nil {
			fmt.Println("Mpving seed to found folder /iblseeds")
			err = os.Rename("seed.iblseed", "/iblseeds/seed.iblseed")

			if err != nil {
				fmt.Println("Failed to copy seed to devel assets server:", err)
			}
		}
	},
}

var applyCmd = &cobra.Command{
	Use:     "apply",
	Example: "apply latest",
	Short:   "Apply a seed to the database. You must specify either 'latest' or the path to a seed file.",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cleanup := func() {
			fmt.Println("Cleaning up...")

			// delete all files in work directory
			err := os.RemoveAll("work")

			if err != nil {
				fmt.Println("Error cleaning up:", err)
			}
		}

		// create a work directory
		err := os.Mkdir("work", 0755)

		if err != nil {
			fmt.Println("Error creating work directory")
			cleanup()
			return
		}

		// Check args as to which file to use
		seedFile := args[0]

		assetsUrl := helpers.GetAssetsURL()

		if seedFile == "latest" {
			// Ensure temp.iblseed is removed if it exists
			_, err := os.Stat("temp.iblseed")

			if err == nil {
				err = os.Remove("temp.iblseed")

				if err != nil {
					fmt.Println("Failed to remove temp seed:", err)
					cleanup()
					return
				}
			}

			// Download seedfile with progress bar
			data, err := helpers.DownloadFileWithProgress(assetsUrl + "/seed.iblseed")

			if err != nil {
				fmt.Println("Failed to download seed file:", err)
				cleanup()
				return
			}

			// Write seedfile to disk as temp.iblseed
			f, err := os.Create("temp.iblseed")

			if err != nil {
				fmt.Println("Failed to create temp file:", err)
				cleanup()
				return
			}

			defer f.Close()

			_, err = f.Write(data)

			if err != nil {
				fmt.Println("Failed to download seed file:", err)
				cleanup()
				return
			}

			seedFile = "temp.iblseed"
		}

		// Open seed file
		f, err := os.Open(seedFile)

		if err != nil {
			fmt.Println("Failed to open seed file:", err)
			cleanup()
			return
		}

		defer f.Close()

		// Extract seed file using lzw to buffer
		tarBuf := bytes.NewBuffer([]byte{})
		r := lzw.NewReader(f, lzw.LSB, 8)

		_, err = io.Copy(tarBuf, r)

		if err != nil {
			fmt.Println("Failed to decompress seed file:", err)
			cleanup()
			return
		}

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
				cleanup()
				return
			}

			// Read file into buffer
			buf := bytes.NewBuffer([]byte{})

			_, err = io.Copy(buf, tarReader)

			if err != nil {
				fmt.Println("Failed to read tar file:", err)
				cleanup()
				return
			}

			// Save file to map
			files[header.Name] = buf
		}

		fmt.Println("Got map keys:", helpers.MapKeys(files))

		seedApi, ok := files["seedapi"]

		if !ok {
			fmt.Println("Seed file is of an invalid version [no version found]")
			cleanup()
			return
		}

		seedApiStr := seedApi.String()

		if seedApiStr != seedApiVer {
			fmt.Println("Seed file is of an invalid version [version is", seedApiStr, "but expected", seedApiVer, "]")
			return
		}

		// Unpack schema to temp file
		schemaBuf, ok := files["schema"]

		if !ok {
			fmt.Println("Seed file is corrupt [no schema]")
			cleanup()
			return
		}

		schemaFile, err := os.Create("work/temp.sql")

		if err != nil {
			fmt.Println("Failed to create temp file:", err)
			cleanup()
			return
		}

		defer schemaFile.Close()

		_, err = schemaFile.Write(schemaBuf.Bytes())

		if err != nil {
			fmt.Println("Failed to write temp file:", err)
			cleanup()
			return
		}

		// Ensure PGDATABASE is NOT set
		os.Unsetenv("PGDATABASE")

		pool, err := helpers.GetPoolNoUrl()

		if err != nil {
			fmt.Println("Failed to acquire database pool:", err)
			cleanup()
			return
		}

		// Create role root
		_, err = pool.Exec(context.Background(), "CREATE ROLE root")

		// Check if error is role already exists
		if err != nil {
			if !strings.Contains(err.Error(), "already exists") {
				fmt.Println("Role root already exists, continuing...")
			} else {
				fmt.Println("Failed to create role root:", err)
				cleanup()
				return
			}
		}

		_, err = pool.Exec(context.Background(), "DROP DATABASE infinity")

		if err != nil {
			fmt.Println("Failed to drop database infinity:", err)
			cleanup()
			return
		}

		_, err = pool.Exec(context.Background(), "CREATE DATABASE infinity")

		if err != nil {
			fmt.Println("Failed to create database infinity:", err)
			cleanup()
			return
		}

		fmt.Println("Restoring database backup")

		// Use pg_restore to restore seedman.tmp
		restoreCmd := exec.Command("pg_restore", "-d", "infinity", "-h", "localhost", "-p", "5432", "work/temp.sql")

		restoreCmd.Stdout = os.Stdout
		restoreCmd.Stderr = os.Stderr

		outCode := restoreCmd.Run()

		if outCode != nil {
			fmt.Println("Failed to restore database backup with error:", outCode)
			cleanup()
			return
		}

		if !restoreCmd.ProcessState.Success() {
			fmt.Println("Failed to restore database backup with unknown error")
			cleanup()
			return
		}

		os.Setenv("PGDATABASE", "infinity")

		pool, err = helpers.GetPool()

		if err != nil {
			fmt.Println("Failed to acquire database pool for newly created database:", err)
			cleanup()
			return
		}

		// Extract out md
		mdBuf, ok := files["md"]

		if !ok {
			fmt.Println("Seed file is corrupt [no md]")
			cleanup()
			return
		}

		var md Meta

		err = json.Unmarshal(mdBuf.Bytes(), &md)

		if err != nil {
			fmt.Println("Failed to unmarshal md:", err)
			cleanup()
			return
		}

		_, err = pool.Exec(context.Background(), "CREATE TABLE seed_info (nonce TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL)")

		if err != nil {
			fmt.Println("Failed to create seed_info table:", err)
			cleanup()
			return
		}

		_, err = pool.Exec(context.Background(), "INSERT INTO seed_info (nonce, created_at) VALUES ($1, $2)", md.Nonce, md.CreatedAt)

		if err != nil {
			fmt.Println("Failed to insert seed info:", err)
			cleanup()
			return
		}

		// Now finally extract out seed data
		seedBuf, ok := files["sample"]

		if !ok {
			fmt.Println("Seed file is corrupt [no sample]")
			cleanup()
			return
		}

		var seed []SourceParsed

		err = json.Unmarshal(seedBuf.Bytes(), &seed)

		if err != nil {
			fmt.Println("Failed to unmarshal seed:", err)
			cleanup()
			return
		}

		// Loop over seed data and insert into db
		for _, s := range seed {
			var i int = 1
			var args []any
			var keys []string
			var sqlArgs []string

			// Loop over all map props
			for k, v := range s.Data {
				keys = append(keys, k)
				args = append(args, v)
				sqlArgs = append(sqlArgs, "$"+strconv.Itoa(i))
				i++
			}

			// Create sql string
			fmt.Println(s.Table)
			sql := "INSERT INTO " + s.Table + " (" + strings.Join(keys, ", ") + ") VALUES (" + strings.Join(sqlArgs, ", ") + ")"

			fmt.Println(sql, args)

			_, err := pool.Exec(context.Background(), sql, args...)

			if err != nil {
				fmt.Println("Failed to insert seed data:", err)
				cleanup()
				return
			}
		}

	},
}

func init() {
	// seed new
	seedCmd.AddCommand(newCmd)

	// seed apply
	seedCmd.AddCommand(applyCmd)

	rootCmd.AddCommand(seedCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// seedCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// seedCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
