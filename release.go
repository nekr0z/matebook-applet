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

// +build ignore

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

var (
	githubRepo string = "matebook-applet"
)

func main() {
	flag.Parse()
	if os.Getenv("GITHUB_TOKEN") == "" {
		log.Fatalln("github token not available, can't work")
	}
	if os.Getenv("GITHUB_USER") == "" {
		log.Fatalln("github user not set, can't work")
	}

	var version string
	if flag.NArg() == 0 {
		// release latest version
		res, err := getString("git", "describe")
		if err != nil {
			log.Fatalln(err)
		}
		r := strings.Split(res, "-")
		version = r[0]
	} else {
		version = flag.Arg(0)
	}
	log.Println("Trying to release version", version)
	process(version)
}

func process(version string) {
	// TODO check version sanity and format

	var gitVersion string
	if strings.HasPrefix(version, "v") {
		gitVersion = version
		version = version[1:]
	}

	// check if corresponding tag even exists
	res, err := getString("git", "ls-remote", "origin", gitVersion)
	if err != nil || res == "" {
		log.Fatalln("Tag doesn't seem to exist, giving up.")
	}

	// check if this version is already released
	res, err = getString("gothub", "info", "-r", githubRepo, "-t", gitVersion)
	if err == nil || res != "error: could not find the release corresponding to tag "+gitVersion {
		log.Fatalln("Something wrong. Already released? Giving up.")
	}

	// git checkout requested version
	checkout(gitVersion)

	// build
	fmt.Println("building...")
	cmd := exec.Command("go", "run", "build.go", "-t", "-d")
	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
	}

	// get release description from tag
	desc, err := getString("git", "tag", "-ln", "--format=%(contents)", gitVersion)
	if err != nil {
		log.Fatalln(err)
	}
	descStrings := strings.Split(desc, "\n")
	for i, descString := range descStrings {
		if descString == "-----BEGIN PGP SIGNATURE-----" {
			descStrings = descStrings[:i]
			break
		}
	}
	desc = strings.Join(descStrings, "\n")

	// release version
	args := []string{
		"release",
		"-r", githubRepo,
		"-t", gitVersion,
		"-n", version,
		"-d", desc,
	}
	cmd = exec.Command("gothub", args...)
	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
	}

	// release packages
	fileNames := []string{
		"matebook-applet-amd64-" + gitVersion + ".tar.gz",
		"matebook-applet_" + version + "_amd64.deb",
	}
	for _, fileName := range fileNames {
		args := []string{
			"upload",
			"-r", githubRepo,
			"-t", gitVersion,
			"-n", fileName,
			"-f", fileName,
		}
		cmd := exec.Command("gothub", args...)
		if err := cmd.Run(); err != nil {
			fmt.Println("failed to upload", fileName)
		}
		fmt.Println(fileName, "uploaded successfully")
	}

	// git checkout back to master
	checkout("master")
}

func checkout(version string) {
	fmt.Printf("trying to git checkout %s...\n", version)
	cmd := exec.Command("git", "checkout", version)
	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
	}
}

func getString(c string, a ...string) (string, error) {
	cmd := exec.Command(c, a...)
	b, err := cmd.CombinedOutput()
	return string(bytes.TrimSpace(b)), err
}
