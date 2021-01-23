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
	got := threshToHexArg(60, 80)
	want := "1346109440"
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestGetThreshFromLog(t *testing.T) {
	logInit(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	log := "Read from the log: Filtering the log data using \"senderImagePath CONTAINS \"ACPIDebug\"\"\nTimestamp                       Thread     Type        Activity             PID    TTL  \n2021-01-23 18:25:38.021007+0100 0x88f      Default     0x0                  0      0    kernel: (ACPIDebug) ACPIDebug: \"View Thresholds\"\n2021-01-23 18:25:38.025894+0100 0x88f      Default     0x0                  0      0    kernel: (ACPIDebug) ACPIDebug: { \"Reading (hexadecimal values):\", 0x28, 0x3c, }"
	min, max, err := getThreshFromLog(log)
	if err != nil {
		t.Fatal(err)
	}
	if min != 40 || max != 60 {
		t.Fatalf("got %d-%d, want 40-60", min, max)
	}
}
