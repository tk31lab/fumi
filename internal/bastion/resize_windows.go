//go:build windows

package bastion

// watchResize is a no-op on Windows, which has no SIGWINCH; the container
// keeps the size the terminal had at start.
func watchResize(func()) (stop func()) {
	return func() {}
}
