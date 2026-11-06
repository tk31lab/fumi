package forward

import (
	"fmt"
	"strconv"
	"strings"
)

// Spec is a single TCP forward from a loopback port on the Docker host to a
// host and port on a Docker network.
type Spec struct {
	LocalPort  uint16
	Host       string
	RemotePort uint16
}

// ParseSpec parses "[LOCAL_PORT:]HOST:PORT", as in `ssh -L`. Without
// LOCAL_PORT the remote port number is used locally too.
func ParseSpec(s string) (Spec, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return Spec{}, fmt.Errorf("invalid forward %q: want [LOCAL_PORT:]HOST:PORT, e.g. 8090:web:80 or db:5432", s)
	}
	host, remote := parts[len(parts)-2], parts[len(parts)-1]
	if host == "" || strings.ContainsAny(host, " \t/") {
		return Spec{}, fmt.Errorf("invalid forward %q: bad host %q", s, host)
	}
	remotePort, err := parsePort(remote)
	if err != nil {
		return Spec{}, fmt.Errorf("invalid forward %q: port: %w", s, err)
	}
	localPort := remotePort
	if len(parts) == 3 {
		if localPort, err = parsePort(parts[0]); err != nil {
			return Spec{}, fmt.Errorf("invalid forward %q: local port: %w", s, err)
		}
	}
	return Spec{LocalPort: localPort, Host: host, RemotePort: remotePort}, nil
}

func parsePort(s string) (uint16, error) {
	n, err := strconv.ParseUint(s, 10, 16)
	if err != nil || n == 0 {
		return 0, fmt.Errorf("%q is not a port number (1-65535)", s)
	}
	return uint16(n), nil
}

func (s Spec) String() string {
	return fmt.Sprintf("%s:%d -> %s:%d", LoopbackAddr, s.LocalPort, s.Host, s.RemotePort)
}
