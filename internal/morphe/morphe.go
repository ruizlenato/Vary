package morphe

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Executor struct {
	cliPath     string
	patchesPath string
}

type Patch struct {
	Name        string
	Description string
	Enabled     bool
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

func (e *Executor) ListPatches(ctx context.Context, packageName string) ([]Patch, error) {
	if err := e.checkJava(); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "java", "-jar", e.cliPath, "list-patches", e.patchesPath, "-f", packageName)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute morphe-cli: %w (stderr: %s)", err, stderr.String())
	}

	return ParsePatches(stdout.String()), nil
}

func (e *Executor) ListCompatibleVersions(ctx context.Context, packageName string) ([]string, error) {
	if err := e.checkJava(); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "java", "-jar", e.cliPath, "list-versions", e.patchesPath, "-f", packageName)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute morphe-cli: %w (stderr: %s)", err, stderr.String())
	}

	return ParseCompatibleVersions(stdout.String()), nil
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

func ParsePatches(output string) []Patch {
	patches := make([]Patch, 0)
	seen := make(map[string]bool)

	appendCurrent := func(current *Patch) {
		if current.Name == "" {
			return
		}
		if seen[current.Name] {
			return
		}
		seen[current.Name] = true
		patches = append(patches, *current)
	}

	current := Patch{}
	lines := strings.Split(output, "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			appendCurrent(&current)
			current = Patch{}
			continue
		}

		switch {
		case strings.HasPrefix(line, "Name:"):
			current.Name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		case strings.HasPrefix(line, "Description:"):
			current.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		case strings.HasPrefix(line, "Enabled:"):
			enabledText := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "Enabled:")))
			current.Enabled = enabledText == "true"
		}
	}
	appendCurrent(&current)

	if len(patches) > 0 {
		sortPatchesByName(patches)
		return patches
	}

	legacy := regexp.MustCompile(`(?m)^\s*(?:INFORMAÇÕES:\s*)?Patch name:\s*([^\r\n]+?)\s*$`)
	matches := legacy.FindAllStringSubmatch(output, -1)
	for _, match := range matches {
		if len(match) <= 1 {
			continue
		}
		name := strings.TrimSpace(match[1])
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		patches = append(patches, Patch{Name: name})
	}

	sortPatchesByName(patches)
	return patches
}

func sortPatchesByName(patches []Patch) {
	sort.Slice(patches, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(patches[i].Name))
		right := strings.ToLower(strings.TrimSpace(patches[j].Name))
		return left < right
	})
}

func ParseCompatibleVersions(output string) []string {
	lines := strings.Split(output, "\n")
	re := regexp.MustCompile(`^([0-9][0-9A-Za-z\.\-_]+)\s*\(`)
	versions := make([]string, 0)
	seen := make(map[string]bool)

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		match := re.FindStringSubmatch(line)
		if len(match) < 2 {
			continue
		}
		version := strings.TrimSpace(match[1])
		if version == "" || seen[version] {
			continue
		}
		seen[version] = true
		versions = append(versions, version)
	}

	return versions
}
