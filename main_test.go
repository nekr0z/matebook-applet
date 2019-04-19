package main

import "testing"

type testval struct {
	status string
	state  string
	min    int
	max    int
}

var tests = []testval{
	{"REG[0xe4]==0x28\nREG[0xe5]==0x46\nbattery protection is on\nthresholds:\nminimum 40 %\nmaximum 70 %", "on", 40, 70},
}

func TestParseStatus(t *testing.T) {
	for _, test := range tests {
		state, min, max := parseStatus(test.status)
		if state != test.state || min != test.min || max != test.max {
			t.Error(
				"For:\n", test.status, "\nexpected", test.state, "got", state,
				"\nexpected min", test.min, "got", min,
				"\nexpected max", test.max, "got", max,
			)
		}
	}
}
