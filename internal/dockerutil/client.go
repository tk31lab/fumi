// Package dockerutil connects to the Docker daemon.
package dockerutil

import (
	"context"
	"fmt"

	"github.com/moby/moby/client"
)

// ManagedLabel marks containers started by fumi.
const ManagedLabel = "fumi.managed"

// Connect creates a Docker client configured from the environment
// (DOCKER_HOST, DOCKER_TLS_VERIFY, ...) and checks that the daemon responds.
func Connect(ctx context.Context) (*client.Client, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}
	if _, err := cli.Ping(ctx, client.PingOptions{NegotiateAPIVersion: true}); err != nil {
		cli.Close()
		return nil, fmt.Errorf("cannot connect to the Docker daemon at %s (is it running?): %w", cli.DaemonHost(), err)
	}
	return cli, nil
}
