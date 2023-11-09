package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/InfinityBotList/ibl/internal/agents/dbparser"
	"github.com/InfinityBotList/ibl/internal/downloader"
	"github.com/InfinityBotList/ibl/internal/links"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/infinitybotlist/iblfile"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/spf13/cobra"
)

type SeedMetadata struct {
	// Seed Nonce
	Nonce string `json:"n"`

	// Default database name
	DefaultDatabase string `json:"d"`

	// Source database name
	SourceDatabase string `json:"s"`

	// Restore table order
	RestoreOrder []string `json:"r"`
}

// Extensions needed. If a git repo is provided under the extensions key,
// there will be an attempt to install the extensions from the git repo
//
// If a git repo is provided, the following will be run:
//
// - gmake
// - gmake install
// - gmake installcheck
type Extension struct {
	// The name of the extension
	Name string `json:"name"`

	// Git URL if any
	GitUrl string `json:"git,omitempty"`
}

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:   "new <type> <output>",
	Short: "Creates a new database file. One of seed/backup/staging",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fileType := args[0]

		if os.Getenv("ALLOW_ROOT") != "true" {
			// Check if user is root
			if os.Geteuid() == 0 {
				fmt.Println("You must not run this command as root!")
				os.Exit(1)
			}
		}

		// Write metadata to buffer
		mdBuf := bytes.NewBuffer([]byte{})

		// Write metadata to md file
		var metadata iblfile.Meta

		// Create a tar file as a io.Writer, NOT a file
		file := iblfile.New()

		parseExtensions := func() []Extension {
			extensions := []Extension{}

			extensionStr := cmd.Flag("extensions").Value.String()

			if extensionStr == "" {
				return extensions
			}

			extensionStrs := strings.Split(extensionStr, "|")

			for _, ext := range extensionStrs {
				extParts := strings.Split(ext, ",")

				switch len(extParts) {
				case 1:
					extensions = append(extensions, Extension{
						Name: extParts[0],
					})
				case 2:
					extensions = append(extensions, Extension{
						Name:   extParts[0],
						GitUrl: extParts[1],
					})
				default:
					fmt.Println("ERROR: Invalid extension format:", ext)
				}
			}

			return extensions
		}

		switch fileType {
		case "backup":
			dbName := cmd.Flag("db").Value.String()

			if dbName == "" {
				fmt.Println("ERROR: You must specify a database to backup!")
				os.Exit(1)
			}

			pubKeyFile := cmd.Flag("pubkey").Value.String()

			if pubKeyFile == "" {
				fmt.Println("ERROR: You must specify a public key to encrypt the seed with!")
				os.Exit(1)
			}

			pubKeyFileContents, err := os.ReadFile(pubKeyFile)

			if err != nil {
				fmt.Println("ERROR: Failed to read public key file:", err)
				os.Exit(1)
			}

			encMap, encDataMap, err := iblfile.EncryptSections(
				iblfile.DataEncrypt{
					Section: "data",
					Data: func() (*bytes.Buffer, error) {
						// Create full backup of the database
						var backupBuf = bytes.NewBuffer([]byte{})
						backupCmd := exec.Command("pg_dump", "-Fc", "-d", dbName)
						backupCmd.Env = os.Environ()
						backupCmd.Stdout = backupBuf

						err = backupCmd.Run()

						if err != nil {
							return nil, err
						}

						fmt.Println("NOTE: Created", backupBuf.Len(), "byte backup file")

						return backupBuf, nil
					},
					Pubkey: pubKeyFileContents,
				},
			)

			if err != nil {
				fmt.Println("ERROR: Failed to encrypt data:", err)
				os.Exit(1)
			}

			metadata = iblfile.Meta{
				EncryptionData: encDataMap,
			}

			for sectionName, encData := range encMap {
				err = file.WriteSection(encData, sectionName)

				if err != nil {
					fmt.Println("ERROR: Failed to write section", sectionName, "to tar file:", err)
					os.Exit(1)
				}
			}

			extensions := parseExtensions()

			if len(extensions) > 0 {
				var buf bytes.Buffer

				err = json.NewEncoder(&buf).Encode(extensions)

				if err != nil {
					fmt.Println("ERROR: Failed to marshal extensions:", err)
					os.Exit(1)
				}

				file.WriteSection(&buf, "extensionsNeeded")
			}
		case "seed":
			dbName := cmd.Flag("db").Value.String()

			if dbName == "" {
				fmt.Println("ERROR: You must specify a database to seed from!")
				os.Exit(1)
			}

			defaultDatabase := cmd.Flag("default-db").Value.String()

			if defaultDatabase == "" {
				fmt.Println("NOTE: No default database specified, will use database name as default")
				defaultDatabase = dbName
			}

			fmt.Println("Creating database backup in schema buffer")

			var schemaBuf = bytes.NewBuffer([]byte{})
			backupCmd := exec.Command("pg_dump", "-Fc", "--schema-only", "--no-owner", "-d", dbName)
			backupCmd.Env = os.Environ()
			backupCmd.Stdout = schemaBuf

			err := backupCmd.Run()

			if err != nil {
				fmt.Println("ERROR: Failed to create schema backup:", err)
				os.Exit(1)
			}

			// Write metadata buf to tar file
			err = file.WriteSection(schemaBuf, "schema")

			if err != nil {
				fmt.Println("ERROR: Failed to write schema to tar file:", err)
				os.Exit(1)
			}

			// Create backup of some core tables
			var coreTables []string
			backupTables := cmd.Flag("backup-tables").Value.String()

			if backupTables != "" {
				coreTables = strings.Split(backupTables, ",")

				for i := range coreTables {
					coreTables[i] = strings.TrimSpace(coreTables[i])
				}
			}

			for i, table := range coreTables {
				fmt.Printf("Backing up table: [%d/%d] %s\n", i+1, len(coreTables), table)

				// Create backup using pg_dump
				var backupBuf = bytes.NewBuffer([]byte{})
				backupCmd = exec.Command("pg_dump", "-Fc", "-d", dbName, "--data-only", "-t", table)

				backupCmd.Env = os.Environ()
				backupCmd.Stdout = backupBuf

				err = backupCmd.Run()

				if err != nil {
					fmt.Println("ERROR: Failed to create backup:", err)
					os.Exit(1)
				}

				// Add to tar file
				err = file.WriteSection(backupBuf, "backup/"+table)

				if err != nil {
					fmt.Println("ERROR: Failed to write backup file to tar file:", err)
					os.Exit(1)
				}
			}

			// Create seed meta file
			seedMeta := SeedMetadata{
				Nonce:           crypto.RandString(32),
				DefaultDatabase: defaultDatabase,
				SourceDatabase:  dbName,
				RestoreOrder:    coreTables,
			}

			seedMetaBuf := bytes.NewBuffer([]byte{})
			enc := json.NewEncoder(seedMetaBuf)
			err = enc.Encode(seedMeta)

			if err != nil {
				fmt.Println("ERROR: Failed to marshal seed meta:", err)
				os.Exit(1)
			}

			// Write metadata buf to tar file
			err = file.WriteSection(seedMetaBuf, "seed_meta")

			if err != nil {
				fmt.Println("ERROR: Failed to write seed-specific meta to tar file:", err)
				os.Exit(1)
			}

			extensions := parseExtensions()

			if len(extensions) > 0 {
				var buf bytes.Buffer

				err = json.NewEncoder(&buf).Encode(extensions)

				if err != nil {
					fmt.Println("ERROR: Failed to marshal extensions:", err)
					os.Exit(1)
				}

				file.WriteSection(&buf, "extensionsNeeded")
			}
		case "staging":
			dbName := cmd.Flag("db").Value.String()
			extensions := parseExtensions()

			if dbName == "" {
				fmt.Println("ERROR: You must specify a database to backup!")
				os.Exit(1)
			}

			createSanitizedDb := func() (*bytes.Buffer, error) {
				ctx := context.Background()

				sanitizeCode := map[string]func(conn *pgx.Conn) error{
					"infinity": func(conn *pgx.Conn) error {
						sqlCmds := []string{
							"DELETE FROM webhooks",
							"UPDATE users SET api_token = uuid_generate_v4()::text",
							"UPDATE bots SET api_token = uuid_generate_v4()::text",
							"UPDATE servers SET api_token = uuid_generate_v4()::text",
						}

						for _, c := range sqlCmds {
							fmt.Println("[psql, copyDb] =>", c)

							_, err := conn.Exec(ctx, c)

							if err != nil {
								return fmt.Errorf("failed to execute sql command: %w", err)
							}
						}

						return nil
					},
				}

				fmt.Println("NOTE: Creating unsanitized database backup in memory")

				var buf = bytes.NewBuffer([]byte{})
				backupCmd := exec.Command("pg_dump", "-Fc", "-d", dbName)
				backupCmd.Env = os.Environ()
				backupCmd.Stdout = buf

				err := backupCmd.Run()

				if err != nil {
					return nil, fmt.Errorf("failed to create db backup: %w", err)
				}

				if buf.Len() == 0 {
					return nil, fmt.Errorf("database backup is empty")
				}

				sanitizer, ok := sanitizeCode[dbName]

				if !ok {
					fmt.Println("WARNING: No sanitization task for database", dbName)
					return buf, nil
				}

				// Make copy (__dbcopy) using created db backup on source server
				fmt.Println("Creating copy of database on source server with name '" + dbName + "__dbcopy'")

				copyDbName := dbName + "__dbcopy"

				conn, err := pgx.Connect(ctx, "postgres:///"+dbName)

				if err != nil {
					return nil, fmt.Errorf("failed to acquire database conn: %w", err)
				}

				sqlCmds := []string{
					"DROP DATABASE IF EXISTS " + copyDbName,
					"CREATE DATABASE " + copyDbName,
				}

				for _, c := range sqlCmds {
					fmt.Println("[psql, origDb] =>", c)
					_, err = conn.Exec(ctx, c)

					if err != nil {
						return nil, fmt.Errorf("failed to execute sql command: %w", err)
					}
				}

				err = conn.Close(ctx)

				if err != nil {
					fmt.Println("WARNING: Failed to close conn:", err)
				}

				conn, err = pgx.Connect(ctx, "postgres:///"+copyDbName)

				if err != nil {
					return nil, fmt.Errorf("failed to acquire copy database conn: %w", err)
				}

				for _, ext := range extensions {
					c := "CREATE EXTENSION IF NOT EXISTS \"" + ext.Name + "\""
					fmt.Println("[psql, addExtension]", c)

					_, err = conn.Exec(ctx, c)

					if err != nil {
						return nil, fmt.Errorf("failed to execute sql command: %w", err)
					}
				}

				err = conn.Close(ctx)

				if err != nil {
					fmt.Println("WARNING: Failed to close conn:", err)
				}

				restoreCmd := exec.Command("pg_restore", "-d", copyDbName)
				restoreCmd.Env = os.Environ()
				restoreCmd.Stdout = os.Stdout
				restoreCmd.Stderr = os.Stderr
				restoreCmd.Stdin = buf

				err = restoreCmd.Run()

				if err != nil {
					return nil, fmt.Errorf("failed to restore db backup: %w", err)
				}

				defer func() {
					cleanup := func() error {
						// Delete copy (__dbcopy) on source server
						fmt.Println("CLEANUP: Deleting copy of database on source server with name '" + copyDbName + "'")

						conn, err = pgx.Connect(ctx, "postgres:///"+dbName)

						if err != nil {
							return fmt.Errorf("failed to acquire database conn: %w", err)
						}

						_, err = conn.Exec(ctx, "DROP DATABASE IF EXISTS "+copyDbName)

						if err != nil {
							return fmt.Errorf("failed to drop copy database: %w", err)
						}

						err = conn.Close(ctx)

						if err != nil {
							fmt.Println("WARNING: Failed to close conn:", err)
						}

						return nil
					}

					err := cleanup()

					if err != nil {
						fmt.Println(err)
						fmt.Println("FATAL: Cleanup task to delete '"+copyDbName+"' has failed! Please do so manually.\nError:", err)
						return
					}
				}()

				fmt.Println("Sanitizing copied database")

				conn, err = pgx.Connect(ctx, "postgres:///"+copyDbName)

				if err != nil {
					return nil, fmt.Errorf("failed to acquire copy database conn: %w", err)
				}

				err = sanitizer(conn)

				if err != nil {
					return nil, fmt.Errorf("failed to sanitize database: %w", err)
				}

				err = conn.Close(ctx)

				if err != nil {
					fmt.Println("WARNING: Failed to close conn:", err)
				}

				fmt.Println("NOTE: Creating sanitized database backup in memory")

				var sanitizedBuf = bytes.NewBuffer([]byte{})
				backupCmd = exec.Command("pg_dump", "-Fc", "-d", copyDbName)
				backupCmd.Env = os.Environ()
				backupCmd.Stdout = sanitizedBuf

				err = backupCmd.Run()

				if err != nil {
					return nil, fmt.Errorf("failed to create db backup: %w", err)
				}

				if sanitizedBuf.Len() == 0 {
					return nil, fmt.Errorf("sanitized database backup is empty")
				}

				return sanitizedBuf, nil
			}

			pubKeyFile := cmd.Flag("pubkey").Value.String()

			if pubKeyFile != "" {
				pubKeyFileContents, err := os.ReadFile(pubKeyFile)

				if err != nil {
					fmt.Println("ERROR: Failed to read specified public key file:", err)
					os.Exit(1)
				}

				encMap, encDataMap, err := iblfile.EncryptSections(
					iblfile.DataEncrypt{
						Section: "data",
						Data:    createSanitizedDb,
						Pubkey:  pubKeyFileContents,
					},
				)

				if err != nil {
					fmt.Println("ERROR: Failed to encrypt data:", err)
					os.Exit(1)
				}

				metadata = iblfile.Meta{
					EncryptionData: encDataMap,
				}

				for sectionName, encData := range encMap {
					err = file.WriteSection(encData, sectionName)

					if err != nil {
						fmt.Println("ERROR: Failed to write section", sectionName, "to tar file:", err)
						os.Exit(1)
					}
				}
			} else {
				fmt.Println("NOTE: No public key specified, will not encrypt database backup")

				backupBuf, err := createSanitizedDb()

				if err != nil {
					fmt.Println("ERROR: Failed to create sanitized database backup:", err)
					os.Exit(1)
				}

				err = file.WriteSection(backupBuf, "data")

				if err != nil {
					fmt.Println("ERROR: Failed to write database backup to tar file:", err)
					os.Exit(1)
				}
			}

			if len(extensions) > 0 {
				var buf bytes.Buffer

				err := json.NewEncoder(&buf).Encode(extensions)

				if err != nil {
					fmt.Println("ERROR: Failed to marshal extensions:", err)
					os.Exit(1)
				}

				file.WriteSection(&buf, "extensionsNeeded")
			}

		default:
			fmt.Println("ERROR: Invalid type:", fileType)
			os.Exit(1)
		}

		fileType = "db." + fileType

		metadata.CreatedAt = time.Now()
		metadata.Protocol = iblfile.Protocol
		metadata.Type = fileType

		f, err := iblfile.GetFormat(fileType)

		if f == nil {
			fmt.Println("ERROR: Internal error: format is not registered:", fileType, err)
			os.Exit(1)
		}

		metadata.FormatVersion = f.Version

		enc := json.NewEncoder(mdBuf)

		err = enc.Encode(metadata)

		if err != nil {
			fmt.Println("ERROR: Failed to write metadata:", err)
			os.Exit(1)
		}

		// Write metadata buf to tar file
		err = file.WriteSection(mdBuf, "meta")

		if err != nil {
			fmt.Println("ERROR: Failed to write metadata to tar file:", err)
			os.Exit(1)
		}

		compressed, err := os.Create(args[1])

		if err != nil {
			fmt.Println("ERROR: Failed to create compressed file:", err)
			os.Exit(1)
		}

		defer compressed.Close()

		err = file.WriteOutput(compressed)

		if err != nil {
			fmt.Println("ERROR: Failed to write file:", err)
			os.Exit(1)
		}
	},
}

var loadCmd = &cobra.Command{
	Use:     "load FILENAME",
	Example: "load latestseed/<backup file>/<seed file>",
	Short:   "Loads a file to the database. You must specify either 'latestseed' or the path to a loadable db file",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if os.Getenv("ALLOW_ROOT") != "true" {
			// Check if user is root
			if os.Geteuid() == 0 {
				fmt.Println("ERROR: You must not run this command as root!")
				os.Exit(1)
			}
		}

		var data io.Reader

		// Check args as to which file to use
		filename := args[0]

		assetsUrl := links.GetCdnURL() + "/dev"

		if filename == "latestseed" {
			// Download seedfile with progress bar
			var err error
			var buf []byte
			buf, err = downloader.DownloadFileWithProgress(assetsUrl + "/seed.iblseed?n=" + crypto.RandString(12))

			if err != nil {
				fmt.Println("ERROR: Failed to download seed file:", err)
				os.Exit(1)
			}

			data = bytes.NewBuffer(buf)
		} else {
			// Open seed file
			f, err := os.Open(filename)

			if err != nil {
				fmt.Println("ERROR: Failed to open seed file:", err)
				os.Exit(1)
			}

			defer f.Close()

			data = f
		}

		sections, meta, err := iblfile.ParseData(data)

		if err != nil {
			fmt.Println("ERROR: Parsing data failed", err)
			os.Exit(1)
		}

		if meta == nil {
			fmt.Println("ERROR: No metadata present!")
			os.Exit(1)
		}

		if meta.Protocol != iblfile.Protocol && os.Getenv("SKIP_PROTOCOL_CHECK") != "true" {
			fmt.Println("ERROR: File is of an invalid version [version is", meta.Protocol, "but expected", iblfile.Protocol, "]")
			os.Exit(1)
		}

		tryHandlingExtensions := func(sections map[string]*bytes.Buffer, meta *iblfile.Meta, dbName string) error {
			extSection, ok := sections["extensionsNeeded"]

			if !ok {
				// No extensions needed
				return nil
			}

			var extensions []Extension

			err := json.NewDecoder(extSection).Decode(&extensions)

			if err != nil {
				return fmt.Errorf("failed to decode extensions: %w", err)
			}

			conn, err := pgx.Connect(context.Background(), "postgres:///"+dbName)

			if err != nil {
				return fmt.Errorf("failed to acquire database conn: %w", err)
			}

			for _, ext := range extensions {
				// Check if extension exists on postgres
				_, err = conn.Exec(context.Background(), "CREATE EXTENSION IF NOT EXISTS \""+ext.Name+"\"")

				if err == nil {
					continue
				}

				if strings.Contains(err.Error(), "not available") {
					fmt.Println("ERROR: Extension", ext.Name, "cannot be loaded:", err)
					fmt.Println("Trying to install it from the git repo:", ext.GitUrl)

					if os.Getenv("SKIP_EXTENSION_INSTALL") == "true" {
						os.Exit(1)
					}

					gitCmd := exec.Command("git", "clone", ext.GitUrl, ext.Name)

					gitCmd.Stdout = os.Stdout
					gitCmd.Stderr = os.Stderr
					gitCmd.Env = os.Environ()

					err = gitCmd.Run()

					if err != nil {
						return fmt.Errorf("failed to clone git repo: %w", err)
					}

					var cmds = [][]string{
						{"gmake"},
						{"gmake", "install"},
						{"gmake", "installcheck"},
					}

					pwd, err := os.Getwd()

					if err != nil {
						return fmt.Errorf("failed to get pwd: %w", err)
					}

					for _, c := range cmds {
						cmd := exec.Command(c[0], c[1:]...)

						cmd.Dir = pwd + "/" + ext.Name
						cmd.Stdout = os.Stdout
						cmd.Stderr = os.Stderr
						cmd.Env = os.Environ()

						err = cmd.Run()

						if err != nil {
							return fmt.Errorf("failed to execute command '%s': %w", c, err)
						}
					}
				}

				if err != nil {
					return fmt.Errorf("failed to create extension: %w", err)
				}
			}

			return nil
		}

		switch meta.Type {
		case "db.backup":
			privKeyFile := cmd.Flag("priv-key").Value.String()

			if privKeyFile == "" {
				fmt.Println("ERROR: You must specify a private key to decrypt the seed with!")
				os.Exit(1)
			}

			dbName := cmd.Flag("db").Value.String()

			if dbName == "" {
				fmt.Println("ERROR: You must specify a database to restore the backup to!")
				os.Exit(1)
			}

			privKeyFileContents, err := os.ReadFile(privKeyFile)

			if err != nil {
				fmt.Println("ERROR: Failed to read private key file:", err)
				os.Exit(1)
			}

			encData, ok := sections["data"]

			if !ok {
				fmt.Println("ERROR: DB file is corrupt [no backup data]")
				os.Exit(1)
			}

			enc, ok := meta.EncryptionData["data"]

			var decrData *bytes.Buffer
			if ok {
				decrData, err = iblfile.DecryptData(encData, enc, privKeyFileContents)

				if err != nil {
					fmt.Println("ERROR: Failed to decrypt data:", err)
					os.Exit(1)
				}
			} else {
				fmt.Println("WARNING: Backup data is not encrypted!")
				decrData = encData
			}

			err = tryHandlingExtensions(sections, meta, dbName)

			if err != nil {
				fmt.Println("ERROR: Failed to handle extensions:", err)
				os.Exit(1)
			}

			// Restore dump
			backupCmd := exec.Command("pg_restore", "-d", dbName)

			backupCmd.Stdout = os.Stdout
			backupCmd.Stderr = os.Stderr
			backupCmd.Env = os.Environ()
			backupCmd.Stdin = decrData

			err = backupCmd.Run()

			if err != nil {
				fmt.Println("ERROR: Failed to restore database backup with error:", err)
				os.Exit(1)
			}

			fmt.Println("NOTE: Backup restored successfully!")
		case "db.seed":
			dbName := cmd.Flag("db").Value.String()

			// Load seed metadata
			var smeta SeedMetadata

			seedMetaBuf, ok := sections["seed_meta"]

			if !ok {
				fmt.Println("ERROR: Seed file is corrupt [no seed meta]")
				os.Exit(1)
			}

			err = json.NewDecoder(seedMetaBuf).Decode(&smeta)

			if err != nil {
				fmt.Println("ERROR: Seed file is corrupt [invalid seed meta]")
				os.Exit(1)
			}

			if dbName == "" {
				if smeta.DefaultDatabase == "" {
					fmt.Println("ERROR: No default database name is specified in this seed. You must specify a database to restore the seed to using the --db argument")
					os.Exit(1)
				} else {
					dbName = smeta.DefaultDatabase
				}
			}

			// Unpack schema to temp file
			schema, ok := sections["schema"]

			if !ok {
				fmt.Println("ERROR: Seed file is corrupt [no schema]")
				os.Exit(1)
			}

			os.Unsetenv("PGDATABASE")

			ctx := context.Background()

			conn, err := pgx.Connect(ctx, "")

			if err != nil {
				fmt.Println("ERROR: Failed to acquire database conn:", err)
				os.Exit(1)
			}

			// Check if a database already exists
			var exists bool

			err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = $1)", dbName).Scan(&exists)

			if err != nil {
				fmt.Println("ERROR: Failed to check if database exists:", err)
				os.Exit(1)
			}

			if exists {
				// Check seed_info table for nonce
				iconn, err := pgx.Connect(ctx, "postgres:///"+dbName)

				if err != nil {
					fmt.Println("ERROR: Failed to acquire iconn:", err, "Ignoring...")
				} else {
					var nonce string

					err = iconn.QueryRow(ctx, "SELECT nonce FROM seed_info").Scan(&nonce)

					if err != nil {
						fmt.Println("ERROR: Failed to check seed_info table:", err, ". Ignoring...")
					} else {
						if nonce == smeta.Nonce {
							fmt.Print("\n\nYou are on the latest seed already!")
							os.Exit(0)
						}
					}
				}

				iconn.Close(ctx)
			}

			// Create role root
			conn.Exec(ctx, "CREATE ROLE postgres")
			conn.Exec(ctx, "CREATE ROLE root")
			conn.Exec(ctx, "DROP DATABASE "+dbName)
			conn.Exec(ctx, "CREATE DATABASE "+dbName)

			fmt.Println("Restoring database schema")

			conn.Close(ctx)

			err = tryHandlingExtensions(sections, meta, dbName)

			if err != nil {
				fmt.Println("ERROR: Failed to handle extensions:", err)
				os.Exit(1)
			}

			// Use pg_restore to restore seedman.tmp
			restoreCmd := exec.Command("pg_restore", "-d", dbName)
			restoreCmd.Stdout = os.Stdout
			restoreCmd.Stderr = os.Stderr
			restoreCmd.Stdin = schema
			restoreCmd.Env = os.Environ()
			err = restoreCmd.Run()

			if err != nil {
				fmt.Println("ERROR: Failed to restore database backup with error:", err)
				os.Exit(1)
			}

			fmt.Println("Restoring backed up tables")

			for i, table := range smeta.RestoreOrder {
				fmt.Printf("Restoring table: [%d/%d] %s\n", i+1, len(smeta.RestoreOrder), table)

				backupBuf, ok := sections["backup/"+table]

				if !ok {
					fmt.Println("ERROR: Failed to find backup for table", table)
					os.Exit(1)
				}

				// Use pg_restore to restore file
				restoreCmd = exec.Command("pg_restore", "-d", dbName)

				restoreCmd.Stdout = os.Stdout
				restoreCmd.Stderr = os.Stderr
				restoreCmd.Stdin = backupBuf
				restoreCmd.Env = os.Environ()
				err = restoreCmd.Run()

				if err != nil {
					fmt.Println("ERROR: Failed to restore database backup with error:", err, " for table", table)
					os.Exit(1)
				}
			}

			conn, err = pgx.Connect(ctx, "postgres:///"+dbName)

			if err != nil {
				fmt.Println("ERROR: Failed to acquire database pool for newly created database:", err)
				os.Exit(1)
			}

			_, err = conn.Exec(ctx, "CREATE TABLE seed_info (nonce TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL)")

			if err != nil {
				fmt.Println("ERROR: Failed to create seed_info table:", err)
				os.Exit(1)
			}

			_, err = conn.Exec(ctx, "INSERT INTO seed_info (nonce, created_at) VALUES ($1, $2)", smeta.Nonce, meta.CreatedAt)

			if err != nil {
				fmt.Println("ERROR: Failed to insert seed info:", err)
				os.Exit(1)
			}
		case "db.staging":
			privKeyFile := cmd.Flag("priv-key").Value.String()

			dbName := cmd.Flag("db").Value.String()

			if dbName == "" {
				fmt.Println("ERROR: You must specify a database to restore the backup to!")
				os.Exit(1)
			}

			var privKeyFileContents []byte
			if privKeyFile != "" {
				privKeyFileContents, err = os.ReadFile(privKeyFile)

				if err != nil {
					fmt.Println("ERROR: Failed to read private key file:", err)
					os.Exit(1)
				}
			}

			encData, ok := sections["data"]

			if !ok {
				fmt.Println("ERROR: DB file is corrupt [no backup data]")
				os.Exit(1)
			}

			enc, ok := meta.EncryptionData["data"]

			if ok && privKeyFile == "" {
				fmt.Println("ERROR: This staging backup is encrypted. You must provide a private key to decrypt with!")
				os.Exit(1)
			}

			var decrData *bytes.Buffer
			if ok {
				decrData, err = iblfile.DecryptData(encData, enc, privKeyFileContents)

				if err != nil {
					fmt.Println("ERROR: Failed to decrypt data:", err)
					os.Exit(1)
				}
			} else {
				fmt.Println("WARNING: Backup data is not encrypted!")
				decrData = encData
			}

			ctx := context.Background()

			prodMarkerName := dbName + "__prodmarker"

			sqlCmds := []string{
				"DROP DATABASE IF EXISTS " + dbName,
				"CREATE DATABASE " + dbName,
				"DROP DATABASE IF EXISTS " + prodMarkerName,
				"CREATE DATABASE " + prodMarkerName,
			}

			conn, err := pgx.Connect(ctx, "postgres:///")

			if err != nil {
				fmt.Println("ERROR: Failed to acquire database conn:", err)
				os.Exit(1)
			}

			for _, c := range sqlCmds {
				fmt.Println("[psql, origDb] =>", c)
				_, err = conn.Exec(ctx, c)

				if err != nil {
					fmt.Println("ERROR: Failed to execute sql command:", err)
					os.Exit(1)
				}
			}

			err = conn.Close(ctx)

			if err != nil {
				fmt.Println("WARNING: Failed to close conn:", err)
			}

			decrDataBytes := decrData.Bytes()

			// Restore dump to dbName and prodMarkerName
			backupCmd := exec.Command("pg_restore", "-d", dbName)
			backupCmd.Stdout = os.Stdout
			backupCmd.Stderr = os.Stderr
			backupCmd.Env = os.Environ()
			backupCmd.Stdin = bytes.NewBuffer(decrDataBytes)

			err = backupCmd.Run()

			if err != nil {
				fmt.Println("ERROR: Failed to restore database backup with error:", err)
				os.Exit(1)
			}

			backupCmd = exec.Command("pg_restore", "-d", prodMarkerName)
			backupCmd.Stdout = os.Stdout
			backupCmd.Stderr = os.Stderr
			backupCmd.Env = os.Environ()
			backupCmd.Stdin = bytes.NewBuffer(decrDataBytes)

			err = backupCmd.Run()

			if err != nil {
				fmt.Println("ERROR: Failed to restore database backup to prodmarker with error:", err)
				os.Exit(1)
			}
		default:
			fmt.Println("ERROR: Invalid type:", meta.Type)
			os.Exit(1)
		}
	},
}

var genCiSchemaCmd = &cobra.Command{
	Use:   "gen-ci-schema <path>",
	Short: "Generates a seed-ci.json file for use in CI",
	Long:  "Generates a seed-ci.json file for use in CI",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Generate schema for CI
		ctx := context.Background()
		pool, err := pgxpool.Connect(ctx, "postgres:///infinity")

		if err != nil {
			fmt.Println("ERROR: Failed to get pool:", err)
			os.Exit(1)
		}

		schema, err := dbparser.GetSchema(ctx, pool)

		if err != nil {
			fmt.Println("ERROR: Failed to get schema for CI etc.:", err)
			os.Exit(1)
		}

		schemaFile, err := os.Create(args[0])

		if err != nil {
			fmt.Println("ERROR: Failed to create schema file:", err)
			os.Exit(1)
		}

		defer schemaFile.Close()

		err = json.NewEncoder(schemaFile).Encode(schema)

		if err != nil {
			fmt.Println("ERROR: Failed to write schema file:", err)
			os.Exit(1)
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
	iblfile.RegisterFormat(
		"db",
		&iblfile.Format{
			Format:  "backup",
			Version: "a1",
		},
		&iblfile.Format{
			Format:  "seed",
			Version: "a2",
			GetExtended: func(sections map[string]*bytes.Buffer, meta *iblfile.Meta) (map[string]any, error) {
				seedMetaBuf, ok := sections["seed_meta"]

				if !ok {
					return nil, fmt.Errorf("no seed metadata found")
				}

				var smeta SeedMetadata

				err := json.NewDecoder(seedMetaBuf).Decode(&smeta)

				if err != nil {
					return nil, fmt.Errorf("seed metadata is invalid: %w", err)
				}

				return map[string]any{
					"Nonce":           smeta.Nonce,
					"DefaultDatabase": smeta.DefaultDatabase,
					"SourceDatabase":  smeta.SourceDatabase,
				}, nil
			},
		},
		&iblfile.Format{
			Format:  "staging",
			Version: "a1",
		},
	)

	loadCmd.PersistentFlags().String("priv-key", "", "The private key to decrypt the backup with [backup only]")
	loadCmd.PersistentFlags().String("db", "", "If type is backup, the database to restore the backup to (backup) or the database name to seed to (seed).")

	newCmd.PersistentFlags().String("pubkey", "", "The public key to encrypt the seed with")
	newCmd.PersistentFlags().String("default-db", "", "If type is seed, the default database name to seed to.")
	newCmd.PersistentFlags().String("db", "", "If type is backup, the database to backup to (backup) or the database name to seed from [seed only].")
	newCmd.PersistentFlags().String("backup-tables", "", "The tables to fully backup in the seed [seed only]")
	newCmd.PersistentFlags().String("extensions", "", "The extensions required. Format: --extensions=NAME,GIT_URL|NAME2,GIT_URL2 [seed/backup/staging only]")

	dbCmd.AddCommand(genCiSchemaCmd)
	dbCmd.AddCommand(newCmd)
	dbCmd.AddCommand(loadCmd)
	rootCmd.AddCommand(dbCmd)
}
