// Copyright (C) 2021 Evgeny Kuznetsov (evgeny@kuznetsov.md)
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
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// +build darwin

package main

import (
	"os"
	"testing"
)

func TestThreshToHexArg(t *testing.T) {
	tests := map[string]struct {
		min  int
		max  int
		want string
	}{
		"60-80": {60, 80, "1346109440"},
		"0-100": {0, 100, "1677721600"},
		"10-15": {10, 15, "252313600"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := threshToHexArg(tc.min, tc.max)
			if got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestGetThreshFromLog(t *testing.T) {
	tests := map[string]struct {
		log string
		min int
		max int
		err error
	}{
		"40-60":   {"Read from the log: Filtering the log data using \"senderImagePath CONTAINS \"ACPIDebug\"\"\nTimestamp                       Thread     Type        Activity             PID    TTL  \n2021-01-23 18:25:38.021007+0100 0x88f      Default     0x0                  0      0    kernel: (ACPIDebug) ACPIDebug: \"View Thresholds\"\n2021-01-23 18:25:38.025894+0100 0x88f      Default     0x0                  0      0    kernel: (ACPIDebug) ACPIDebug: { \"Reading (hexadecimal values):\", 0x28, 0x3c, }", 40, 60, nil},
		"strange": {"kernel: (ACPIDebug) ACPIDebug: { \"Reading (hexadecimal values):\", 0x40, 0x6, }", 64, 6, nil},
	}
	logInit(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			min, max, err := getThreshFromLog(tc.log)
			if err != tc.err {
				t.Fatalf("want: %s, got: %s", tc.err, err)
			}
			if min != tc.min || max != tc.max {
				t.Fatalf("got %d-%d, want %d-%d", min, max, tc.min, tc.max)
			}
		})
	}
}
