// Package forward exposes a service on a Docker network on a loopback port
// of the Docker host, by running socat in a throwaway container.
package forward

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	mobynetwork "github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/tk31lab/fumi/internal/dockerutil"
	"github.com/tk31lab/fumi/internal/network"
)

const (
	// Image runs socat as its entrypoint.
	Image = "alpine/socat:1.8.1.3"

	// LoopbackAddr is the only address forwards are published on, so that
	// other machines cannot reach the forwarded service.
	LoopbackAddr = "127.0.0.1"

	// SpecLabel records the forward on the container, e.g. for listing.
	SpecLabel = "fumi.forward"
)

// Options configures a forward.
type Options struct {
	Network string
	Spec    Spec
	// Name is the container name; a themed name is generated when empty.
	Name string

	Stderr io.Writer
}

// Run starts the forward and blocks until it is stopped with Ctrl+C (or
// SIGTERM/SIGHUP) or socat exits. The container is removed afterwards.
func Run(ctx context.Context, cli *client.Client, opts Options) error {
	if opts.Name != "" {
		if err := dockerutil.ValidateName(opts.Name); err != nil {
			return err
		}
	}
	if err := network.Check(ctx, cli, opts.Network); err != nil {
		return err
	}
	if err := checkHost(ctx, cli, opts.Network, opts.Spec.Host); err != nil {
		return err
	}
	if err := dockerutil.EnsureImage(ctx, cli, Image, opts.Stderr); err != nil {
		return err
	}

	id, name, err := dockerutil.CreateContainer(ctx, cli, opts.Name, func(string) client.ContainerCreateOptions {
		return createOptions(opts)
	})
	if err != nil {
		return err
	}
	return attachAndStart(ctx, cli, id, name, opts)
}

// checkHost rejects a host that nothing on the network answers to, listing
// the names that do. IP addresses and dotted names may live outside the
// network (socat can reach them too), so they are not checked.
func checkHost(ctx context.Context, cli *client.Client, networkName, host string) error {
	if net.ParseIP(host) != nil || strings.Contains(host, ".") {
		return nil
	}
	names, err := network.DNSNames(ctx, cli, networkName)
	if err != nil {
		return fmt.Errorf("list containers on network %q: %w", networkName, err)
	}
	if slices.Contains(names, host) {
		return nil
	}
	msg := fmt.Sprintf("no running container on network %q answers to %q", networkName, host)
	if len(names) > 0 {
		msg += "\navailable names:\n  " + strings.Join(names, "\n  ")
	}
	return errors.New(msg)
}

func createOptions(opts Options) client.ContainerCreateOptions {
	spec := opts.Spec
	// socat listens on the remote port number inside the container; only
	// the published host port is visible to the user.
	port, _ := mobynetwork.PortFrom(spec.RemotePort, mobynetwork.TCP)
	labels := dockerutil.Labels(dockerutil.KindForward)
	labels[SpecLabel] = spec.String()

	return client.ContainerCreateOptions{
		Config: &container.Config{
			Image:      Image,
			Entrypoint: []string{"socat"},
			Cmd: []string{
				fmt.Sprintf("tcp-listen:%d,reuseaddr,fork", spec.RemotePort),
				fmt.Sprintf("tcp:%s:%d", spec.Host, spec.RemotePort),
			},
			Labels:       labels,
			ExposedPorts: mobynetwork.PortSet{port: {}},
			AttachStdout: true,
			AttachStderr: true,
		},
		HostConfig: &container.HostConfig{
			AutoRemove:  true,
			NetworkMode: container.NetworkMode(opts.Network),
			PortBindings: mobynetwork.PortMap{port: {{
				HostIP:   netip.MustParseAddr(LoopbackAddr),
				HostPort: strconv.Itoa(int(spec.LocalPort)),
			}}},
		},
	}
}

func attachAndStart(ctx context.Context, cli *client.Client, id, name string, opts Options) error {
	forceRemove := func() { dockerutil.ForceRemove(ctx, cli, id) }

	// Attach before start so that socat's errors (e.g. a refused connection
	// to the target) are shown.
	att, err := cli.ContainerAttach(ctx, id, client.ContainerAttachOptions{
		Stream: true, Stdout: true, Stderr: true,
	})
	if err != nil {
		forceRemove()
		return fmt.Errorf("attach to container: %w", err)
	}
	defer att.Close()

	// Register the wait before start; with AutoRemove the container may be
	// gone before a later wait request would reach the daemon.
	wait := cli.ContainerWait(ctx, id, client.ContainerWaitOptions{Condition: container.WaitConditionRemoved})

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigs)

	if _, err := cli.ContainerStart(ctx, id, client.ContainerStartOptions{}); err != nil {
		forceRemove()
		if msg := err.Error(); strings.Contains(msg, "port is already allocated") || strings.Contains(msg, "address already in use") {
			return fmt.Errorf("local port %d is already in use (pick another with LOCAL_PORT:%s:%d)",
				opts.Spec.LocalPort, opts.Spec.Host, opts.Spec.RemotePort)
		}
		return fmt.Errorf("start container: %w", err)
	}

	fmt.Fprintf(opts.Stderr, "Forwarding %s on %s via %s (Ctrl+C to stop)\n", opts.Spec, opts.Network, name)

	go func() {
		// Without a TTY, stdout and stderr are multiplexed.
		_, _ = stdcopy.StdCopy(opts.Stderr, opts.Stderr, att.Reader)
	}()

	select {
	case <-sigs:
		forceRemove()
		fmt.Fprintln(opts.Stderr, "Stopped")
		return nil
	case res := <-wait.Result:
		if res.Error != nil {
			return fmt.Errorf("socat stopped: %s", res.Error.Message)
		}
		return fmt.Errorf("socat exited unexpectedly with status %d", res.StatusCode)
	case err := <-wait.Error:
		forceRemove()
		return fmt.Errorf("wait for container: %w", err)
	}
}
