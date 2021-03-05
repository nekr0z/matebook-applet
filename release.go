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
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	keyID      string = "F25E85CB21A79726"
	githubRepo string = "matebook-applet"
	maxKeep           = 10
)

func main() {
	cleanup := flag.Bool("c", false, "clean debian repository")
	flag.Parse()

	if *cleanup {
		if cleanRepo(githubRepo) {
			publishRepo(githubRepo)
		}
		os.Exit(0)
	}

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

	release := true
	// check if corresponding tag even exists
	res, err := getString("git", "ls-remote", "origin", gitVersion)
	if err != nil || res == "" {
		fmt.Println("Tag doesn't seem to exist, won't be releasing.")
		release = false
	}

	releaseGithub := true

	// build
	fmt.Println("building...")
	if err := runWithOutput("go", "run", "build.go", "-t", "-d"); err != nil {
		log.Fatalln(err)
	}

	debFilenames, err := filepath.Glob("./*.deb")
	if err != nil {
		log.Fatalln(err)
	}

	if release && releaseGithub {
		// release packages
		fileNames := []string{
			"matebook-applet-amd64-" + gitVersion + ".tar.gz",
		}
		fileNames = append(fileNames, debFilenames...)
		for _, fileName := range fileNames {
			args := []string{
				fileName,
				"release/",
			}
			cmd := exec.Command("cp", args...)
			if err := cmd.Run(); err != nil {
				fmt.Println("failed to copy", fileName)
			}
			fmt.Println(fileName, "copied to release/ successfully")
		}
	}

	// update debian repo
	if release && !(strings.Contains(version, "-")) {
		updateRepo(debFilenames)
	}
}

func getString(c string, a ...string) (string, error) {
	cmd := exec.Command(c, a...)
	b, err := cmd.CombinedOutput()
	return string(bytes.TrimSpace(b)), err
}

func runWithOutput(c string, a ...string) error {
	cmd := exec.Command(c, a...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return err
}

// use aptly and rsync to update debian repo
func updateRepo(filenames []string) {
	if updateLocalRepo(filenames) {
		publishRepo("matebook-applet")
	}
}

// update aptly local repo
func updateLocalRepo(filenames []string) bool {
	for _, filename := range filenames {
		cmd := exec.Command("aptly", "repo", "add", "matebook-applet", filename)
		if err := cmd.Run(); err != nil {
			fmt.Println("failed to add", filename, "to local aptly repo")
			return false
		}
	}
	return true
}
func publishRepo(repo string) {
	// need to prime GPG with passphrase for signing, because aptly can't really do that
	if err := runWithOutput("gpg", "--detach-sign", "--yes", "--passphrase", os.Getenv("GPG_PASSPHRASE"), "--pinentry-mode", "loopback", "-a", "-u", keyID, ".travis.yml"); err != nil {
		log.Fatalln(err)
	}

	if err := runWithOutput("aptly", "publish", "repo", repo); err != nil {
		log.Fatalln("failed to locally publish repo")
		return
	}
	fmt.Println("local repo update successful")
	usr, err := user.Current()
	if err != nil {
		log.Fatalln(err)
	}
	local := filepath.Join(usr.HomeDir, ".aptly/public/")
	if err := runWithOutput("rsync", "-r", "-v", "--del", local+"/", "evgeny@evgenykuznetsov.org:~/repository/"); err != nil {
		fmt.Printf("failed to rsync to evgenykuznetsov.org: %s", err)
	}
}

func cleanRepo(repo string) bool {
	repoContents := readRepo(repo)
	done := false

	for pkg, versions := range repoContents {
		var drop string
		if len(versions) > maxKeep {
			drop = fmt.Sprintf("%s (<= %s)", pkg, versions[maxKeep])
		} else if len(versions) > 1 {
			drop = fmt.Sprintf("%s (= %s)", pkg, versions[len(versions)-1])
		}
		if drop != "" {
			err := runWithOutput("aptly", "repo", "remove", repo, drop)
			if err != nil {
				log.Fatalln(err)
			}
			done = true
		}
	}
	return done
}

func readRepo(repo string) map[string][]string {
	contents := make(map[string][]string)
	cmd := exec.Command("aptly", "repo", "search", `-format={{.Package}} {{.Version}}`, repo, "Name")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return contents
	}

	r := bytes.NewReader(b)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		values := strings.Split(line, " ")
		if len(values) != 2 {
			continue
		}
		var versions []string
		versions = contents[values[0]]
		versions = append(versions, values[1])
		contents[values[0]] = versions
	}

	return contents
}
