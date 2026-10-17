package tf

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"charm.land/lipgloss/v2"
)

// lipgloss.Fprint* downsample the styles to what the writer supports, and drop
// them when it is not a terminal.
var (
	dimStyle  = lipgloss.NewStyle().Faint(true)
	warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

func logf(format string, a ...any) {
	lipgloss.Fprintln(os.Stderr, dimStyle.Render(fmt.Sprintf(format, a...)))
}

func warnf(format string, a ...any) {
	lipgloss.Fprintf(os.Stderr, "%s %s\n", warnStyle.Render("warning:"), fmt.Sprintf(format, a...))
}

func die(format string, a ...any) {
	lipgloss.Fprintf(os.Stderr, "%s %s\n", errStyle.Render("error:"), fmt.Sprintf(format, a...))
	Exit(1)
}

// Anything that has to be cleaned up on the way out. Set as we go.
var cleanup struct {
	sync.Mutex
	// A plan file that terragrunt did not finish writing is worse than none
	// at all: it would still be picked up by '--plan-file latest'.
	plan string
	// Ciphertext only, but a half written one must not replace the real file.
	secrets string
	lock    string
}

// Exit removes whatever a run left behind and exits with code.
func Exit(code int) {
	cleanup.Lock()
	if cleanup.plan != "" {
		_ = os.Remove(cleanup.plan)
		_ = os.Remove(cleanup.plan + ".meta")
	}
	if cleanup.secrets != "" {
		_ = os.Remove(cleanup.secrets)
	}
	if cleanup.lock != "" {
		_ = os.RemoveAll(cleanup.lock)
	}
	os.Exit(code)
}

// The process tofu or terragrunt runs in, if any.
var child struct {
	sync.Mutex
	p *os.Process
}

// A Ctrl-C at the terminal reaches the whole process group, so terragrunt
// already has it and gets to shut down cleanly; forwarding it would count as a
// second interrupt and make tofu abandon its state lock. SIGTERM only reaches
// us and is passed on. Without a child there is nothing to wait for.
func handleSignals() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		for s := range sigs {
			child.Lock()
			p := child.p
			child.Unlock()
			if p != nil {
				if s == syscall.SIGTERM {
					_ = p.Signal(s)
				}
				continue
			}
			if s == syscall.SIGTERM {
				Exit(143)
			}
			Exit(130)
		}
	}()
}

// runCmd runs a command with the terminal attached and returns its exit code.
func runCmd(dir string, stdout io.Writer, name string, args ...string) int {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr

	child.Lock()
	if err := cmd.Start(); err != nil {
		child.Unlock()
		die("could not start %s: %v", name, err)
	}
	child.p = cmd.Process
	child.Unlock()

	err := cmd.Wait()

	child.Lock()
	child.p = nil
	child.Unlock()

	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		die("%s failed: %v", name, err)
	}
	if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return 128 + int(status.Signal())
	}
	return exitErr.ExitCode()
}
