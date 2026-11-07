package configsock

import (
	"strings"
	"testing"
)

func TestReadinessSocketPath(t *testing.T) {
	got := ReadinessSocketPath("/run/sandbox", "sandbox-id")
	if got != "/run/sandbox/sandboxes/sandbox-id/ready.sock" {
		t.Fatalf("ReadinessSocketPath = %q", got)
	}
}

func TestReadReadinessTiming(t *testing.T) {
	wire := "control_ready\nready\t{\"version\":1,\"sandbox_ctl_start_unix_ns\":100,\"ch_start_unix_ns\":200,\"ch_resume_done_unix_ns\":300,\"sandbox_init_ack_unix_ns\":400}\n"
	timing, err := ReadReadinessTiming(strings.NewReader(wire))
	if err != nil {
		t.Fatal(err)
	}
	if timing == nil || timing.Version != 1 || timing.SandboxCtlStartUnixNs != 100 || timing.CHStartUnixNs != 200 || timing.CHResumeDoneUnixNs != 300 || timing.SandboxInitAckUnixNs != 400 {
		t.Fatalf("timing = %+v, want all wire points", timing)
	}
	legacy, err := ReadReadinessTiming(strings.NewReader("control_ready\nready\n"))
	if err != nil || legacy != nil {
		t.Fatalf("legacy timing = %+v, err=%v, want nil timing and nil error", legacy, err)
	}
}

func TestReadReadiness(t *testing.T) {
	tests := []struct {
		name    string
		wire    string
		wantErr bool
	}{
		{name: "exact", wire: "control_ready\nready\n"},
		{name: "EOF before control", wire: "", wantErr: true},
		{name: "EOF before ready", wire: "control_ready\n", wantErr: true},
		{name: "unknown", wire: "unknown\nready\n", wantErr: true},
		{name: "duplicate", wire: "control_ready\ncontrol_ready\n", wantErr: true},
		{name: "reverse", wire: "ready\ncontrol_ready\n", wantErr: true},
		{name: "trailing event", wire: "control_ready\nready\nready\n", wantErr: true},
		{name: "unterminated control", wire: "control_ready", wantErr: true},
		{name: "line too long", wire: strings.Repeat("x", readinessLineLimit+1) + "\nready\n", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ReadReadiness(strings.NewReader(tt.wire))
			if (err != nil) != tt.wantErr {
				t.Fatalf("ReadReadiness(%q) error = %v, want error=%t", tt.wire, err, tt.wantErr)
			}
		})
	}
}
