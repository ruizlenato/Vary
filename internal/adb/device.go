package adb

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

const commandTimeout = 5 * time.Second

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

func FirstConnectedSerial() (string, error) {
	if _, err := exec.LookPath("adb"); err != nil {
		return "", nil
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

	cmd := exec.CommandContext(ctx, "adb", "-s", serial, "shell", "getprop", "ro.product.model")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}

	model := strings.TrimSpace(stdout.String())
	if model == "" {
		return "", errors.New("empty model from adb")
	}

	return model, nil
}

func WatchFirstDeviceModel(ctx context.Context, onChange func(model string)) {
	if onChange == nil {
		return
	}

	if _, err := exec.LookPath("adb"); err != nil {
		onChange("")
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

	refresh()
	backoff := 500 * time.Millisecond

	for {
		if ctx.Err() != nil {
			return
		}

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
	cmd := exec.CommandContext(ctx, "adb", "track-devices")
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

	cmd := exec.CommandContext(ctx, "adb", "devices")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}

	lines := strings.Split(stdout.String(), "\n")
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
