package helpers

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"
	"unsafe"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/schollz/progressbar/v3"
)

// Creates a env based on os.Environ
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

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
func RandString(n int) string {
	var src = rand.NewSource(time.Now().UnixNano())

	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
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
		return "https://devel.infinitybots.xyz"
	} else {
		return os.Getenv("ASSETS_URL")
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
