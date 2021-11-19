// Copyright (C) 2019-2020 Evgeny Kuznetsov (evgeny@kuznetsov.md)
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

//+build ignore

package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
)

func main() {
	ver := getVersion()
	fmt.Println(ver)
}

func getVersion() string {
	desc, err1 := getString("git", "describe", "--always", "--dirty")
	br, err2 := getString("git", "symbolic-ref", "--short", "-q", "HEAD")
	if err1 == nil && (br == "" || err2 == nil) {
		return parseVersion(desc, br)
	}
	return "unknown"
}

func parseVersion(desc, branch string) string {
	descRe := regexp.MustCompile(`^v?(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))??(?:\-(?P<after>0|[1-9]\d*)\-(?P<commit>g[0-9a-f]{5,15}))?(?:\-(?P<dirty>dirty))??$`)
	if !descRe.MatchString(desc) {
		return "unknown"
	}

	ver := descRe.ReplaceAllString(desc, "${major}.${minor}.${patch}")
	pre := descRe.ReplaceAllString(desc, "${prerelease}")
	after := descRe.ReplaceAllString(desc, "${after}")
	commit := descRe.ReplaceAllString(desc, "${commit}")
	dirty := descRe.ReplaceAllString(desc, "${dirty}")

	if pre != "" {
		ver = fmt.Sprintf("%s-%s", ver, pre)
	}

	if (after != "" || dirty != "") && branch != "master" && branch != "" {
		ver = fmt.Sprintf("%s.%s", ver, branch)
	}

	if after != "" {
		ver = fmt.Sprintf("%s.%s", ver, after)
	}

	if dirty != "" {
		ver = fmt.Sprintf("%s.%s", ver, "dirty")
	}

	if commit != "" {
		ver = fmt.Sprintf("%s+%s", ver, commit)
	}

	return ver
}

func getString(c string, a ...string) (string, error) {
	cmd := exec.Command(c, a...)
	b, err := cmd.CombinedOutput()
	return string(bytes.TrimSpace(b)), err
}
