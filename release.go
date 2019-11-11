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
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

var (
	githubRepo string = "matebook-applet"
)

func main() {
	if os.Getenv("GITHUB_TOKEN") == "" {
		log.Fatalln("github token not available, can't work")
	}
	if os.Getenv("GITHUB_USER") == "" {
		log.Fatalln("github user not set, can't work")
	}

	version, err := getString("git", "describe")
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Trying to release version", version)
	process(version)
}

func process(version string) {
	gitVersion := version
	version = strings.TrimPrefix(version, "v")

	release := true
	// check if corresponding tag even exists
	res, err := getString("git", "ls-remote", "origin", gitVersion)
	if err != nil || res == "" {
		fmt.Println("Tag doesn't seem to exist, won't be releasing.")
		release = false
	}

	releaseGithub := true
	// check if this version is already released
	res, err = getString("gothub", "info", "-r", githubRepo, "-t", gitVersion)
	if err == nil || res != "error: could not find the release corresponding to tag "+gitVersion {
		fmt.Println("Something wrong. Already released? Won't be pushing to Github.")
		releaseGithub = false
	}

	// build
	fmt.Println("building...")
	cmd := exec.Command("go", "run", "build.go", "-t", "-d")
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
	}

	debFilenames, err := filepath.Glob("*.deb")
	if err != nil {
		log.Fatalln(err)
	}

	if release && releaseGithub {
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
		}
		fileNames = append(fileNames, debFilenames...)
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
	}

	// update debian repo
	if release {
		updateRepo(debFilenames)
	}
}

func getString(c string, a ...string) (string, error) {
	cmd := exec.Command(c, a...)
	b, err := cmd.CombinedOutput()
	return string(bytes.TrimSpace(b)), err
}

// use aptly and rsync to update debian repo
func updateRepo(filenames []string) {
	if updateLocalRepo(filenames) {
		usr, err := user.Current()
		if err != nil {
			log.Fatalln(err)
		}
		local := filepath.Join(usr.HomeDir, ".aptly/public/")
		cmd := exec.Command("rsync", "-r", "-v", "--del", local, "nekr0z@evgenykuznetsov.org:~/repository/")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stdout
		if err := cmd.Run(); err != nil {
			fmt.Printf("failed to rsync to evgenykuznetsov.org: %s", err)
		}
	}
}

// update aptly local repo
func updateLocalRepo(filenames []string) bool {
	for _, filename := range filenames {
		cmd := exec.Command("aptly", "repo", "add", "matebook-applet", "*.deb")
		if err := cmd.Run(); err != nil {
			fmt.Println("failed to add", filename, "to local aptly repo")
			return false
		}
	}
	cmd := exec.Command("aptly", "publish", "repo", "matebook-applet")
	if err := cmd.Run(); err != nil {
		fmt.Println("failed to locally publish repo")
		return false
	}
	fmt.Println("local repo update successful")
	return true
}
