package dockerutil

import (
	"context"
	"fmt"
	"io"
	"regexp"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/client"

	"github.com/tk31lab/fumi/internal/namegen"
)

const (
	// KindLabel records what a fumi container is for.
	KindLabel   = "fumi.kind"
	KindShell   = "shell"
	KindForward = "forward"

	// maxNameAttempts bounds retries when a generated name is already taken.
	maxNameAttempts = 5
)

// validName is the container name rule enforced by the Docker daemon.
var validName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]+$`)

// Labels returns the labels every fumi container of the given kind carries.
func Labels(kind string) map[string]string {
	return map[string]string{ManagedLabel: "true", KindLabel: kind}
}

// ValidateName reports whether name is acceptable as a container name.
func ValidateName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid container name %q: only [a-zA-Z0-9][a-zA-Z0-9_.-]+ is allowed", name)
	}
	return nil
}

// EnsureImage pulls image unless it is already present locally.
func EnsureImage(ctx context.Context, cli *client.Client, image string, progress io.Writer) error {
	_, err := cli.ImageInspect(ctx, image)
	if err == nil {
		return nil
	}
	if !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("inspect image %q: %w", image, err)
	}
	fmt.Fprintf(progress, "Pulling %s ...\n", image)
	resp, err := cli.ImagePull(ctx, image, client.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %q: %w", image, err)
	}
	defer resp.Close()
	if err := resp.Wait(ctx); err != nil {
		return fmt.Errorf("pull image %q: %w", image, err)
	}
	return nil
}

// CreateContainer creates a container named name, or a generated name when
// name is empty, in which case a fresh name is tried on conflict. build
// returns the create options for a given name.
func CreateContainer(ctx context.Context, cli *client.Client, name string, build func(name string) client.ContainerCreateOptions) (id, usedName string, err error) {
	for attempt := 1; ; attempt++ {
		usedName = name
		if usedName == "" {
			usedName = namegen.Generate()
		}
		opts := build(usedName)
		opts.Name = usedName
		res, err := cli.ContainerCreate(ctx, opts)
		if err == nil {
			return res.ID, usedName, nil
		}
		if !cerrdefs.IsConflict(err) {
			return "", "", fmt.Errorf("create container: %w", err)
		}
		if name != "" {
			return "", "", fmt.Errorf("container name %q is already in use", name)
		}
		if attempt == maxNameAttempts {
			return "", "", fmt.Errorf("could not find a free container name after %d attempts; pass --name", attempt)
		}
	}
}

// ForceRemove removes a container, ignoring errors. It does not depend on
// ctx being live, since it is used for cleanup after cancellation.
func ForceRemove(ctx context.Context, cli *client.Client, id string) {
	_, _ = cli.ContainerRemove(context.WithoutCancel(ctx), id, client.ContainerRemoveOptions{Force: true})
}
