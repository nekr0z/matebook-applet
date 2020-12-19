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

import (
	"testing"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func TestParseStatus(t *testing.T) {
	type testval struct {
		status string
		state  string
		min    int
		max    int
	}

	var tests = []testval{
		{"REG[0xe4]==0x28\nREG[0xe5]==0x46\nbattery protection is on\nthresholds:\nminimum 40 %\nmaximum 70 %", "on", 40, 70},
	}

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

func TestBtobb(t *testing.T) {
	type testval struct {
		b bool
		s string
	}

	var tests = []testval{
		{true, "1"},
		{false, "0"},
	}

	for _, test := range tests {
		result := string(btobb(test.b))
		if result != test.s {
			t.Error("For", test.b, "expected", test.s, "got", result)
		}
	}
}

func TestGetStatus(t *testing.T) {
	bundle := i18nPrepare()
	localizer = i18n.NewLocalizer(bundle, "en-US")
	config.thresh = threshDriver{&mockDriver{}}

	tests := map[string]struct {
		min, max int
		want     string
	}{
		"0 0": {
			min:  0,
			max:  0,
			want: "Battery protection is OFF",
		},
		"0 100": {
			min:  0,
			max:  100,
			want: "Battery protection is OFF",
		},
		"travel": {
			min:  95,
			max:  100,
			want: "Battery protection mode: TRAVEL",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			config.thresh.set(tc.min, tc.max)
			got := getStatus()

			if got != tc.want {
				t.Fatalf("want: %v, got: %v", tc.want, got)
			}
		})
	}
}

type mockDriver struct {
	vMin, vMax int
}

func (drv *mockDriver) get() (min, max int, err error) {
	return drv.vMin, drv.vMax, nil
}

func (drv *mockDriver) write(min, max int) error {
	drv.vMin = min
	drv.vMax = max
	return nil
}
