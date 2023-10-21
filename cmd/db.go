package cmd

import (
	"archive/tar"
	"bytes"
	"compress/lzw"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/InfinityBotList/ibl/internal/agents/dbparser"
	"github.com/InfinityBotList/ibl/internal/devmode"
	"github.com/InfinityBotList/ibl/internal/downloader"
	"github.com/InfinityBotList/ibl/internal/links"
	"github.com/InfinityBotList/ibl/types"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/spf13/cobra"
)

const protocol = "frostpaw-rev4-e1" // e means encryption protocol version

var formatVersionMap = map[string]string{
	"backup": "a1",
	"seed":   "a1",
}

// The number of keys to encrypt the data with
//
// Note that changing keycount does not need a change in protocol version
const keyCount = 16

type EncryptionData struct {
	// Public key to encrypt data with
	PEM []byte `json:"p"`

	// Encrypted OEAP keys
	Keys [][]byte `json:"k"`

	// Encryption nonce
	Nonce string `json:"n"`
}

type SeedMetadata struct {
	// Seed Nonce
	Nonce string `json:"n"`

	// Default database name
	DefaultDatabase string `json:"d"`

	// Source database name
	SourceDatabase string `json:"s"`
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

	// Type of the db file, either 'backup' or 'seed'
	Type string `json:"t"`
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

func readTarFile(tarBuf io.Reader) map[string]*bytes.Buffer {
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
			return nil
		}

		// Read file into buffer
		buf := bytes.NewBuffer([]byte{})

		_, err = io.Copy(buf, tarReader)

		if err != nil {
			fmt.Println("Failed to read tar file:", err)
			return nil
		}

		// Save file to map
		files[header.Name] = buf
	}

	return files
}

func parseData(data io.Reader) (map[string]*bytes.Buffer, *Meta, error) {
	tarBuf := bytes.NewBuffer([]byte{})
	r := lzw.NewReader(data, lzw.LSB, 8)

	_, err := io.Copy(tarBuf, r)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to decompress seed file: %w", err)
	}

	// Get size of decompressed file
	tarSize := tarBuf.Len()

	fmt.Println("Decompressed size:", tarSize, "bytes")

	files := readTarFile(tarBuf)

	if len(files) == 0 {
		return nil, nil, fmt.Errorf("failed to read tar file")
	}

	fmt.Println("Keys present:", mapKeys(files))
	if meta, ok := files["meta"]; ok {
		var metadata Meta

		err = json.NewDecoder(meta).Decode(&metadata)

		if err != nil {
			fmt.Println("Invalid meta, unmarshal fail:", err)
			return nil, nil, fmt.Errorf("failed to unmarshal meta: %w", err)
		}

		fmt.Println("")
		fmt.Println("== Metadata ==")
		fmt.Println("Protocol:", metadata.Protocol)
		fmt.Println("File Version:", metadata.FormatVersion)
		fmt.Println("Type:", metadata.Type)
		fmt.Println("Created At:", metadata.CreatedAt)

		v, ok := formatVersionMap[metadata.Type]

		if !ok {
			return nil, nil, fmt.Errorf("invalid type: %s", metadata.Type)
		}

		if metadata.FormatVersion != v {
			return nil, nil, fmt.Errorf("this %s uses format version %s, but this version of the tool only supports version %s", metadata.Type, metadata.FormatVersion, v)
		}

		fmt.Println("")

		return files, &metadata, nil
	} else {
		return files, nil, fmt.Errorf("no metadata present")
	}
}

type dataEncrypt struct {
	section string
	data    func() (*bytes.Buffer, error)
	pubkey  []byte
}

func encryptSections(de ...dataEncrypt) (map[string]*bytes.Buffer, map[string]*EncryptionData, error) {
	var dataMap = make(map[string]*bytes.Buffer)
	var encDataMap = make(map[string]*EncryptionData)
	for _, d := range de {
		if len(d.pubkey) == 0 {
			return nil, nil, fmt.Errorf("no public key provided for section %s", d.section)
		}

		if d.section == "" {
			return nil, nil, fmt.Errorf("no section name provided")
		}

		if d.data == nil {
			return nil, nil, fmt.Errorf("no data function provided for section %s", d.section)
		}

		pem, _ := pem.Decode(d.pubkey)

		if pem == nil {
			return nil, nil, fmt.Errorf("failed to decode public key file")
		}

		hash := sha512.New()
		random := rand.Reader

		// Generate a random 32 byte key
		var pub *rsa.PublicKey
		pubInterface, parseErr := x509.ParsePKIXPublicKey(pem.Bytes)

		if parseErr != nil {
			fmt.Println("Failed to parse public key:", parseErr)
			return nil, nil, fmt.Errorf("failed to parse public key: %s", parseErr)
		}

		encNonce := crypto.RandString(128)

		pub = pubInterface.(*rsa.PublicKey)

		var keys [][]byte
		var encPass = []byte(encNonce)
		for i := 0; i < keyCount; i++ {
			msg := crypto.RandString(32)
			key, encryptErr := rsa.EncryptOAEP(hash, random, pub, []byte(msg), nil)

			if encryptErr != nil {
				return nil, nil, fmt.Errorf("failed to encrypt data: %s", encryptErr)
			}

			keys = append(keys, key)
			encPass = append(encPass, msg...)
		}

		// Encrypt backupBuf with encryptedKey using aes-512-gcm
		keyHash := sha256.New()
		keyHash.Write(encPass)

		c, err := aes.NewCipher(keyHash.Sum(nil))

		if err != nil {
			return nil, nil, fmt.Errorf("failed to create cipher: %s", err)
		}

		gcm, err := cipher.NewGCM(c)

		if err != nil {
			return nil, nil, fmt.Errorf("failed to create gcm: %s", err)
		}

		aesNonce := make([]byte, gcm.NonceSize())
		if _, err = io.ReadFull(rand.Reader, aesNonce); err != nil {
			return nil, nil, fmt.Errorf("failed to generate AES nonce: %s", err)
		}

		dataBuf, err := d.data()

		if err != nil {
			return nil, nil, fmt.Errorf("failed to get data: %s", err)
		}

		encData := gcm.Seal(aesNonce, aesNonce, dataBuf.Bytes(), nil)

		encDataMap[d.section] = &EncryptionData{
			PEM:   d.pubkey,
			Keys:  keys,
			Nonce: encNonce,
		}
		dataMap[d.section] = bytes.NewBuffer(encData)
	}

	return dataMap, encDataMap, nil
}

func decryptData(encData *bytes.Buffer, enc *EncryptionData, privkey []byte) (*bytes.Buffer, error) {
	var decrPass = []byte(enc.Nonce)
	for _, key := range enc.Keys {
		hash := sha512.New()
		random := rand.Reader

		pem, _ := pem.Decode(privkey)

		if pem == nil {
			return nil, fmt.Errorf("failed to decode private key file")
		}

		privInterface, parseErr := x509.ParsePKCS8PrivateKey(pem.Bytes)

		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse private key: %s", parseErr)
		}

		priv := privInterface.(*rsa.PrivateKey)
		msg, err := rsa.DecryptOAEP(hash, random, priv, key, nil)

		if err != nil {
			return nil, fmt.Errorf("failed to decrypt data: %s", err)
		}

		decrPass = append(decrPass, msg...)
	}

	// Decrypt backupBuf with encryptedKey using aes-512-gcm
	keyHash := sha256.New()
	keyHash.Write(decrPass)
	c, err := aes.NewCipher(keyHash.Sum(nil))

	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %s", err)
	}

	gcm, err := cipher.NewGCM(c)

	if err != nil {
		return nil, fmt.Errorf("failed to create gcm: %s", err)
	}

	nonceSize := gcm.NonceSize()
	// Extract nonce from encrypted data which is a bytes buffer
	aesNonce := encData.Next(nonceSize)

	if len(aesNonce) != nonceSize {
		return nil, fmt.Errorf("failed to extract nonce from encrypted data: %d != %d", len(aesNonce), nonceSize)
	}

	encData = bytes.NewBuffer(encData.Bytes())

	// Decrypt data
	decData, err := gcm.Open(nil, aesNonce, encData.Bytes(), nil)

	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %s", err)
	}

	return bytes.NewBuffer(decData), nil
}

func mapKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:   "new <type> <output>",
	Short: "Creates a new database file. Either 'seed' or 'backup'",
	Long:  `Creates a new database file. Either 'seed' or 'backup'`,
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
		var metadata Meta

		// Create a tar file as a io.Writer, NOT a file
		tarFile := bytes.NewBuffer([]byte{})
		tarWriter := tar.NewWriter(tarFile)

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

			encMap, encDataMap, err := encryptSections(
				dataEncrypt{
					section: "data",
					data: func() (*bytes.Buffer, error) {
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
					pubkey: pubKeyFileContents,
				},
			)

			if err != nil {
				fmt.Println("ERROR: Failed to encrypt data:", err)
				os.Exit(1)
			}

			metadata = Meta{
				EncryptionData: encDataMap,
			}

			for sectionName, encData := range encMap {
				err = tarAddBuf(tarWriter, encData, sectionName)

				if err != nil {
					fmt.Println("ERROR: Failed to write section", sectionName, "to tar file:", err)
					os.Exit(1)
				}
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

			fmt.Println("Creating database backup as schema.sql")

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
			err = tarAddBuf(tarWriter, schemaBuf, "schema")

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
				err = tarAddBuf(tarWriter, backupBuf, "backup/"+table)

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
			}

			seedMetaBuf := bytes.NewBuffer([]byte{})
			enc := json.NewEncoder(seedMetaBuf)
			err = enc.Encode(seedMeta)

			if err != nil {
				fmt.Println("ERROR: Failed to marshal seed meta:", err)
				os.Exit(1)
			}

			// Write metadata buf to tar file
			err = tarAddBuf(tarWriter, seedMetaBuf, "seed_meta")

			if err != nil {
				fmt.Println("ERROR: Failed to write seed-specific meta to tar file:", err)
				os.Exit(1)
			}
		default:
			fmt.Println("ERROR: Invalid type:", fileType)
			os.Exit(1)
		}

		metadata.CreatedAt = time.Now()
		metadata.Protocol = protocol
		metadata.Type = fileType

		v, ok := formatVersionMap[fileType]

		if !ok {
			fmt.Println("ERROR: Internal error: format has no version", fileType)
			os.Exit(1)
		}

		metadata.FormatVersion = v

		enc := json.NewEncoder(mdBuf)

		err := enc.Encode(metadata)

		if err != nil {
			fmt.Println("ERROR: Failed to write metadata:", err)
			os.Exit(1)
		}

		// Write metadata buf to tar file
		err = tarAddBuf(tarWriter, mdBuf, "meta")

		if err != nil {
			fmt.Println("ERROR: Failed to write metadata to tar file:", err)
			os.Exit(1)
		}

		// Close tar file
		tarWriter.Close()

		compressed, err := os.Create(args[1])

		if err != nil {
			fmt.Println("ERROR: Failed to create compressed file:", err)
			os.Exit(1)
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
	},
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Gets info about a ibl db file",
	Long:  `Gets info about a ibl db file`,
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

		sections, meta, err := parseData(f)

		if err != nil {
			fmt.Println("ERROR:", err)
			os.Exit(1)
		}

		switch meta.Type {
		case "seed":
			seedMetaBuf, ok := sections["seed_meta"]

			if !ok {
				fmt.Println("Seed file is corrupt [no seed meta]")
				return
			}

			var smeta SeedMetadata

			err = json.NewDecoder(seedMetaBuf).Decode(&smeta)

			if err != nil {
				fmt.Println("WARNING: Seed file is corrupt [invalid seed meta]")
				return
			}

			fmt.Println("\n== Seed Info ==")
			fmt.Println("Nonce:", smeta.Nonce)
			fmt.Println("Default Database:", smeta.DefaultDatabase)
			fmt.Println("Source Database:", smeta.SourceDatabase)
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

		sections, meta, err := parseData(data)

		if err != nil {
			fmt.Println("ERROR: Parsing data failed", err)
			os.Exit(1)
		}

		if meta == nil {
			fmt.Println("ERROR: No metadata present!")
			os.Exit(1)
		}

		if meta.Protocol != protocol && os.Getenv("SKIP_PROTOCOL_CHECK") != "true" {
			fmt.Println("ERROR: Database file is of an invalid version [version is", meta.Protocol, "but expected", protocol, "]")
			os.Exit(1)
		}

		switch meta.Type {
		case "backup":
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
				decrData, err = decryptData(encData, enc, privKeyFileContents)

				if err != nil {
					fmt.Println("ERROR: Failed to decrypt data:", err)
					os.Exit(1)
				}
			} else {
				fmt.Println("WARNING: Backup data is not encrypted!")
				decrData = encData
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
		case "seed":
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

			// Restore backed up tables
			var tables []string

			for key := range sections {
				if strings.HasPrefix(key, "backup/") {
					tables = append(tables, key[7:])
				}
			}

			for i, table := range tables {
				fmt.Printf("Restoring table: [%d/%d] %s\n", i+1, len(tables), table)

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
		}
	},
}

// copyDb represents the copydb command
var copyDb = &cobra.Command{
	Use:   "copydb <database> <targetServer>",
	Short: "Copies a database from current server to target server. User must currently be on current server",
	Long:  `Copies a database from current server to target server. User must currently be on current server`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			fmt.Println("Cleaning up...")

			// delete all files in work directory
			err := os.RemoveAll("work")

			if err != nil {
				fmt.Println("Error cleaning up:", err)
			}
		}()

		ctx := context.Background()

		dbName := args[1]

		// create a work directory
		err := os.Mkdir("work", 0755)

		if err != nil {
			fmt.Println("Error creating work directory:", err)
			return
		}

		fmt.Println("Creating unsanitized database backup in memory")

		var buf = bytes.NewBuffer([]byte{})
		backupCmd := exec.Command("pg_dump", "-Fc", "-d", dbName)
		backupCmd.Env = os.Environ()
		backupCmd.Stdout = buf

		err = backupCmd.Run()

		if err != nil {
			fmt.Println("Error when creating db backup", err)
			return
		}

		if buf.Len() == 0 {
			fmt.Println("ERROR: Database backup is empty!")
			return
		}

		// Make copy (__dbcopy) using created db backup on source server
		fmt.Println("Creating copy of database on source server with name '" + dbName + "__dbcopy'")

		copyDbName := dbName + "__dbcopy"

		conn, err := pgx.Connect(ctx, "postgres:///"+dbName)

		if err != nil {
			fmt.Println("Failed to acquire database conn:", err)
			return
		}

		sqlCmds := []string{
			"DROP DATABASE IF EXISTS " + copyDbName,
			"CREATE DATABASE " + copyDbName,
		}

		for _, c := range sqlCmds {
			fmt.Println("[psql, origDb] =>", c)
			_, err = conn.Exec(ctx, c)

			if err != nil {
				fmt.Println("Failed to execute sql command:", err)
				return
			}
		}

		err = conn.Close(ctx)

		if err != nil {
			fmt.Println("WARNING: Failed to close conn:", err)
		}

		conn, err = pgx.Connect(ctx, "postgres:///"+copyDbName)

		if err != nil {
			fmt.Println("Failed to acquire copy database conn:", err)
			return
		}

		sqlCmds = []string{
			"CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"",
			"CREATE EXTENSION IF NOT EXISTS \"citext\"",
		}

		for _, c := range sqlCmds {
			fmt.Println("[psql, copyDb] =>", c)

			_, err = conn.Exec(ctx, c)

			if err != nil {
				fmt.Println("Failed to execute sql command:", err)
				return
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
			fmt.Println("Error when restoring db backup", err)
			return
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

		switch dbName {
		case "infinity":
			conn, err = pgx.Connect(ctx, "postgres:///"+copyDbName)

			if err != nil {
				fmt.Println("Failed to acquire copy database conn:", err)
				return
			}

			sqlCmds = []string{
				"DELETE FROM webhooks",
				"UPDATE users SET api_token = uuid_generate_v4()::text",
				"UPDATE bots SET api_token = uuid_generate_v4()::text",
				"UPDATE servers SET api_token = uuid_generate_v4()::text",
			}

			for _, c := range sqlCmds {
				fmt.Println("[psql, copyDb] =>", c)

				_, err = conn.Exec(ctx, c)

				if err != nil {
					fmt.Println("Failed to execute sql command:", err)
					return
				}
			}
		default:
			fmt.Println("WARNING: No sanitization task for database", dbName)
		}

		fmt.Println("Creating sanitized database backup as work/schema.sql")

		backupCmd = exec.Command("pg_dump", "-Fc", "-d", copyDbName, "-f", "work/schema.sql")

		backupCmd.Env = os.Environ()

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

		prodMarkerName := dbName + "__prodmarker"

		cmds := [][]string{
			{
				"psql", "-c", "'DROP DATABASE IF EXISTS " + dbName + "'",
			},
			{
				"psql", "-c", "'CREATE DATABASE " + dbName + "'",
			},
			{
				"psql", "-d", dbName, "-c", "'CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"'",
			},
			{
				"psql", "-d", dbName, "-c", "'CREATE EXTENSION IF NOT EXISTS \"citext\"'",
			},
			{
				"pg_restore", "-d", dbName, "/tmp/schema.sql",
			},
			{
				"psql", "-c", "'DROP DATABASE IF EXISTS " + prodMarkerName,
			},
			{
				"psql", "-c", "'CREATE DATABASE " + prodMarkerName,
			},
			{
				"psql", "-d", prodMarkerName, "-c", "'CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"'",
			},
			{
				"psql", "-d", prodMarkerName, "-c", "'CREATE EXTENSION IF NOT EXISTS \"citext\"'",
			},
			{
				"pg_restore", "-d", prodMarkerName, "/tmp/schema.sql",
			},
			{
				"rm", "/tmp/schema.sql",
			},
		}

		for _, c := range cmds {
			fmt.Println("(ssh) =>", strings.Join(c, " "))

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
	infoCmd.PersistentFlags().Bool("show-pubkey", false, "Whether or not to show the public key for the encrypted data")

	copyDb.PersistentFlags().String("db", "", "The database to copy from")

	loadCmd.PersistentFlags().String("priv-key", "", "The private key to decrypt the backup with [backup only]")
	loadCmd.PersistentFlags().String("db", "", "If type is backup, the database to restore the backup to (backup) or the database name to seed to (seed).")

	newCmd.PersistentFlags().String("pubkey", "", "The public key to encrypt the seed with")
	newCmd.PersistentFlags().String("default-db", "", "If type is seed, the default database name to seed to.")
	newCmd.PersistentFlags().String("db", "", "If type is backup, the database to backup to (backup) or the database name to seed from (seed).")
	newCmd.PersistentFlags().String("backup-tables", "", "The tables to fully backup in the seed [seed only]")

	if devmode.DevMode().Allows(types.DevModeFull) {
		dbCmd.AddCommand(infoCmd)
		dbCmd.AddCommand(genCiSchemaCmd)
		dbCmd.AddCommand(newCmd)
	}

	dbCmd.AddCommand(loadCmd)

	rootCmd.AddCommand(dbCmd)
}
