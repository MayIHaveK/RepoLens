package gitutil

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
)

type Client struct {
	Binary string
}

func New() Client {
	return Client{Binary: "git"}
}

func (c Client) command(ctx context.Context, repository string, args ...string) *exec.Cmd {
	base := []string{
		"-c", "core.quotepath=false",
		"-c", "color.ui=false",
		"-c", "log.showSignature=false",
		"--no-pager",
		"-C", repository,
	}
	return exec.CommandContext(ctx, c.Binary, append(base, args...)...)
}

func (c Client) Output(ctx context.Context, repository string, args ...string) ([]byte, error) {
	cmd := c.command(ctx, repository, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", args[0], message)
	}
	return output, nil
}

func (c Client) Stream(ctx context.Context, repository string, args []string, consume func(io.Reader) error) error {
	cmd := c.command(ctx, repository, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	consumeErr := consume(stdout)
	if consumeErr != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if consumeErr != nil {
		return consumeErr
	}
	if waitErr != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = waitErr.Error()
		}
		return errors.New(message)
	}
	return nil
}

func (c Client) ResolveRepository(ctx context.Context, repository, ref string) (root, sha, name, headDate string, err error) {
	abs, err := filepath.Abs(repository)
	if err != nil {
		return "", "", "", "", err
	}
	out, err := c.Output(ctx, abs, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", "", "", fmt.Errorf("not a Git repository: %w", err)
	}
	root = strings.TrimSpace(string(out))
	out, err = c.Output(ctx, root, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		return "", "", "", "", fmt.Errorf("cannot resolve ref %q: %w", ref, err)
	}
	sha = strings.TrimSpace(string(out))
	out, err = c.Output(ctx, root, "show", "-s", "--format=%aI", sha)
	if err == nil {
		headDate = strings.TrimSpace(string(out))
	}
	name = filepath.Base(root)
	return root, sha, name, headDate, nil
}

func ScanLines(reader io.Reader, callback func(string) error) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		if err := callback(scanner.Text()); err != nil {
			return err
		}
	}
	return scanner.Err()
}
