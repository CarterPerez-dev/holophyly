/*
AngelaMos | 2026
compose.go
*/

package docker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type ComposeResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

/*
ComposeUp starts services defined in a compose file.
Equivalent to: docker compose -f <file> up -d
*/
func ComposeUp(
	ctx context.Context,
	composePath string,
) (*ComposeResult, error) {
	return runComposeCommand(ctx, composePath, "up", "-d", "--remove-orphans")
}

/*
ComposeDown stops and removes services defined in a compose file.
Equivalent to: docker compose -f <file> down
*/
func ComposeDown(
	ctx context.Context,
	composePath string,
) (*ComposeResult, error) {
	return runComposeCommand(ctx, composePath, "down")
}

/*
ComposeRestart restarts services defined in a compose file.
*/
func ComposeRestart(
	ctx context.Context,
	composePath string,
) (*ComposeResult, error) {
	return runComposeCommand(ctx, composePath, "restart")
}

/*
ComposePull pulls latest images for services defined in a compose file.
*/
func ComposePull(
	ctx context.Context,
	composePath string,
) (*ComposeResult, error) {
	return runComposeCommand(ctx, composePath, "pull")
}

/*
ComposePs lists containers for a compose project.
*/
func ComposePs(
	ctx context.Context,
	composePath string,
) (*ComposeResult, error) {
	return runComposeCommand(ctx, composePath, "ps", "--format", "json")
}

/*
ComposeLogs gets logs from compose services.
*/
func ComposeLogs(
	ctx context.Context,
	composePath, tail string,
) (*ComposeResult, error) {
	if tail == "" {
		tail = "100"
	}
	return runComposeCommand(
		ctx,
		composePath,
		"logs",
		"--tail",
		tail,
		"--no-color",
	)
}

/*
ComposeConfig validates and returns the compose configuration.
*/
func ComposeConfig(
	ctx context.Context,
	composePath string,
) (*ComposeResult, error) {
	return runComposeCommand(ctx, composePath, "config")
}

func runComposeCommand(
	ctx context.Context,
	composePath string,
	args ...string,
) (*ComposeResult, error) {
	dir := filepath.Dir(composePath)
	file := filepath.Base(composePath)

	cmdArgs := []string{"compose", "-f", file}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &ComposeResult{
		Success: err == nil,
		Output:  strings.TrimSpace(stdout.String()),
		Error:   strings.TrimSpace(stderr.String()),
	}

	if err != nil {
		if result.Error == "" {
			result.Error = err.Error()
		}
		return result, fmt.Errorf("compose command failed: %w", err)
	}

	return result, nil
}

/*
GetComposeProjectName extracts the project name from a compose file path.
Docker Compose uses the directory name as the default project name.
*/
func GetComposeProjectName(composePath string) string {
	dir := filepath.Dir(composePath)
	return filepath.Base(dir)
}

/*
IsComposeInstalled checks if docker compose is available.
*/
func IsComposeInstalled(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "compose", "version")
	return cmd.Run() == nil
}
