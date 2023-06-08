// Package downloader defines a progress bar downloader
package downloader

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/schollz/progressbar/v3"
)

// DownloadFileWithProgress downloads a file with a progress bar
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
