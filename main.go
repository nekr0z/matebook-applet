package main

import (
	"bytes"
	"fmt"
	"github.com/getlantern/systray"
	"io/ioutil"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(getIcon("assets/Iconsmind-Outline-Battery-Charge.ico"))
	mStatus := systray.AddMenuItem(getStatus(), "")
	systray.AddSeparator()
	mOff := systray.AddMenuItem("OFF", "Switch off battery protection")
	mTravel := systray.AddMenuItem("TRAVEL (95%-100%)", "Set battery protection to TRAVEL")
	mOffice := systray.AddMenuItem("OFFICE (70%-90%)", "Set battery protection to OFFICE")
	mHome := systray.AddMenuItem("HOME (40%-70%)", "Set battery protection to HOME")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the applet")
	go func() {
		for {
			select {
			case <-mStatus.ClickedCh:
				mStatus.SetTitle(getStatus())
			case <-mOff.ClickedCh:
				setBatproOff()
				mStatus.SetTitle(getStatus())
			case <-mTravel.ClickedCh:
				setThresholds(95, 100)
				mStatus.SetTitle(getStatus())
			case <-mOffice.ClickedCh:
				setThresholds(70, 90)
				mStatus.SetTitle(getStatus())
			case <-mHome.ClickedCh:
				setThresholds(40, 70)
				mStatus.SetTitle(getStatus())
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func setThresholds(min int, max int) {
	cmd := exec.Command("/usr/bin/sudo", "batpro", "custom", strconv.Itoa(min), strconv.Itoa(max))
	if err := cmd.Run(); err != nil {
		fmt.Println("Failed to set thresholds")
	}
}

func setBatproOff() {
	cmd := exec.Command("/usr/bin/sudo", "batpro", "off")
	if err := cmd.Run(); err != nil {
		fmt.Println("Failed to turn off battery protection")
	}
}

func getStatus() string {
	cmd := exec.Command("/usr/bin/sudo", "batpro", "status")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "ERROR running script"
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
	b, err := ioutil.ReadFile(s)
	if err != nil {
		fmt.Print(err)
	}
	return b
}
