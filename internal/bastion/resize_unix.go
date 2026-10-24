//go:build !windows

package bastion

import (
	"os"
	"os/signal"
	"syscall"
)

// watchResize calls resize whenever the terminal window changes size and
// returns a function that stops watching.
func watchResize(resize func()) (stop func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ch:
				resize()
			case <-done:
				return
			}
		}
	}()
	return func() {
		signal.Stop(ch)
		close(done)
	}
}
