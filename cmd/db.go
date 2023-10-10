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
	"strings"
	"time"

	"github.com/InfinityBotList/ibl/internal/agents/dbcommon"
	"github.com/InfinityBotList/ibl/internal/agents/dbparser"
	"github.com/InfinityBotList/ibl/internal/copyfile"
	"github.com/InfinityBotList/ibl/internal/devmode"
	"github.com/InfinityBotList/ibl/internal/downloader"
	"github.com/InfinityBotList/ibl/internal/links"
	"github.com/InfinityBotList/ibl/types"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/spf13/cobra"
)

var coreSeedTables = []string{
	"changelogs",
	"partner_types",
}

const seedApiVer = "frostpaw-rev1" // e means encryption protocol version

type Meta struct {
	CreatedAt time.Time `json:"c"`
	Nonce     string    `json:"n"`
	SeedVer   string    `json:"v"`
}

type SourceParsed struct {
	Data  map[string]any
	Table string
}

// Adds a buffer to a tar archive
func tarAddBuf(tarWriter *tar.Writer, buf *bytes.Buffer, name string) error {
	err := tarWriter.WriteHeader(&tar.Header{
		Name: name,
		Mode: 0600,
		Size: int64(buf.Len()),
	})

	if err != nil {
		return err
	}

	_, err = tarWriter.Write(buf.Bytes())

	if err != nil {
		return err
	}

	return nil
}

func mapKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// seedCmd represents the seed command
var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Database Seeding Commands",
	Long:  ``,
}

// newCmd represents the new command
var seedNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Creates a new database seed",
	Long:  `Creates a new database seed`,
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			fmt.Println("Cleaning up...")

			// delete all files in work directory
			err := os.RemoveAll("work")

			if err != nil {
				fmt.Println("Error cleaning up:", err)
			}
		}()

		if os.Getenv("ALLOW_ROOT") != "true" {
			// Check if user is root
			if os.Geteuid() == 0 {
				fmt.Println("You must not run this command as root!")
				return
			}
		}

		// create a work directory
		err := os.Mkdir("work", 0755)

		if err != nil {
			fmt.Println("Error creating work directory:", err)
			return
		}

		fmt.Println("Creating database backup as schema.sql")

		backupCmd := exec.Command("pg_dump", "-Fc", "--schema-only", "--no-owner", "-d", "infinity", "-f", "work/schema.sql")

		backupCmd.Env = dbcommon.CreateEnv()

		err = backupCmd.Run()

		if err != nil {
			fmt.Println(err)
			return
		}

		// Write metadata to buffer
		mdBuf := bytes.NewBuffer([]byte{})

		// Write metadata to md file
		metadata := Meta{
			CreatedAt: time.Now(),
			Nonce:     crypto.RandString(32),
			SeedVer:   seedApiVer,
		}

		enc := json.NewEncoder(mdBuf)

		err = enc.Encode(metadata)

		if err != nil {
			fmt.Println("Failed to write metadata:", err)
			return
		}

		// Create backup of some core tables
		for i, table := range coreSeedTables {
			fmt.Printf("Backing up table: [%d/%d] %s\n", i+1, len(coreSeedTables), table)

			// Create backup using pg_dump
			backupCmd = exec.Command("pg_dump", "-Fc", "-d", "infinity", "--data-only", "-t", table, "-f", fmt.Sprintf("work/%s.sql", table))

			backupCmd.Env = dbcommon.CreateEnv()

			err = backupCmd.Run()

			if err != nil {
				fmt.Println("Failed to create backup:", err)
				return
			}
		}

		// Create a tar file as a io.Writer, NOT a file
		tarFile := bytes.NewBuffer([]byte{})

		if err != nil {
			fmt.Println("Failed to create tar file:", err)
			return
		}

		tarWriter := tar.NewWriter(tarFile)

		// Write metadata buf to tar file
		err = tarAddBuf(tarWriter, mdBuf, "meta")

		if err != nil {
			fmt.Println("Failed to write metadata to tar file:", err)
			return
		}

		// Write schema to tar file
		schemaFile, err := os.Open("work/schema.sql")

		if err != nil {
			fmt.Println("Failed to open schema file:", err)
			return
		}

		// -- convert to bytes.Buffer
		schemaBuf := bytes.NewBuffer([]byte{})

		_, err = schemaBuf.ReadFrom(schemaFile)

		if err != nil {
			fmt.Println("Failed to read schema file:", err)
			return
		}

		// Write metadata buf to tar file
		err = tarAddBuf(tarWriter, schemaBuf, "schema")

		if err != nil {
			fmt.Println("Failed to write schema to tar file:", err)
			return
		}

		// Write backup table assets to tar file
		for _, table := range coreSeedTables {
			// Open backup file
			backupFile, err := os.Open(fmt.Sprintf("work/%s.sql", table))

			if err != nil {
				fmt.Println("Failed to open backup file:", err)
				return
			}

			// -- convert to bytes.Buffer
			backupBuf := bytes.NewBuffer([]byte{})

			_, err = backupBuf.ReadFrom(backupFile)

			if err != nil {
				fmt.Printf("Failed to read backup file file [%s] [error=%s]\n", table, err)
				return
			}

			// Add to tar file
			err = tarAddBuf(tarWriter, backupBuf, "backup/"+table)

			if err != nil {
				fmt.Println("Failed to write backup file to tar file:", err)
				return
			}
		}

		// Close tar file
		tarWriter.Close()

		compressed, err := os.Create("work/seed.iblseed")

		if err != nil {
			fmt.Println("Failed to create compressed file:", err)
			return
		}

		defer compressed.Close()

		// Compress
		w := lzw.NewWriter(compressed, lzw.LSB, 8)

		_, err = io.Copy(w, tarFile)

		if err != nil {
			fmt.Println("ERROR: Failed to compress file:", err)
			return
		}

		w.Close()

		// Generate schema for CI
		pool, err := pgxpool.Connect(context.Background(), "postgres:///infinity")

		if err != nil {
			fmt.Println("ERROR: Failed to get pool:", err)
			return
		}

		schema, err := dbparser.GetSchema(context.Background(), pool)

		if err != nil {
			fmt.Println("ERROR: Failed to get schema for CI etc.:", err)
			return
		}

		// Dump schema to JSON file named "seed-ci.json"
		schemaFile, err = os.Create("work/seed-ci.json")

		if err != nil {
			fmt.Println("ERROR: Failed to create schema file:", err)
			return
		}

		defer schemaFile.Close()

		err = json.NewEncoder(schemaFile).Encode(schema)

		if err != nil {
			fmt.Println("ERROR: Failed to write schema file:", err)
			return
		}

		// Try to find seeds folder (devel assets server)
		path := "/silverpelt/cdn/ibl/dev"
		_, err = os.Stat(path)

		if err == nil {
			fmt.Println("Mpving seed to found folder: " + path)
			err = copyfile.CopyFile("work/seed.iblseed", path+"/seed.iblseed")

			if err != nil {
				fmt.Println("Failed to copy seed to devel assets server:", err)
				return
			}

			err = copyfile.CopyFile("work/seed-ci.json", path+"/seed-ci.json")

			if err != nil {
				fmt.Println("Failed to copy seed to devel assets server:", err)
				return
			}
		}
	},
}

var seedApplyCmd = &cobra.Command{
	Use:     "apply FILENAME",
	Example: "apply latest",
	Short:   "Apply a seed to the database. You must specify either 'latest' or the path to a seed file.",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			fmt.Println("Cleaning up...")

			// delete all files in work directory
			err := os.RemoveAll("work")

			if err != nil {
				fmt.Println("Error cleaning up:", err)
			}
		}()

		// create a work directory
		err := os.Mkdir("work", 0755)

		if err != nil {
			fmt.Println("Error creating work directory", err)
			return
		}

		var seedf *os.File

		// Check args as to which file to use
		seedFile := args[0]

		assetsUrl := links.GetCdnURL() + "/dev"

		if seedFile == "latest" {
			// Download seedfile with progress bar
			data, err := downloader.DownloadFileWithProgress(assetsUrl + "/seed.iblseed?n=" + crypto.RandString(12))

			if err != nil {
				fmt.Println("Failed to download seed file:", err)
				return
			}

			// Write seedfile to disk as temp.iblseed
			f, err := os.Create("work/temp.iblseed")

			if err != nil {
				fmt.Println("Failed to create temp file:", err)
				return
			}

			defer f.Close()

			_, err = f.Write(data)

			if err != nil {
				fmt.Println("Failed to download seed file:", err)
				return
			}

			seedFile = "work/temp.iblseed"
		}

		// Open seed file
		seedf, err = os.Open(seedFile)

		if err != nil {
			fmt.Println("Failed to open seed file:", err)
			return
		}

		// Extract seed file using lzw to buffer
		tarBuf := bytes.NewBuffer([]byte{})
		r := lzw.NewReader(seedf, lzw.LSB, 8)

		_, err = io.Copy(tarBuf, r)

		if err != nil {
			fmt.Println("Failed to decompress seed file:", err)
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

		fmt.Println("Got map keys:", mapKeys(files))

		// Extract out meta
		mdBuf, ok := files["meta"]

		if !ok {
			fmt.Println("Seed file is corrupt [no meta]")
			return
		}

		var md Meta

		err = json.Unmarshal(mdBuf.Bytes(), &md)

		if err != nil {
			fmt.Println("Failed to unmarshal meta:", err)
			return
		}

		if md.SeedVer != seedApiVer {
			fmt.Println("Seed file is of an invalid version [version is", md.SeedVer, "but expected", seedApiVer, "]")
			return
		}

		// Unpack schema to temp file
		schemaBuf, ok := files["schema"]

		if !ok {
			fmt.Println("Seed file is corrupt [no schema]")
			return
		}

		schemaFile, err := os.Create("work/temp.sql")

		if err != nil {
			fmt.Println("Failed to create temp file:", err)
			return
		}

		defer schemaFile.Close()

		_, err = schemaFile.Write(schemaBuf.Bytes())

		if err != nil {
			fmt.Println("Failed to write temp file:", err)
			return
		}

		// Ensure PGDATABASE is NOT set
		os.Unsetenv("PGDATABASE")

		pool, err := pgxpool.Connect(context.Background(), "")

		if err != nil {
			fmt.Println("Failed to acquire database pool:", err)
			return
		}

		// Check if a infinity database already exists
		var exists bool

		err = pool.QueryRow(context.Background(), "SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = 'infinity')").Scan(&exists)

		if err != nil {
			fmt.Println("Failed to check if infinity database exists:", err)
			return
		}

		if exists {
			// Check seed_info table for nonce
			iblPool, err := pgxpool.Connect(context.Background(), "postgres:///infinity")

			if err != nil {
				fmt.Println("Failed to acquire iblPool:", err, "Ignoring...")
			} else {

				var nonce string

				err = iblPool.QueryRow(context.Background(), "SELECT nonce FROM seed_info").Scan(&nonce)

				if err != nil {
					fmt.Println("Failed to check seed_info table:", err, ". Ignoring...")
				} else {
					if nonce == md.Nonce {
						fmt.Println("You are on the latest seed already!")
						return
					}
				}
			}

			iblPool.Close()
		}

		// Create role root
		pool.Exec(context.Background(), "CREATE ROLE postgres")
		pool.Exec(context.Background(), "CREATE ROLE root")

		pool.Exec(context.Background(), "DROP DATABASE infinity")

		pool.Exec(context.Background(), "CREATE DATABASE infinity")

		fmt.Println("Restoring database schema")

		pool.Close()

		// Use pg_restore to restore seedman.tmp
		restoreCmd := exec.Command("pg_restore", "-d", "infinity", "-h", "localhost", "-p", "5432", "work/temp.sql")

		restoreCmd.Stdout = os.Stdout
		restoreCmd.Stderr = os.Stderr

		err = restoreCmd.Run()

		if err != nil {
			fmt.Println("Failed to restore database backup with error:", err)
			return
		}

		fmt.Println("Restoring backed up tables")

		// Restore backed up tables
		var tables []string

		for key := range files {
			if strings.HasPrefix(key, "backup/") {
				tables = append(tables, key[7:])
			}
		}

		for i, table := range tables {
			fmt.Printf("Restoring table: [%d/%d] %s\n", i+1, len(tables), table)

			backupBuf, ok := files["backup/"+table]

			if !ok {
				fmt.Println("Failed to find backup for table", table)
				return
			}

			backupFile, err := os.Create("work/temp-" + table + ".sql")

			if err != nil {
				fmt.Println("Failed to create temp file:", err)
				return
			}

			defer backupFile.Close()

			_, err = backupFile.Write(backupBuf.Bytes())

			if err != nil {
				fmt.Println("Failed to write temp file:", err)
				return
			}

			// Use pg_restore to restore file
			restoreCmd = exec.Command("pg_restore", "-d", "infinity", "-h", "localhost", "-p", "5432", "work/temp-"+table+".sql")

			restoreCmd.Stdout = os.Stdout
			restoreCmd.Stderr = os.Stderr

			err = restoreCmd.Run()

			if err != nil {
				fmt.Println("Failed to restore database backup with error:", err)
				return
			}
		}

		os.Setenv("PGDATABASE", "infinity")

		pool, err = pgxpool.Connect(context.Background(), "postgres:///infinity")

		if err != nil {
			fmt.Println("Failed to acquire database pool for newly created database:", err)
			return
		}

		_, err = pool.Exec(context.Background(), "CREATE TABLE seed_info (nonce TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL)")

		if err != nil {
			fmt.Println("Failed to create seed_info table:", err)
			return
		}

		_, err = pool.Exec(context.Background(), "INSERT INTO seed_info (nonce, created_at) VALUES ($1, $2)", md.Nonce, md.CreatedAt)

		if err != nil {
			fmt.Println("Failed to insert seed info:", err)
			return
		}
	},
}

// copyDb represents the copydb command
var copyDb = &cobra.Command{
	Use:   "copydb TO",
	Short: "Copies the database from 'olympia' to current server. User must currently be on 'olympia'",
	Long:  `Add an experiment to a user`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			fmt.Println("Cleaning up...")

			// delete all files in work directory
			err := os.RemoveAll("work")

			if err != nil {
				fmt.Println("Error cleaning up:", err)
			}
		}()

		// create a work directory
		err := os.Mkdir("work", 0755)

		if err != nil {
			fmt.Println("Error creating work directory:", err)
			return
		}

		fmt.Println("Creating database backup as schema.sql")

		backupCmd := exec.Command("pg_dump", "-Fc", "-d", "infinity", "-f", "work/schema.sql")

		backupCmd.Env = dbcommon.CreateEnv()

		err = backupCmd.Run()

		if err != nil {
			fmt.Println("Error when creating db backup", err)
			return
		}

		fmt.Println("Copying file to target server")

		cpCmd := exec.Command("scp", "work/schema.sql", fmt.Sprintf("root@%s:/tmp/schema.sql", args[0]))
		cpCmd.Env = os.Environ()
		err = cpCmd.Run()

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("Restoring database on target server")

		cmds := [][]string{
			{
				"psql", "-c", "'DROP DATABASE IF EXISTS infinity_bak'",
			},
			{
				"psql", "-c", "'CREATE DATABASE infinity_bak'",
			},
			{
				"psql", "-d", "infinity_bak", "-c", "'CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"'",
			},
			{
				"psql", "-d", "infinity_bak", "-c", "'CREATE EXTENSION IF NOT EXISTS \"citext\"'",
			},
			{
				"pg_restore", "-d", "infinity_bak", "/tmp/schema.sql",
			},
			{
				"psql", "-d", "infinity_bak", "-c", "'DELETE FROM webhooks'",
			},
			{
				"psql", "-d", "infinity_bak", "-c", "'UPDATE users SET api_token = uuid_generate_v4()::text'",
			},
			{
				"psql", "-d", "infinity_bak", "-c", "'UPDATE bots SET api_token = uuid_generate_v4()::text'",
			},
			{
				"psql", "-d", "infinity_bak", "-c", "'UPDATE servers SET api_token = uuid_generate_v4()::text'",
			},
			{
				"psql", "-d", "infinity", "-c", "'DROP DATABASE IF EXISTS infinity_old'",
			},
			{
				"psql", "-d", "infinity_bak", "-c", "'ALTER DATABASE infinity RENAME TO infinity_old'",
			},
			{
				"psql", "-c", "'ALTER DATABASE infinity_bak RENAME TO infinity'",
			},
			{
				"psql", "-d", "infinity", "-c", "'DROP DATABASE IF EXISTS infinity_old'",
			},
			{
				"pg_dump", "-Fc", "-d", "infinity", "-f", "/tmp/prod.sql",
			},
			{
				"psql", "-c", "'DROP DATABASE IF EXISTS infinity__prodmarker'",
			},
			{
				"psql", "-c", "'CREATE DATABASE infinity__prodmarker'",
			},
			{
				"psql", "-d", "infinity__prodmarker", "-c", "'CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"'",
			},
			{
				"psql", "-d", "infinity__prodmarker", "-c", "'CREATE EXTENSION IF NOT EXISTS \"citext\"'",
			},
			{
				"pg_restore", "-d", "infinity__prodmarker", "/tmp/prod.sql",
			},
			{
				"rm", "/tmp/prod.sql", "/tmp/schema.sql",
			},
		}

		for _, c := range cmds {
			fmt.Println("=>", strings.Join(c, " "))

			cmd := exec.Command("ssh", args[0], strings.Join(c, " "))

			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Env = os.Environ()

			err = cmd.Run()

			if err != nil {
				fmt.Println(err)
				return
			}
		}
	},
}

// dbCmd represents the db command
var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "DB operations",
	Long:  `DB operations`,
}

func init() {
	if devmode.DevMode().Allows(types.DevModeFull) {
		dbCmd.AddCommand(copyDb)
		seedCmd.AddCommand(seedNewCmd)
	}

	seedCmd.AddCommand(seedApplyCmd)
	dbCmd.AddCommand(seedCmd)

	rootCmd.AddCommand(dbCmd)
}
