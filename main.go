// Copyright (C) 2019 Evgeny Kuznetsov (evgeny@kuznetsov.md)
//
// This Source Code Form is subject to the terms of the General Public License v. 3.0

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
	logTrace   *log.Logger
	logInfo    *log.Logger
	logWarning *log.Logger
	logError   *log.Logger
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
	if checkBatpro() {
		systray.Run(onReady, onExit)
	} else {
		logError.Println("The required script is not properly installed\nsee https://github.com/nekr0z/matebook-applet#installation-and-setup for instructions")
	}
}

func onReady() {
	logTrace.Println("Setting up menu...")
	systray.SetIcon(getIcon("/Iconsmind-Outline-Battery-Charge.ico"))
	mStatus := systray.AddMenuItem(getStatus(), "")
	systray.AddSeparator()
	mOff := systray.AddMenuItem("OFF", "Switch off battery protection")
	mTravel := systray.AddMenuItem("TRAVEL (95%-100%)", "Set battery protection to TRAVEL")
	mOffice := systray.AddMenuItem("OFFICE (70%-90%)", "Set battery protection to OFFICE")
	mHome := systray.AddMenuItem("HOME (40%-70%)", "Set battery protection to HOME")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the applet")
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
	cmd := exec.Command("/usr/bin/sudo", "-n", "batpro")
	if err := cmd.Run(); err != nil {
		return false
	}
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
		logError.Print(err)
	}
	defer file.Close()
	b, err := ioutil.ReadAll(file)
	if err != nil {
		logError.Print(err)
	}
	return b
}
