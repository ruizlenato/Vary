package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type ProgressCallback func(downloaded, total int64)

func Download(url, destPath string, progress ProgressCallback) error {

	tempPath := destPath + ".part"

	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	out, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to start download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	totalSize := resp.ContentLength

	reader := &progressReader{
		reader:   resp.Body,
		total:    totalSize,
		progress: progress,
	}

	_, err = io.Copy(out, reader)
	if err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to download: %w", err)
	}

	out.Close()

	if err := os.Rename(tempPath, destPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to finalize download: %w", err)
	}

	return nil
}

type progressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	progress   ProgressCallback
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.downloaded += int64(n)
	if pr.progress != nil {
		pr.progress(pr.downloaded, pr.total)
	}
	return n, err
}
