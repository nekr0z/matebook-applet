// Copyright (C) 2019 Evgeny Kuznetsov (evgeny@kuznetsov.md)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along tihe this program. If not, see <https://www.gnu.org/licenses/>.

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
