package orch

import (
	"testing"
	"time"

	"github.com/kuasar-sandbox/orchestrator/internal/types"
)

func TestLaunchTimelineExportDoesNotDoubleCountProfileTail(t *testing.T) {
	base := time.Unix(0, 0)
	points := map[string]time.Time{
		"accepted":                base,
		"node_prepare_done":       base.Add(100 * time.Millisecond),
		"runner_assign_start":     base.Add(100 * time.Millisecond),
		"runner_commit_start":     base.Add(150 * time.Millisecond),
		"runner_commit_done":      base.Add(200 * time.Millisecond),
		"runner_handoff_done":     base.Add(250 * time.Millisecond),
		"sandbox_ctl_start":       base.Add(300 * time.Millisecond),
		"ch_start":                base.Add(400 * time.Millisecond),
		"ch_resume_done":          base.Add(500 * time.Millisecond),
		"sandbox_init_ack":        base.Add(600 * time.Millisecond),
		"runtime_ready_received":  base.Add(700 * time.Millisecond),
		"envd_init_attempt1_done": base.Add(750 * time.Millisecond),
		"envd_init_done":          base.Add(800 * time.Millisecond),
		"launch_done":             base.Add(1000 * time.Millisecond),
	}
	timing := (&launchTimeline{profile: types.ProfileE2B, points: points}).export(1000)
	if _, ok := timing.PhasesMs["runtime_ready_to_launch_done"]; ok {
		t.Fatal("e2b timing double-counted runtime_ready_to_launch_done")
	}
	if got := timing.AccountedMs; got != 1000 {
		t.Fatalf("accountedMs = %v, want 1000", got)
	}
	if timing.UnaccountedMs != 0 {
		t.Fatalf("unaccountedMs = %v, want 0", timing.UnaccountedMs)
	}
	if got := timing.PhasesMs["envd_attempt1"]; got != 50 {
		t.Fatalf("envd_attempt1 = %v, want 50", got)
	}
	if got := timing.PhasesMs["envd_retry_tail"]; got != 50 {
		t.Fatalf("envd_retry_tail = %v, want 50", got)
	}
	delete(points, "ch_resume_done")
	delete(points, "sandbox_init_ack")
	delete(points, "envd_init_done")
	bare := (&launchTimeline{profile: types.ProfileBare, points: points}).export(1000)
	if _, ok := bare.PhasesMs["runtime_ready_to_envd_init_done"]; ok {
		t.Fatal("bare timing contains envd phase")
	}
	if _, ok := bare.PhasesMs["runtime_ready_to_launch_done"]; !ok {
		t.Fatal("bare timing lacks runtime_ready_to_launch_done")
	}
}
