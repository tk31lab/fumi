// Package network lists and checks the Docker networks fumi containers join.
package network

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/client"
)

const (
	composeProjectLabel = "com.docker.compose.project"
	composeServiceLabel = "com.docker.compose.service"
)

// predefined are the networks Docker creates on its own. host and none
// cannot be joined at all, and the default bridge has no DNS by container
// name, so none of them are useful as a bastion's network.
var predefined = []string{"bridge", "host", "none"}

// IsPredefined reports whether name is one of Docker's built-in networks.
func IsPredefined(name string) bool {
	return slices.Contains(predefined, name)
}

// Info describes a network and the running containers attached to it.
type Info struct {
	Name       string   `json:"name"`
	Driver     string   `json:"driver"`
	Project    string   `json:"project,omitempty"`
	Containers []string `json:"containers"`
}

// Names returns the names of all networks, without the predefined ones
// unless all is true. It is cheap enough to back shell completion.
func Names(ctx context.Context, cli *client.Client, all bool) ([]string, error) {
	res, err := cli.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, n := range res.Items {
		if all || !IsPredefined(n.Name) {
			names = append(names, n.Name)
		}
	}
	slices.Sort(names)
	return names, nil
}

// List returns networks sorted by name, without the predefined ones unless
// all is true. Containers are named by their Compose service when they have
// one, otherwise by container name.
func List(ctx context.Context, cli *client.Client, all bool) ([]Info, error) {
	nets, err := cli.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return nil, err
	}
	ctrs, err := cli.ContainerList(ctx, client.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	members := map[string][]string{}
	for _, c := range ctrs.Items {
		if c.NetworkSettings == nil {
			continue
		}
		label := c.Labels[composeServiceLabel]
		if label == "" && len(c.Names) > 0 {
			label = strings.TrimPrefix(c.Names[0], "/")
		}
		for netName := range c.NetworkSettings.Networks {
			members[netName] = append(members[netName], label)
		}
	}

	var infos []Info
	for _, n := range nets.Items {
		if !all && IsPredefined(n.Name) {
			continue
		}
		cs := members[n.Name]
		slices.Sort(cs)
		infos = append(infos, Info{
			Name:       n.Name,
			Driver:     n.Driver,
			Project:    n.Labels[composeProjectLabel],
			Containers: cs,
		})
	}
	slices.SortFunc(infos, func(a, b Info) int { return strings.Compare(a.Name, b.Name) })
	return infos, nil
}

// Check verifies that a container can join the network name. When the
// network does not exist, the error lists the ones that do.
func Check(ctx context.Context, cli *client.Client, name string) error {
	if name == "host" || name == "none" {
		return fmt.Errorf("network %q cannot be used: fumi containers must join a bridge or overlay network", name)
	}
	_, err := cli.NetworkInspect(ctx, name, client.NetworkInspectOptions{})
	if err == nil {
		return nil
	}
	if !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("inspect network %q: %w", name, err)
	}
	msg := fmt.Sprintf("network %q not found", name)
	if names, err := Names(ctx, cli, false); err == nil && len(names) > 0 {
		msg += "\navailable networks:\n  " + strings.Join(names, "\n  ")
	} else {
		msg += " (run `fumi networks --all` to see every network)"
	}
	return errors.New(msg)
}

// DNSNames returns the names other containers can use to reach the running
// containers on network name: Compose service names, container names and
// aliases.
func DNSNames(ctx context.Context, cli *client.Client, name string) ([]string, error) {
	ctrs, err := cli.ContainerList(ctx, client.ContainerListOptions{
		Filters: make(client.Filters).Add("network", name),
	})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, c := range ctrs.Items {
		res, err := cli.ContainerInspect(ctx, c.ID, client.ContainerInspectOptions{})
		if err != nil {
			if cerrdefs.IsNotFound(err) {
				continue // stopped in the meantime
			}
			return nil, err
		}
		if res.Container.NetworkSettings == nil {
			continue
		}
		if ep := res.Container.NetworkSettings.Networks[name]; ep != nil {
			for _, n := range slices.Concat(ep.DNSNames, ep.Aliases) {
				// Docker also registers the short container ID; nobody
				// means that when naming a service.
				if len(c.ID) < 12 || n != c.ID[:12] {
					names = append(names, n)
				}
			}
		}
	}
	slices.Sort(names)
	return slices.Compact(names), nil
}
