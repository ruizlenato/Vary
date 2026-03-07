package adb

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"vary/internal/downloader"
	"vary/internal/storage"
)

const commandTimeout = 5 * time.Second

var errAdbUnavailable = errors.New("adb unavailable")

var platformToolsURLs = map[string]string{
	"linux":   "https://dl.google.com/android/repository/platform-tools-latest-linux.zip",
	"darwin":  "https://dl.google.com/android/repository/platform-tools-latest-darwin.zip",
	"windows": "https://dl.google.com/android/repository/platform-tools-latest-windows.zip",
}

func IsAvailable() bool {
	_, err := adbCommandPath()
	return err == nil
}

func EnsurePlatformTools(progress downloader.ProgressCallback) error {
	if IsAvailable() {
		return nil
	}

	url, ok := platformToolsURLs[runtime.GOOS]
	if !ok {
		return fmt.Errorf("unsupported OS for platform-tools: %s", runtime.GOOS)
	}

	appDir, err := storage.AppDataDir("Vary")
	if err != nil {
		return err
	}
	if err := storage.EnsureDir(appDir); err != nil {
		return err
	}

	zipPath := filepath.Join(appDir, "platform-tools-latest-"+runtime.GOOS+".zip")
	if err := downloader.Download(url, zipPath, progress); err != nil {
		return err
	}

	if err := unzip(zipPath, appDir); err != nil {
		return err
	}
	_ = os.Remove(zipPath)

	if _, err := bundledAdbPath(); err != nil {
		return fmt.Errorf("platform-tools installed but adb not found: %w", err)
	}

	return nil
}

func FirstConnectedModel() (string, error) {
	serial, err := FirstConnectedSerial()
	if err != nil {
		return "", err
	}
	if serial == "" {
		return "", nil
	}

	return ModelForSerial(serial)
}

func FirstConnectedABI() (string, error) {
	serial, err := FirstConnectedSerial()
	if err != nil {
		return "", err
	}
	if serial == "" {
		return "", nil
	}

	return ABIForSerial(serial)
}

func FirstConnectedSerial() (string, error) {
	if _, err := adbCommandPath(); err != nil {
		if errors.Is(err, errAdbUnavailable) {
			return "", nil
		}
		return "", err
	}

	serial, err := firstConnectedSerial()
	if err != nil {
		return "", err
	}
	if serial == "" {
		return "", nil
	}
	return serial, nil
}

func ModelForSerial(serial string) (string, error) {
	if serial == "" {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	stdout, err := runAdbOutput(ctx, "-s", serial, "shell", "getprop", "ro.product.model")
	if err != nil {
		return "", err
	}

	model := strings.TrimSpace(stdout)
	if model == "" {
		return "", errors.New("empty model from adb")
	}

	return model, nil
}

func ABIForSerial(serial string) (string, error) {
	if serial == "" {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	stdout, err := runAdbOutput(ctx, "-s", serial, "shell", "getprop", "ro.product.cpu.abi")
	if err != nil {
		return "", err
	}

	abi := strings.TrimSpace(stdout)
	if abi == "" {
		return "", errors.New("empty abi from adb")
	}

	return abi, nil
}

func WatchFirstDeviceModel(ctx context.Context, onChange func(model string)) {
	if onChange == nil {
		return
	}

	lastModel := "\x00"
	emit := func(model string) {
		if model == lastModel {
			return
		}
		lastModel = model
		onChange(model)
	}

	refresh := func() {
		serial, err := FirstConnectedSerial()
		if err != nil || serial == "" {
			emit("")
			return
		}
		model, err := ModelForSerial(serial)
		if err != nil {
			emit("")
			return
		}
		emit(model)
	}

	backoff := 500 * time.Millisecond

	for {
		if ctx.Err() != nil {
			return
		}

		refresh()
		err := trackLoop(ctx, refresh)
		if ctx.Err() != nil {
			return
		}

		if err == nil {
			backoff = 500 * time.Millisecond
		} else if backoff < 5*time.Second {
			backoff *= 2
		}

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func trackLoop(ctx context.Context, onEvent func()) error {
	adbPath, err := adbCommandPath()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, adbPath, "track-devices")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		onEvent()
	}

	if err := scanner.Err(); err != nil {
		_ = cmd.Wait()
		return err
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}

	return nil
}

func firstConnectedSerial() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	stdout, err := runAdbOutput(ctx, "devices")
	if err != nil {
		return "", err
	}

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of devices attached") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "device" {
			return fields[0], nil
		}
	}

	return "", nil
}

func runAdbOutput(ctx context.Context, args ...string) (string, error) {
	adbPath, err := adbCommandPath()
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, adbPath, args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}

	return stdout.String(), nil
}

func adbCommandPath() (string, error) {
	if path, err := exec.LookPath("adb"); err == nil {
		return path, nil
	}

	return bundledAdbPath()
}

func bundledAdbPath() (string, error) {
	appDir, err := storage.AppDataDir("Vary")
	if err != nil {
		return "", errAdbUnavailable
	}

	adbName := "adb"
	if runtime.GOOS == "windows" {
		adbName = "adb.exe"
	}

	path := filepath.Join(appDir, "platform-tools", adbName)
	if _, err := os.Stat(path); err != nil {
		return "", errAdbUnavailable
	}

	return path, nil
}

func unzip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		targetPath := filepath.Join(destDir, f.Name)
		cleanDest := filepath.Clean(destDir) + string(os.PathSeparator)
		if !strings.HasPrefix(filepath.Clean(targetPath), cleanDest) {
			return fmt.Errorf("invalid zip path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		src, err := f.Open()
		if err != nil {
			return err
		}

		mode := f.Mode()
		if mode == 0 {
			mode = 0644
		}

		dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			src.Close()
			return err
		}

		_, copyErr := io.Copy(dst, src)
		closeDstErr := dst.Close()
		closeSrcErr := src.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeDstErr != nil {
			return closeDstErr
		}
		if closeSrcErr != nil {
			return closeSrcErr
		}

		if runtime.GOOS != "windows" && filepath.Base(targetPath) == "adb" {
			if err := os.Chmod(targetPath, 0755); err != nil {
				return err
			}
		}
	}

	return nil
}
