// Package managed lists and removes the containers fumi started, which are
// recognised by the fumi.managed label.
package managed

import (
	"context"
	"fmt"
	"slices"
	"strings"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/client"

	"github.com/tk31lab/fumi/internal/bastion"
	"github.com/tk31lab/fumi/internal/dockerutil"
	"github.com/tk31lab/fumi/internal/forward"
)

// Container describes a container started by fumi.
type Container struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Kind    string `json:"kind,omitempty"`
	Network string `json:"network"`
	Image   string `json:"image"`
	// Detail is what the container does: the mounted directory of a shell,
	// or the forward of a forward container.
	Detail string `json:"detail,omitempty"`
	State  string `json:"state"`
	Status string `json:"status"`
}

// List returns fumi's containers, running or not, sorted by name.
func List(ctx context.Context, cli *client.Client) ([]Container, error) {
	res, err := cli.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: make(client.Filters).Add("label", dockerutil.ManagedLabel+"=true"),
	})
	if err != nil {
		return nil, err
	}
	var out []Container
	for _, c := range res.Items {
		ctr := Container{
			ID:     c.ID,
			Kind:   c.Labels[dockerutil.KindLabel],
			Image:  c.Image,
			State:  string(c.State),
			Status: c.Status,
		}
		if len(c.Names) > 0 {
			ctr.Name = strings.TrimPrefix(c.Names[0], "/")
		}
		ctr.Network = c.HostConfig.NetworkMode
		if c.NetworkSettings != nil && len(c.NetworkSettings.Networks) > 0 {
			nets := make([]string, 0, len(c.NetworkSettings.Networks))
			for n := range c.NetworkSettings.Networks {
				nets = append(nets, n)
			}
			slices.Sort(nets)
			ctr.Network = strings.Join(nets, ",")
		}
		switch {
		case c.Labels[forward.SpecLabel] != "":
			ctr.Detail = c.Labels[forward.SpecLabel]
		default:
			for _, m := range c.Mounts {
				if m.Destination == bastion.WorkMountPoint {
					ctr.Detail = m.Source + " -> " + bastion.WorkMountPoint
				}
			}
		}
		out = append(out, ctr)
	}
	slices.SortFunc(out, func(a, b Container) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// Names returns the names of fumi's containers, for shell completion.
func Names(ctx context.Context, cli *client.Client) ([]string, error) {
	ctrs, err := List(ctx, cli)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(ctrs))
	for i, c := range ctrs {
		names[i] = c.Name
	}
	return names, nil
}

// Resolve maps container names to fumi's containers. A name that does not
// exist, or belongs to a container fumi did not start, is an error, so that
// `fumi rm` can never remove anything else.
func Resolve(ctx context.Context, cli *client.Client, names []string) ([]Container, error) {
	all, err := List(ctx, cli)
	if err != nil {
		return nil, err
	}
	var out []Container
	for _, name := range names {
		i := slices.IndexFunc(all, func(c Container) bool { return c.Name == name })
		if i >= 0 {
			if !slices.ContainsFunc(out, func(c Container) bool { return c.ID == all[i].ID }) {
				out = append(out, all[i])
			}
			continue
		}
		if _, err := cli.ContainerInspect(ctx, name, client.ContainerInspectOptions{}); err == nil {
			return nil, fmt.Errorf("container %q was not started by fumi; remove it with docker rm instead", name)
		} else if !cerrdefs.IsNotFound(err) {
			return nil, fmt.Errorf("inspect container %q: %w", name, err)
		}
		return nil, fmt.Errorf("no fumi container named %q (see fumi ls)", name)
	}
	return out, nil
}

// Remove force-removes a container, stopping it first if it is running.
func Remove(ctx context.Context, cli *client.Client, c Container) error {
	_, err := cli.ContainerRemove(ctx, c.ID, client.ContainerRemoveOptions{Force: true})
	if err != nil && !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("remove %s: %w", c.Name, err)
	}
	return nil
}
