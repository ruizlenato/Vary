package morphe

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"vary/internal/storage"
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

	cmd := exec.CommandContext(ctx, "java", "-jar", e.cliPath, "list-patches", "--patches", e.patchesPath, "-f", packageName)

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

func (e *Executor) PatchApp(ctx context.Context, inputFile string, includePatches []string) error {
	return e.patchApp(ctx, inputFile, includePatches, nil)
}

func (e *Executor) PatchAppWithLogs(ctx context.Context, inputFile string, includePatches []string, onLog func(line string, isErr bool)) error {
	return e.patchApp(ctx, inputFile, includePatches, onLog)
}

func (e *Executor) patchApp(ctx context.Context, inputFile string, includePatches []string, onLog func(line string, isErr bool)) error {
	if err := e.checkJava(); err != nil {
		return err
	}
	if inputFile == "" {
		return fmt.Errorf("missing input file")
	}
	if len(includePatches) == 0 {
		return fmt.Errorf("no patches selected")
	}

	args := []string{"-jar", e.cliPath, "patch", "--patches", e.patchesPath, "--exclusive"}
	for _, patch := range includePatches {
		name := strings.TrimSpace(patch)
		if name == "" {
			continue
		}
		args = append(args, "-e", name)
	}
	args = append(args, inputFile)

	cmd := exec.CommandContext(ctx, "java", args...)
	appDir, err := storage.AppDataDir("Vary")
	if err == nil {
		if mkErr := storage.EnsureDir(appDir); mkErr == nil {
			cmd.Dir = appDir
		}
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	var stderr bytes.Buffer
	var mu sync.Mutex
	emit := func(line string, isErr bool) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			return
		}
		if isErr {
			mu.Lock()
			if stderr.Len() > 0 {
				stderr.WriteByte('\n')
			}
			stderr.WriteString(trimmed)
			mu.Unlock()
		}
		if onLog != nil {
			onLog(trimmed, isErr)
		}
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	startedAt := time.Now()

	var wg sync.WaitGroup
	scan := func(scanner *bufio.Scanner, isErr bool) {
		defer wg.Done()
		for scanner.Scan() {
			emit(scanner.Text(), isErr)
		}
	}

	stdoutScanner := bufio.NewScanner(stdoutPipe)
	stderrScanner := bufio.NewScanner(stderrPipe)
	wg.Add(2)
	go scan(stdoutScanner, false)
	go scan(stderrScanner, true)

	wg.Wait()

	if err := cmd.Wait(); err != nil {
		mu.Lock()
		errText := strings.TrimSpace(stderr.String())
		mu.Unlock()
		return fmt.Errorf("failed to patch app: %w (stderr: %s)", err, errText)
	}

	if movedPath, moveErr := movePatchedOutputToInputDir(cmd.Dir, inputFile, startedAt); moveErr == nil && movedPath != "" {
		emit("Output file: "+movedPath, false)
	}

	return nil
}

func movePatchedOutputToInputDir(workDir, inputFile string, since time.Time) (string, error) {
	if workDir == "" || inputFile == "" {
		return "", nil
	}

	inputDir := filepath.Dir(inputFile)
	inputBase := filepath.Base(inputFile)

	if samePath(workDir, inputDir) {
		return "", nil
	}

	generated, err := findLatestGeneratedAPK(workDir, inputBase, since)
	if err != nil || generated == "" {
		return "", err
	}

	destination := filepath.Join(inputDir, filepath.Base(generated))
	if samePath(generated, destination) {
		return destination, nil
	}

	if err := os.Rename(generated, destination); err != nil {
		if copyErr := copyFile(generated, destination); copyErr != nil {
			return "", copyErr
		}
		_ = os.Remove(generated)
	}

	return destination, nil
}

func findLatestGeneratedAPK(dir, inputBase string, since time.Time) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	best := ""
	bestTime := time.Time{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".apk") {
			continue
		}
		if name == inputBase {
			continue
		}
		full := filepath.Join(dir, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(since.Add(-2 * time.Second)) {
			continue
		}
		if best == "" || info.ModTime().After(bestTime) {
			best = full
			bestTime = info.ModTime()
		}
	}

	return best, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}

	return nil
}

func samePath(a, b string) bool {
	ca := filepath.Clean(a)
	cb := filepath.Clean(b)
	if ca == cb {
		return true
	}
	if filepath.VolumeName(ca) != "" || filepath.VolumeName(cb) != "" {
		return strings.EqualFold(ca, cb)
	}
	return false
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
