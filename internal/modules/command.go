package modules

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type CommandRunner interface {
	Run(context.Context, string, []string, []string) error
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args []string, env []string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), env...)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s failed: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}
