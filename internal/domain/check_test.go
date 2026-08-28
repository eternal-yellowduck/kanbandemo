package domain

import "testing"

func TestCheckLifecycleAndRevision(t *testing.T) {
	check, err := NewCheck("check-1", "task-1", CheckAutomatedTest, true, 1, testTime(0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := check.Pass(metaForTest(0)); err == nil {
		t.Fatal("pending check passed without running")
	}
	if _, err := check.AssignRun("run-check-1", metaForTest(0)); err != nil {
		t.Fatal(err)
	}
	if _, err := check.Pass(metaForTest(1)); err != nil {
		t.Fatal(err)
	}
	if check.Status != CheckPassed || check.Revision != 1 {
		t.Fatalf("status/revision = %s/%d", check.Status, check.Revision)
	}
}
