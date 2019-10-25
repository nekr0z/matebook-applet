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
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// +build ignore

package main

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	testCases := []struct {
		desc   string
		branch string
		v      string
	}{
		{"v2.3.0-6-g30208ef-dirty", "master", "2.3.0.6.dirty+g30208ef"},
		{"1.2.3", "master", "1.2.3"},
		{"43.765.39-beta", "dev", "43.765.39-beta"},
		{"67.1.1-3-g0101afde", "dev", "67.1.1.dev.3+g0101afde"},
		{"v2.0.0-dirty", "dev", "2.0.0.dev.dirty"},
		{"1.0.1-dirty", "master", "1.0.1.dirty"},
		{"3.2.1-beta.2-4-g666eafde", "master", "3.2.1-beta.2.4+g666eafde"},
		{"afd", "fake", "unknown"},
	}

	for _, tc := range testCases {
		s := parseVersion(tc.desc, tc.branch)
		if s != tc.v {
			t.Errorf("for %s on %s want %s, got %s", tc.desc, tc.branch, tc.v, s)
		}
	}
}
