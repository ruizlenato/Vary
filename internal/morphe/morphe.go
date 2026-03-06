package morphe

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type Executor struct {
	cliPath     string
	patchesPath string
}

func NewExecutor(cliPath, patchesPath string) *Executor {
	return &Executor{
		cliPath:     cliPath,
		patchesPath: patchesPath,
	}
}

func (e *Executor) ListPackages(ctx context.Context) ([]string, error) {
	if err := e.checkJava(); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "java", "-jar", e.cliPath, "list-versions", e.patchesPath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute morphe-cli: %w (stderr: %s)", err, stderr.String())
	}

	return ParsePackages(stdout.String()), nil
}

func (e *Executor) checkJava() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "java", "-version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Java not found. Install Java and try again")
	}
	return nil
}

func ParsePackages(output string) []string {
	re := regexp.MustCompile(`(?m)^\s*(?:INFORMAÇÕES:\s*)?Package name:\s*([\w\.]+)\s*$`)

	matches := re.FindAllStringSubmatch(output, -1)

	seen := make(map[string]bool)
	var packages []string

	for _, match := range matches {
		if len(match) > 1 {
			pkg := strings.TrimSpace(match[1])
			if pkg != "" && !seen[pkg] {
				seen[pkg] = true
				packages = append(packages, pkg)
			}
		}
	}

	return packages
}
