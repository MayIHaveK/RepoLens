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
	"strconv"
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
		"-c", "core.pager=cat",
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
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
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
	if ctx.Err() != nil {
		return ctx.Err()
	}
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
	out, err = c.Output(ctx, root, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
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

// Batch keeps one cat-file process open; callers consume one object at a time.
type Batch struct {
	cmd    *exec.Cmd
	input  io.WriteCloser
	output *bufio.Reader
	stderr bytes.Buffer
	ctx    context.Context
	check  bool
}

func (c Client) OpenBatch(ctx context.Context, repository string, check bool) (*Batch, error) {
	mode := "--batch"
	if check {
		mode = "--batch-check"
	}
	b := &Batch{cmd: c.command(ctx, repository, "cat-file", mode), ctx: ctx, check: check}
	var err error
	b.input, err = b.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, err := b.cmd.StdoutPipe()
	if err != nil {
		_ = b.input.Close()
		return nil, err
	}
	b.output = bufio.NewReaderSize(output, 128*1024)
	b.cmd.Stderr = &b.stderr
	if err := b.cmd.Start(); err != nil {
		_ = b.input.Close()
		return nil, err
	}
	return b, nil
}

func (b *Batch) Read(objectID string, maxSize int64) ([]byte, int64, error) {
	if b.ctx.Err() != nil {
		return nil, 0, b.ctx.Err()
	}
	if len(objectID) != 40 && len(objectID) != 64 {
		return nil, 0, fmt.Errorf("invalid Git object ID")
	}
	for _, char := range objectID {
		if !strings.ContainsRune("0123456789abcdef", char) {
			return nil, 0, fmt.Errorf("invalid Git object ID")
		}
	}
	if _, err := io.WriteString(b.input, objectID+"\n"); err != nil {
		return nil, 0, err
	}
	header, err := b.output.ReadString('\n')
	if err != nil {
		return nil, 0, err
	}
	parts := strings.Fields(header)
	if len(parts) != 3 || parts[1] != "blob" {
		return nil, 0, fmt.Errorf("cannot read Git blob")
	}
	size, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || size < 0 {
		return nil, 0, fmt.Errorf("invalid Git blob size")
	}
	if b.check {
		return nil, size, nil
	}
	if size > maxSize {
		if _, err := io.CopyN(io.Discard, b.output, size+1); err != nil {
			return nil, size, err
		}
		return nil, size, fmt.Errorf("Git blob exceeds size limit")
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(b.output, data); err != nil {
		return nil, size, err
	}
	if delimiter, err := b.output.ReadByte(); err != nil || delimiter != '\n' {
		return nil, size, fmt.Errorf("invalid Git blob delimiter")
	}
	return data, size, nil
}

func (b *Batch) Close() error {
	_ = b.input.Close()
	// All requested responses were consumed; prevent a blocked child on parse failure.
	_ = b.cmd.Process.Kill()
	_ = b.cmd.Wait()
	return nil
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
