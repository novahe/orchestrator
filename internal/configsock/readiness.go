package configsock

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/kuasar-sandbox/orchestrator/internal/nodepath"
)

const (
	ReadinessControlReady = "control_ready"
	ReadinessReady        = "ready"

	readinessSocketName = "ready.sock"
	readinessLineLimit  = 4096
)

// ReadinessTiming is the optional sandbox-ctl timing payload attached to the
// ready event. Unix nanoseconds are used so node-ctl can correlate the child
// process points with its own wall clock. Zero fields are intentionally valid
// for cold image boots, which have no CH resume/restore ACK phase.
type ReadinessTiming struct {
	Version               int   `json:"version"`
	SandboxCtlStartUnixNs int64 `json:"sandbox_ctl_start_unix_ns"`
	CHStartUnixNs         int64 `json:"ch_start_unix_ns,omitempty"`
	CHResumeDoneUnixNs    int64 `json:"ch_resume_done_unix_ns,omitempty"`
	SandboxInitAckUnixNs  int64 `json:"sandbox_init_ack_unix_ns,omitempty"`
}

// ReadinessSocketPath is the one-shot runtime readiness socket shared by the
// orchestrator and node-ctl. It deliberately is not part of LaunchSpec: the
// assigned SandboxID and node RunRoot already define the SandboxRunDir for both
// processes.
func ReadinessSocketPath(runRoot, sandboxID string) string {
	return filepath.Join(nodepath.SandboxRunDir(runRoot, sandboxID), readinessSocketName)
}

// ReadReadiness consumes the complete sandbox-ctl readiness stream. The wire is
// deliberately tiny and closed: exactly control_ready, ready, then EOF. Keeping
// the parser here makes the UDS bridge and builder pipe enforce one protocol.
func ReadReadiness(r io.Reader) error {
	_, err := ReadReadinessTiming(r)
	return err
}

// ReadReadinessTiming consumes the readiness stream and returns the optional
// timing payload carried by the ready line. It accepts the historical plain
// `ready` line for mixed-version node-ctl/sandbox-ctl deployments.
func ReadReadinessTiming(r io.Reader) (*ReadinessTiming, error) {
	br := bufio.NewReaderSize(r, readinessLineLimit)
	for i, want := range []string{ReadinessControlReady, ReadinessReady} {
		got, err := readReadinessLine(br)
		if err != nil {
			return nil, fmt.Errorf("read readiness event %d (%s): %w", i+1, want, err)
		}
		if i == 1 {
			if got == want {
				break
			}
			if !strings.HasPrefix(got, want+"\t") {
				return nil, fmt.Errorf("readiness event %d = %q, want %q", i+1, got, want)
			}
			var timing ReadinessTiming
			if err := json.Unmarshal([]byte(strings.TrimPrefix(got, want+"\t")), &timing); err != nil {
				return nil, fmt.Errorf("decode readiness timing: %w", err)
			}
			if timing.Version != 1 {
				return nil, fmt.Errorf("unsupported readiness timing version %d", timing.Version)
			}
			if err := consumeReadinessEOF(br); err != nil {
				return nil, err
			}
			return &timing, nil
		}
		if got != want {
			return nil, fmt.Errorf("readiness event %d = %q, want %q", i+1, got, want)
		}
	}
	if err := consumeReadinessEOF(br); err != nil {
		return nil, err
	}
	return nil, nil
}

func consumeReadinessEOF(br *bufio.Reader) error {
	b, err := br.ReadByte()
	switch {
	case errors.Is(err, io.EOF):
		return nil
	case err != nil:
		return fmt.Errorf("read readiness EOF: %w", err)
	default:
		return fmt.Errorf("unexpected readiness data after ready (first byte %#x)", b)
	}
}

func readReadinessLine(br *bufio.Reader) (string, error) {
	line, err := br.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) {
		return "", fmt.Errorf("readiness line exceeds %d bytes", readinessLineLimit)
	}
	if err != nil {
		if errors.Is(err, io.EOF) {
			return "", io.ErrUnexpectedEOF
		}
		return "", err
	}
	if len(line) > readinessLineLimit {
		return "", fmt.Errorf("readiness line exceeds %d bytes", readinessLineLimit)
	}
	return string(line[:len(line)-1]), nil
}
