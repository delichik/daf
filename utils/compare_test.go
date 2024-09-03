package utils

import "testing"

func TestDeepCompare(t *testing.T) {
	type s struct {
		A string
		B []string
		C map[string]int
	}

	a := &s{
		A: "1",
		B: []string{"1", "2"},
		C: map[string]int{
			"1": 1,
			"2": 1,
			"3": 1,
		},
	}
	b := &s{
		A: "1",
		B: []string{"1", "2"},
		C: map[string]int{
			"1": 1,
			"2": 1,
			"3": 1,
		},
	}
	if !DeepCompare(a, b) {
		t.FailNow()
	}
}
