// Package bastion starts a throwaway container on a Docker network and
// attaches the local terminal to a shell inside it.
package bastion

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
	"golang.org/x/term"

	"github.com/tk31lab/fumi/internal/dockerutil"
	"github.com/tk31lab/fumi/internal/network"
)

const (
	// DefaultImage is small, ships a POSIX sh and has a package manager
	// (apk) for adding clients later.
	DefaultImage = "alpine:3.24"
	DefaultShell = "sh"

	// WorkMountPoint is where the directory fumi was started from is
	// mounted inside the bastion.
	WorkMountPoint = "/work"
)

// Options configures a bastion container.
type Options struct {
	Network string
	Image   string
	Shell   string
	// Name is the container name; a themed name is generated when empty.
	Name string
	// HostDir is an absolute path mounted read-write at WorkMountPoint, and
	// the shell's working directory. Nothing is mounted when empty.
	HostDir string

	Stdin  *os.File
	Stdout *os.File
	Stderr io.Writer
}

// Run starts the bastion, attaches the terminal until the shell exits and
// returns the shell's exit code. The container is removed afterwards.
func Run(ctx context.Context, cli *client.Client, opts Options) (int, error) {
	if opts.Name != "" {
		if err := dockerutil.ValidateName(opts.Name); err != nil {
			return 0, err
		}
	}
	if !term.IsTerminal(int(opts.Stdin.Fd())) || !term.IsTerminal(int(opts.Stdout.Fd())) {
		return 0, errors.New("run needs an interactive terminal (stdin and stdout must be a TTY)")
	}
	if err := network.Check(ctx, cli, opts.Network); err != nil {
		return 0, err
	}
	if err := dockerutil.EnsureImage(ctx, cli, opts.Image, opts.Stderr); err != nil {
		return 0, err
	}

	id, name, err := dockerutil.CreateContainer(ctx, cli, opts.Name, func(name string) client.ContainerCreateOptions {
		return createOptions(name, opts)
	})
	if err != nil {
		if opts.HostDir != "" && strings.Contains(err.Error(), "bind source path does not exist") {
			return 0, fmt.Errorf("cannot mount %s: it does not exist on the Docker host (is DOCKER_HOST a remote daemon?)", opts.HostDir)
		}
		return 0, err
	}
	return attachAndStart(ctx, cli, id, name, opts)
}

func createOptions(name string, opts Options) client.ContainerCreateOptions {
	cfg := &container.Config{
		Image:        opts.Image,
		Cmd:          []string{opts.Shell},
		Hostname:     name,
		Env:          promptEnv(opts.Shell, name),
		Labels:       dockerutil.Labels(dockerutil.KindShell),
		Tty:          true,
		OpenStdin:    true,
		StdinOnce:    true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}
	hostCfg := &container.HostConfig{
		AutoRemove:  true,
		NetworkMode: container.NetworkMode(opts.Network),
	}
	if opts.HostDir != "" {
		// A structured mount, unlike a "src:dst" bind string, needs no
		// parsing and so copes with Windows paths such as C:\work.
		hostCfg.Mounts = []mount.Mount{{Type: mount.TypeBind, Source: opts.HostDir, Target: WorkMountPoint}}
		cfg.WorkingDir = WorkMountPoint
	}
	return client.ContainerCreateOptions{Config: cfg, HostConfig: hostCfg}
}

// promptEnv shows the container name in the prompt. Only shells that
// understand the \w escape get it; others keep their own default.
func promptEnv(shell, name string) []string {
	switch shell {
	case "sh", "ash", "bash", "/bin/sh", "/bin/ash", "/bin/bash":
		return []string{`PS1=` + name + `:\w \$ `}
	}
	return nil
}

func attachAndStart(ctx context.Context, cli *client.Client, id, name string, opts Options) (int, error) {
	forceRemove := func() { dockerutil.ForceRemove(ctx, cli, id) }

	// Attach before start so that the first prompt is not lost.
	att, err := cli.ContainerAttach(ctx, id, client.ContainerAttachOptions{
		Stream: true, Stdin: true, Stdout: true, Stderr: true,
	})
	if err != nil {
		forceRemove()
		return 0, fmt.Errorf("attach to container: %w", err)
	}
	defer att.Close()

	// Register the wait before start; with AutoRemove the container may be
	// gone before a later wait request would reach the daemon.
	wait := cli.ContainerWait(ctx, id, client.ContainerWaitOptions{Condition: container.WaitConditionRemoved})

	if _, err := cli.ContainerStart(ctx, id, client.ContainerStartOptions{}); err != nil {
		forceRemove()
		if strings.Contains(err.Error(), "executable file not found") || strings.Contains(err.Error(), "no such file or directory") {
			return 0, fmt.Errorf("shell %q was not found in image %q (try --shell sh)", opts.Shell, opts.Image)
		}
		return 0, fmt.Errorf("start container: %w", err)
	}

	fmt.Fprintf(opts.Stderr, "Started %s on network %s (exit the shell to remove it)\n", name, opts.Network)
	if opts.HostDir != "" {
		fmt.Fprintf(opts.Stderr, "Mounted %s at %s\n", opts.HostDir, WorkMountPoint)
	}

	stdinFd := int(opts.Stdin.Fd())
	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		forceRemove()
		return 0, fmt.Errorf("set terminal to raw mode: %w", err)
	}
	defer term.Restore(stdinFd, oldState)

	resize := func() {
		w, h, err := term.GetSize(int(opts.Stdout.Fd()))
		if err != nil || w == 0 || h == 0 {
			return
		}
		_, _ = cli.ContainerResize(ctx, id, client.ContainerResizeOptions{Width: uint(w), Height: uint(h)})
	}
	resize()
	stopResize := watchResize(resize)
	defer stopResize()

	// Signals that would kill fumi (the terminal closing, kill) must not
	// leave the container behind. Ctrl+C is not a signal in raw mode: it is
	// sent to the shell as a byte.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigs)

	go func() {
		_, _ = io.Copy(att.Conn, opts.Stdin)
		_ = att.CloseWrite()
	}()
	outputDone := make(chan error, 1)
	go func() {
		// With a TTY the stream is raw, not multiplexed.
		_, err := io.Copy(opts.Stdout, att.Reader)
		outputDone <- err
	}()

	select {
	case <-outputDone:
	case <-sigs:
		forceRemove()
		return 1, nil
	}

	select {
	case res := <-wait.Result:
		if res.Error != nil {
			return 0, fmt.Errorf("wait for container: %s", res.Error.Message)
		}
		return int(res.StatusCode), nil
	case err := <-wait.Error:
		return 0, fmt.Errorf("wait for container: %w", err)
	}
}
