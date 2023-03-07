package helpers

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"syscall"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/crypto/scrypt"
	"golang.org/x/term"
)

// Creates a env based on os.Environ()
func GetEnv() []string {
	var env []string = make([]string, 0)
	if os.Getenv("PGDATABASE") != "" {
		env = append(env, "PGDATABASE="+os.Getenv("PGDATABASE"))
	} else {
		env = append(env, "PGDATABASE=infinity")
	}

	if os.Getenv("PGUSER") != "" {
		env = append(env, "PGUSER="+os.Getenv("PGUSER"))
	}

	return env
}

func GetPool() (*pgxpool.Pool, error) {
	return pgxpool.Connect(context.Background(), "postgres:///infinity")
}

func GetPoolNoUrl() (*pgxpool.Pool, error) {
	return pgxpool.Connect(context.Background(), "")
}

// Adds a buffer to a tar archive
func TarAddBuf(tarWriter *tar.Writer, buf *bytes.Buffer, name string) error {
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

func GetAssetsURL() string {
	if os.Getenv("ASSETS_URL") == "" {
		return "https://cdn.infinitybots.xyz/dev"
	} else {
		return os.Getenv("ASSETS_URL")
	}
}

func GetFrontendURL() string {
	if os.Getenv("FRONTEND_URL") == "" {
		return "https://ptb.infinitybots.gg"
	} else {
		return os.Getenv("FRONTEND_URL")
	}
}

func DownloadFileWithProgress(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"downloading",
	)
	var dlBuf = bytes.NewBuffer([]byte{})
	w, err := io.Copy(io.MultiWriter(dlBuf, bar), resp.Body)

	if err != nil {
		return nil, fmt.Errorf("error downloading file: %w with %d written", err, w)
	}

	return dlBuf.Bytes(), nil
}

func MapKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func GetPassword(msg string) string {
	fmt.Print(msg + ": ")
	bytepw, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		os.Exit(1)
	}
	pass := string(bytepw)

	return pass
}

func GetInput(msg string, check func(s string) bool) string {
	for {
		fmt.Print(msg + ": ")
		var input string
		fmt.Scanln(&input)

		if check(input) {
			return input
		}

		fmt.Println("")
	}
}

func GenKey(pass string, salt string) (key []byte) {
	// Generates a passkey from pass and salt, 32 bit
	dk, err := scrypt.Key([]byte(pass), []byte(salt), 32768, 8, 1, 32)

	if err != nil {
		panic(err)
	}

	return dk
}

func Encrypt(key []byte, data []byte) ([]byte, error) {
	// generate a new aes cipher using our 32 byte long key
	c, err := aes.NewCipher(key)
	// if there are any errors, handle them
	if err != nil {
		return nil, err
	}

	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	gcm, err := cipher.NewGCM(c)
	// if any error generating new GCM
	// handle them
	if err != nil {
		return nil, err
	}

	// creates a new byte array the size of the nonce
	// which must be passed to Seal
	nonce := make([]byte, gcm.NonceSize())
	// populates our nonce with a cryptographically secure
	// random sequence
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// here we encrypt our text using the Seal function
	// Seal encrypts and authenticates plaintext, authenticates the
	// additional data and appends the result to dst, returning the updated
	// slice. The nonce must be NonceSize() bytes long and unique for all
	// time, for a given key.
	return gcm.Seal(nonce, nonce, data, nil), nil
}

func Decrypt(key []byte, data []byte) ([]byte, error) {
	// generate a new aes cipher using our 32 byte long key
	c, err := aes.NewCipher(key)
	// if any errors, handle them
	if err != nil {
		return nil, err
	}

	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	gcm, err := cipher.NewGCM(c)
	// if any error generating new GCM
	// handle them
	if err != nil {
		return nil, err
	}

	// gets the nonce size
	nonceSize := gcm.NonceSize()
	// extract the nonce from the encrypted data
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	// decrypt the data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	// if any errors decrypting
	// handle them
	if err != nil {
		return nil, err
	}

	// return the decrypted data
	return plaintext, nil
}

func ConfigFile() string {
	envCfg := os.Getenv("INFINITY_CONFIG")

	if envCfg != "" {
		return envCfg
	}

	s, err := os.UserConfigDir()

	if err != nil {
		panic(err)
	}

	if s == "" {
		panic("Error getting config dir")
	}

	return s
}

func WriteConfig(name string, data any) error {
	cfgFile := ConfigFile()

	return Write(cfgFile+"/"+name, data)
}

func Write(fn string, data any) error {
	// Create config file
	f, err := os.Create(fn)

	if err != nil {
		return err
	}

	bytes, err := json.Marshal(data)

	if err != nil {
		return err
	}

	w, err := f.Write(bytes)

	if err != nil {
		return err
	}

	fmt.Println("Write: wrote", w, "lines to", fn)

	return nil
}

func LoadConfig(name string) (string, bool) {
	cfgFile := ConfigFile()

	if fsi, err := os.Stat(cfgFile + "/" + name); err != nil || fsi.IsDir() {
		return "", false
	} else {
		f, err := os.Open(cfgFile + "/" + name)

		if err != nil {
			return "", false
		}

		defer f.Close()

		buf := bytes.Buffer{}

		_, err = io.Copy(&buf, f)

		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading config file:", err)
			return "", false
		}

		return buf.String(), true
	}
}
