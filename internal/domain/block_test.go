package domain

import "testing"

func TestHumanInputBlockCanOnlyBeAnsweredOnce(t *testing.T) {
	block, err := NewHumanInputBlock("block-1", "task-1", "run-1", "需要哪些页面？", []string{"首页", "关于"}, testTime(0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := block.SubmitAnswer("首页", metaForTest(0)); err != nil {
		t.Fatal(err)
	}
	if block.Answer == nil || block.ResolvedAt == nil {
		t.Fatal("answer was not persisted in block")
	}
	if _, err := block.SubmitAnswer("关于", metaForTest(1)); err == nil {
		t.Fatal("answered block accepted a second answer")
	}
}
