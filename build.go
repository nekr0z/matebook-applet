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
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"github.com/nekr0z/changelog"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type packFile struct {
	src string
	dst string
	mod os.FileMode
}

type distFile struct {
	src string
	dst string
}

var (
	appName  string = "matebook-applet"
	appID    string = "nekr0z.matebook-applet"
	filename string
	keyID    string = "F25E85CB21A79726"

	packFiles = []packFile{
		{src: "LICENSE", dst: "LICENSE", mod: 0644},
		{src: "README.md", dst: "README.md", mod: 0644},
		{src: "SOURCE.txt", dst: "SOURCE.txt", mod: 0644},
		{src: "matebook-applet.1", dst: "matebook-applet.1", mod: 0644},
		{src: "CHANGELOG.md", dst: "CHANGELOG.md", mod: 0644},
	}
	distFiles = []distFile{
		{src: "LICENSE", dst: "/usr/share/doc/matebook-applet/"},
		{src: "README.md", dst: "/usr/share/doc/matebook-applet/"},
		{src: "SOURCE.txt", dst: "/usr/share/doc/matebook-applet/"},
		{src: "matebook-applet.1", dst: "/usr/share/man/man1/"},
		{src: "matebook-applet.desktop", dst: "/usr/share/applications/"},
		{src: "assets/matebook-applet.png", dst: "/usr/share/icons/hicolor/512x512/apps/"},
	}
	bundleFiles = []packFile{
		{"LICENSE", "License/LICENSE.txt", 0644},
		{"README.md", "README.md", 0644},
		{"SOURCE.txt", "License/SOURCE.txt", 0644},
		{"assets/matebook-applet.png", "Resources/matebook-applet.png", 0644},
		{"matebook-applet.1", "manpage/matebook-applet.1", 0644},
	}
	debDeps = []string{
		"libappindicator3-1",
		"libc6",
		"libgtk-3-0 >= 3.10",
	}
	debRecs = []string{
		"huawei-wmi",
	}
)

func main() {
	sign := flag.Bool("s", false, "sign binary")
	tar := flag.Bool("t", false, "generate tar.gz")
	deb := flag.Bool("d", false, "build .deb")
	mac := flag.Bool("m", false, "generate MacOS App Bundle")
	flag.Parse()
	version := getVersion()
	btime := buildTime()
	if *tar {
		*sign = true
	}

	fmt.Printf("Building version %s\n", version)
	fmt.Println("Building as of", time.Unix(btime, 0))
	buildAssets(btime)
	buildBinary(version, btime)

	if *sign {
		signFile("matebook-applet", keyID)
	}

	if *tar {
		filename = "matebook-applet-amd64" + "-" + version
		buildTar()
		fmt.Println("archive", filename, "created")
	}

	if *deb {
		buildDeb(version)
	}

	if *mac {
		appBundle(version)
	}
}

func buildBinary(version string, t int64) {
	cmdline := fmt.Sprintf("go build -buildmode=pie -trimpath -ldflags=\"-buildid= -X main.version=%s\"", version)
	cmd := exec.Command("bash", "-c", cmdline)
	if runtime.GOOS == "darwin" {
		fmt.Println("Building for darwin, adding necessary flags...")
		cmd.Env = append(os.Environ(), "CGO_CFLAGS=-mmacosx-version-min=10.8")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	if err := cmd.Run(); err != nil {
		log.Fatalln("failed to build binary")
	}
	setFileTime("matebook-applet", t)
	packFiles = append(packFiles, packFile{"matebook-applet", "matebook-applet", 0755})
	distFiles = append(distFiles, distFile{"matebook-applet", "/usr/bin/"})
	bundleFiles = append(bundleFiles, packFile{appName, filepath.Join("MacOS", appName), 0755})
}

func buildAssets(t int64) {
	cmd := exec.Command("go", "run", "assets_generate.go")
	if err := cmd.Run(); err != nil {
		log.Fatalln("failed to rebuild assets")
	}
	setFileTime("assets.go", t)
}

func appBundle(version string) {
	cPath := filepath.Join(fmt.Sprintf("%s.app", appName), "Contents")
	if err := os.MkdirAll(cPath, 0777); err != nil {
		fmt.Println(err)
		return
	}
	info := struct {
		AppName string
		Version string
		ID      string
		Exec    string
		Icon    string
	}{
		appName,
		version,
		appID,
		fmt.Sprintf("MacOS/%s", appName),
		"Resources/matebook-applet.png",
	}

	t, err := template.ParseFiles("Info.plist")
	if err != nil {
		fmt.Println(err)
		return
	}
	plist, err := os.Create(filepath.Join(cPath, "Info.plist"))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer plist.Close()
	if err := t.Execute(plist, info); err != nil {
		fmt.Println(err)
		return
	}

	for _, f := range bundleFiles {
		f.dst = filepath.Join(cPath, f.dst)
		if err := copyFile(f); err != nil {
			fmt.Println(err)
		}
	}
}

func buildDeb(ver string) {
	maintainer := changelog.Maintainer{Name: "Evgeny Kuznetsov", Email: "evgeny@kuznetsov.md"}
	var clok bool
	fd, err := os.Open("CHANGELOG.md")
	if err != nil {
		fmt.Printf("error opening changelog: %s\n", err)
	} else {
		defer fd.Close()
		cl, err := changelog.ParseMd(fd)
		if err != nil {
			fmt.Printf("error parsing changelog: %s\n", err)
		}
		cmd := exec.Command("git", "tag", "-l", `--format=%(creatordate:iso)|%(refname:short)`)
		var bb bytes.Buffer
		out := bufio.NewWriter(&bb)
		cmd.Stdout = out
		if err := cmd.Run(); err != nil {
			fmt.Printf("failed to read tag times: %s", err)
		} else {
			out.Flush()
			scanner := bufio.NewScanner(&bb)
			for scanner.Scan() {
				line := scanner.Text()
				s := strings.Split(line, "|")
				if len(s) == 2 {
					d, err := time.Parse("2006-01-02 15:04:05 -0700", s[0])
					if err == nil {
						ver, err := changelog.ToVersion(strings.TrimPrefix(s[1], "v"))
						if err == nil {
							rel := cl[ver]
							rel.Date = d
							cl[ver] = rel
						}
					}
				}
			}
		}
		for v, rel := range cl {
			rel.Maintainer = maintainer
			cl[v] = rel
		}
		b, err := cl.Debian("matebook-applet")
		if err != nil {
			fmt.Printf("error converting changelog to Debian format: %s\n", err)
		}
		clDeb, err := os.Create("debian.changelog")
		if err != nil {
			fmt.Printf("error creating Debian changelog: %s\n", err)
		} else {
			defer clDeb.Close()
			_, err := clDeb.Write(b)
			if err != nil {
				fmt.Printf("error writing Debian changelog: %s\n", err)
			}
			clDeb.Sync()
			clok = true
		}
	}
	ver = strings.Replace(strings.TrimPrefix(ver, "v"), "-", "~", 1)
	args := []string{
		"-f",
		"-t", "deb",
		"-s", "dir",
		"-n", appName,
		"-v", ver,
		"-m", fmt.Sprintf("%s <%s>", maintainer.Name, maintainer.Email),
		"--vendor", fmt.Sprintf("%s <%s>", maintainer.Name, maintainer.Email),
		"--category", "misc",
		"--description", "System tray applet for Huawei MateBook\nAllows one to control Huawei MateBook features,\nlike Fn-Lock and Battery Protection settings, via GUI.",
		"--url", "https://github.com/nekr0z/matebook-applet",
		"--license", "GPL-3",
		"--deb-priority", "optional",
	}
	if clok {
		args = append(args, "--deb-changelog", "debian.changelog")
	}
	for _, dep := range debDeps {
		args = append(args, "-d", dep)
	}
	for _, rec := range debRecs {
		args = append(args, "--deb-recommends", rec)
	}
	for _, file := range distFiles {
		arg := file.src + "=" + file.dst
		args = append(args, arg)
	}
	cmd := exec.Command("fpm", args...)
	if err := cmd.Run(); err != nil {
		fmt.Println(err)
		log.Fatalln("failed to build .deb")
	}
	fmt.Println(".deb package created")
}

func buildTar() {
	for i := range packFiles {
		packFiles[i].dst = filename + "/" + packFiles[i].dst
	}
	filename = filename + ".tar.gz"
	fd, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	gw, err := gzip.NewWriterLevel(fd, gzip.BestCompression)
	if err != nil {
		log.Fatal(err)
	}
	tw := tar.NewWriter(gw)
	for _, f := range packFiles {
		sf, err := os.Open(f.src)
		if err != nil {
			log.Fatal(err)
		}
		info, err := sf.Stat()
		if err != nil {
			log.Fatal(err)
		}
		h := &tar.Header{
			Name:    f.dst,
			Size:    info.Size(),
			Mode:    int64(f.mod),
			ModTime: info.ModTime(),
		}
		err = tw.WriteHeader(h)
		if err != nil {
			log.Fatal(err)
		}
		_, err = io.Copy(tw, sf)
		if err != nil {
			log.Fatal(err)
		}
		sf.Close()
	}
	err = tw.Close()
	if err != nil {
		log.Fatal(err)
	}
	err = gw.Close()
	if err != nil {
		log.Fatal(err)
	}
	err = fd.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func copyFile(f packFile) (err error) {
	src, err := os.Open(f.src)
	if err != nil {
		return
	}
	defer src.Close()

	if err = os.MkdirAll(filepath.Dir(f.dst), 0777); err != nil {
		return
	}
	dst, e := os.Create(f.dst)
	if e != nil {
		return e
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return
	}

	return dst.Chmod(f.mod)
}

func setFileTime(f string, t int64) {
	cmd := exec.Command("touch", "-t", fmt.Sprint(time.Unix(t, 0).Format("200601021504.05")), f)
	if err := cmd.Run(); err != nil {
		log.Fatalln("failed to set time on", f)
	}
}

func signFile(f string, k string) {
	cmd := exec.Command("gpg", "--detach-sign", "--yes", "--passphrase", os.Getenv("GPG_PASSPHRASE"), "--pinentry-mode", "loopback", "-a", "-u", k, f)
	cmd.Stderr = os.Stdout
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		fmt.Println("signing", f, "failed")
		filename = filename + "-unsigned"
	} else {
		fmt.Println(f, "successfully signed with key", k)
		packFiles = append(packFiles, packFile{"matebook-applet.asc", "matebook-applet.asc", 0644})
	}
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

func buildTime() int64 {
	s, err := getString("git", "show", "-s", "--format=%ct")
	if err == nil {
		if i, e := strconv.ParseInt(s, 10, 64); e == nil {
			return i
		}
	}
	return time.Now().Unix()
}
