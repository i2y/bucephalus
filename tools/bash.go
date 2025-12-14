package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/i2y/bucephalus/llm"
)

// BashInput defines the input for the Bash tool.
type BashInput struct {
	Command string `json:"command" jsonschema:"required,description=Shell command to execute"`
	Timeout int    `json:"timeout,omitempty" jsonschema:"description=Timeout in seconds (default: 30)"`
	WorkDir string `json:"workdir,omitempty" jsonschema:"description=Working directory for the command"`
}

// BashOutput defines the output of the Bash tool.
type BashOutput struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// BashTool returns the Bash tool.
func BashTool() (llm.Tool, error) {
	return llm.NewTool(
		"bash",
		"Execute a shell command and return stdout, stderr, and exit code.",
		executeBash,
	)
}

// MustBash returns the Bash tool, panicking on error.
func MustBash() llm.Tool {
	tool, err := BashTool()
	if err != nil {
		panic(err)
	}
	return tool
}

func executeBash(ctx context.Context, input BashInput) (BashOutput, error) {
	timeout := input.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "bash", "-c", input.Command)

	if input.WorkDir != "" {
		cmd.Dir = input.WorkDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if execCtx.Err() == context.DeadlineExceeded {
			return BashOutput{
				Stdout:   stdout.String(),
				Stderr:   fmt.Sprintf("command timed out after %d seconds", timeout),
				ExitCode: -1,
			}, nil
		} else {
			return BashOutput{}, fmt.Errorf("failed to execute command: %w", err)
		}
	}

	return BashOutput{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}, nil
}
