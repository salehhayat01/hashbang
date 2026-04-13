package main

import "testing"

func TestCMSBasic(t *testing.T) {
	cms := NewCMS(4, 2000)

	cms.Add("a")
	cms.Add("a")
	cms.Add("b")

	if cms.Estimate("a") < 2 {
		t.Errorf("expected at least 2, got %d", cms.Estimate("a"))
	}

	if cms.Estimate("b") < 1 {
		t.Errorf("expected at least 1, got %d", cms.Estimate("b"))
	}
}
func TestCMSMerge(t *testing.T) {
	c1 := NewCMS(4, 2000)
	c2 := NewCMS(4, 2000)

	c1.Add("x")
	c1.Add("x")

	c2.Add("x")
	c2.Add("y")

	c1.Merge(c2)

	if c1.Estimate("x") < 3 {
		t.Errorf("expected >= 3, got %d", c1.Estimate("x"))
	}

	if c1.Estimate("y") < 1 {
		t.Errorf("expected >= 1, got %d", c1.Estimate("y"))
	}
}