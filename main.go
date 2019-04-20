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

//go:generate go run assets_generate.go

package main

import (
	"bytes"
	"fmt"
	"github.com/getlantern/systray"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	logTrace     *log.Logger
	logInfo      *log.Logger
	logWarning   *log.Logger
	logError     *log.Logger
	scriptBatpro bool
	scriptFnlock bool
)

func logInit(
	traceHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {
	logTrace = log.New(traceHandle,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	logInfo = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	logWarning = log.New(warningHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	logError = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}

func main() {
	logInit(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	scriptBatpro = checkBatpro()
	scriptFnlock = checkFnlock()
	if scriptBatpro || scriptFnlock {
		systray.Run(onReady, onExit)
	} else {
		logError.Println("The required script is not properly installed, see README.md#installation-and-setup for instructions")
	}
}

func onReady() {
	logTrace.Println("Setting up menu...")
	systray.SetIcon(getIcon("/Iconsmind-Outline-Battery-Charge.ico"))
	mStatus := systray.AddMenuItem("", "")
	systray.AddSeparator()
	mOff := systray.AddMenuItem("OFF", "Switch off battery protection")
	mTravel := systray.AddMenuItem("TRAVEL (95%-100%)", "Set battery protection to TRAVEL")
	mOffice := systray.AddMenuItem("OFFICE (70%-90%)", "Set battery protection to OFFICE")
	mHome := systray.AddMenuItem("HOME (40%-70%)", "Set battery protection to HOME")
	systray.AddSeparator()
	mFnlock := systray.AddMenuItem("", "")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the applet")
	if scriptBatpro == false {
		mStatus.Hide()
		mOff.Hide()
		mTravel.Hide()
		mOffice.Hide()
		mHome.Hide()
	} else {
		mStatus.SetTitle(getStatus())
	}
	if scriptFnlock == false {
		mFnlock.Hide()
	} else {
		mFnlock.SetTitle(getFnlockStatus())
	}
	logTrace.Println("Menu is now ready")
	go func() {
		for {
			select {
			case <-mStatus.ClickedCh:
				logTrace.Println("Got a click on BP status")
				mStatus.SetTitle(getStatus())
			case <-mOff.ClickedCh:
				logTrace.Println("Got a click on BP OFF")
				setBatproOff()
				mStatus.SetTitle(getStatus())
			case <-mTravel.ClickedCh:
				logTrace.Println("Got a click on BP TRAVEL")
				setThresholds(95, 100)
				mStatus.SetTitle(getStatus())
			case <-mOffice.ClickedCh:
				logTrace.Println("Got a click on BP OFFICE")
				setThresholds(70, 90)
				mStatus.SetTitle(getStatus())
			case <-mHome.ClickedCh:
				logTrace.Println("Got a click on BP HOME")
				setThresholds(40, 70)
				mStatus.SetTitle(getStatus())
			case <-mFnlock.ClickedCh:
				toggleFnlock()
				logTrace.Println("Got a click on fnlock")
				mFnlock.SetTitle(getFnlockStatus())
			case <-mQuit.ClickedCh:
				logTrace.Println("Got a click on Quit")
				systray.Quit()
				return
			}
		}
	}()
}

func checkBatpro() bool {
	// TODO check sudo
	logTrace.Println("Checking to see if batpro script is available")
	cmd := exec.Command("/usr/bin/sudo", "-n", "batpro")
	if err := cmd.Run(); err != nil {
		logTrace.Println("batpro script not accessible")
		return false
	}
	logInfo.Println("Found batpro script")
	return true
}

func checkFnlock() bool {
	// TODO check sudo
	logTrace.Println("Checking to see if fnlock script is available")
	cmd := exec.Command("/usr/bin/sudo", "-n", "fnlock")
	if err := cmd.Run(); err != nil {
		logTrace.Println("fnlock script not accessible")
		return false
	}
	logInfo.Println("Found fnlock script")
	return true
}

func setThresholds(min int, max int) {
	cmd := exec.Command("/usr/bin/sudo", "-n", "batpro", "custom", strconv.Itoa(min), strconv.Itoa(max))
	if err := cmd.Run(); err != nil {
		logError.Println("Failed to set thresholds")
	}
}

func setBatproOff() {
	cmd := exec.Command("/usr/bin/sudo", "-n", "batpro", "off")
	if err := cmd.Run(); err != nil {
		logError.Println("Failed to turn off battery protection")
	}
}

func toggleFnlock() {
	cmd := exec.Command("/usr/bin/sudo", "-n", "fnlock", "toggle")
	if err := cmd.Run(); err != nil {
		logError.Println("Failed to toggle Fn-Lock")
	}
}

func getStatus() string {
	cmd := exec.Command("/usr/bin/sudo", "-n", "batpro", "status")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		logError.Println("Failed to get battery protection status from script")
		return "BP status unavailable"
	}
	r := "ERROR"
	state, min, max := parseStatus(out.String())
	switch state {
	case "off":
		r = "Battery protection OFF"
	case "on":
		if min != 0 && min <= 100 && max != 0 && max <= 100 {
			r = "Battery protection mode "
			switch {
			case min == 40 && max == 70:
				r = r + "HOME"
			case min == 70 && max == 90:
				r = r + "OFFICE (70%-90%)"
			case min == 95 && max == 100:
				r = r + "TRAVEL (95%-100%)"
			default:
				r = r + fmt.Sprintf("custom: %d%%-%d%%", min, max)
			}
		} else {
			logWarning.Println("BP thresholds don't make sense: min %d%%, max %d%%", min, max)
			r = "ON, but thresholds make no sense."
		}
	default:
		r = "ERROR: can not get BP status!"
	}
	return r
}

func getFnlockStatus() string {
	cmd := exec.Command("/usr/bin/sudo", "-n", "fnlock", "status")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		logError.Println("Failed to get fnlock status from script")
		return "Fn-Lock status unavailable"
	}
	r := "ERROR"
	state := parseOnOffStatus(out.String())
	switch state {
	case "off":
		r = "Fn-Lock OFF"
	case "on":
		r = "Fn-Lock ON"
	}
	return r
}

func parseOnOffStatus(s string) string {
	stateRe := regexp.MustCompile(`^o(n|ff)$`)
	lines := strings.Split(s, "\n")
	state := ""
	for _, line := range lines {
		if stateRe.MatchString(line) {
			state = line
		}
	}
	return state
}

func parseStatus(s string) (string, int, int) {
	stateRe := regexp.MustCompile(`^battery protection is o[a-z]{1,}`)
	minRe := regexp.MustCompile(`^minimum \d* %$`)
	maxRe := regexp.MustCompile(`^maximum \d* %$`)
	lines := strings.Split(s, "\n")
	state := ""
	min := 0
	max := 0
	for _, line := range lines {
		if stateRe.MatchString(line) {
			state = (strings.TrimPrefix(line, "battery protection is "))
		}
		if minRe.MatchString(line) {
			val, err := strconv.Atoi((strings.TrimPrefix(strings.TrimSuffix(line, " %"), "minimum ")))
			if err != nil {
				min = 0
			} else {
				min = val
			}
		}
		if maxRe.MatchString(line) {
			val, err := strconv.Atoi((strings.TrimPrefix(strings.TrimSuffix(line, " %"), "maximum ")))
			if err != nil {
				max = 0
			} else {
				max = val
			}
		}
	}
	return state, min, max
}

func onExit() {
	// cleanup (maybe TODO)
}

func getIcon(s string) []byte {
	file, err := assets.Open(s)
	if err != nil {
		logError.Println(err)
	}
	defer file.Close()
	b, err := ioutil.ReadAll(file)
	if err != nil {
		logError.Println(err)
	}
	return b
}
