package runtime

import (
	"context"
	"io"
	"os/exec"
)

// Transport abstracts the byte channel between host and VM so that the
// session logic is identical for real subprocesses (nilosd / nil serve) and
// for in-process fakes used by tests and the built-in app host.
type Transport interface {
	Reader() io.Reader
	Writer() io.Writer
	Close() error
	Wait() error
	Pid() int
}

// execTransport runs an external worker binary.
type execTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

// NewExecTransport starts bin with args, wiring stdin/stdout for the protocol
// and stderr to the provided writer (may be nil).
func NewExecTransport(ctx context.Context, bin string, args []string, dir string, env []string, stderr io.Writer) (*execTransport, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if env != nil {
		cmd.Env = env
	}
	if stderr != nil {
		cmd.Stderr = stderr
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &execTransport{cmd: cmd, stdin: stdin, stdout: stdout}, nil
}

func (t *execTransport) Reader() io.Reader { return t.stdout }
func (t *execTransport) Writer() io.Writer { return t.stdin }
func (t *execTransport) Close() error {
	_ = t.stdin.Close()
	if t.cmd.Process != nil {
		return t.cmd.Process.Kill()
	}
	return nil
}
func (t *execTransport) Wait() error {
	if t.cmd.Process != nil {
		return t.cmd.Wait()
	}
	return nil
}
func (t *execTransport) Pid() int {
	if t.cmd.Process != nil {
		return t.cmd.Process.Pid
	}
	return 0
}

// pipeTransport hosts an in-process VM implementation.
type pipeTransport struct {
	hostRead  *io.PipeReader
	hostWrite *io.PipeWriter
	vm        io.Closer
	done      chan error
}

// NewPipeTransport connects the session to an in-process VM. run receives the
// VM-side reader/writer and should block until the VM stops.
func NewPipeTransport(run func(vmIn io.Reader, vmOut io.Writer) error) *pipeTransport {
	vmToHostR, vmToHostW := io.Pipe()
	hostToVMR, hostToVMW := io.Pipe()
	done := make(chan error, 1)
	vm := &pipeCloser{w: vmToHostW, r: hostToVMR, done: done}
	go func() {
		err := run(hostToVMR, vmToHostW)
		_ = vmToHostW.Close()
		done <- err
	}()
	return &pipeTransport{
		hostRead:  vmToHostR,
		hostWrite: hostToVMW,
		vm:        vm,
		done:      done,
	}
}

func (t *pipeTransport) Reader() io.Reader { return t.hostRead }
func (t *pipeTransport) Writer() io.Writer { return t.hostWrite }
func (t *pipeTransport) Close() error {
	_ = t.hostWrite.Close()
	_ = t.vm.Close()
	return nil
}
func (t *pipeTransport) Wait() error {
	select {
	case err := <-t.done:
		return err
	default:
		return nil
	}
}
func (t *pipeTransport) Pid() int { return 0 }

type pipeCloser struct {
	w    *io.PipeWriter
	r    *io.PipeReader
	done chan error
}

func (p *pipeCloser) Close() error {
	_ = p.w.Close()
	_ = p.r.Close()
	return nil
}
