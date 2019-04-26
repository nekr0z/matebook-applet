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
	"flag"
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
	driverGet    bool
	driverSet    bool
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
	verbose := flag.Bool("v", false, "be verbose")
	verboseMore := flag.Bool("vv", false, "be very verbose")
	flag.Parse()

	switch {
	case *verbose:
		logInit(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	case *verboseMore:
		logInit(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	default:
		logInit(ioutil.Discard, ioutil.Discard, os.Stdout, os.Stderr)
	}

	driverGet, driverSet = checkWmi()
	if driverSet {
		logInfo.Println("will use driver interface")
		scriptBatpro = false
		scriptFnlock = false
	} else {
		logInfo.Println("no full functionality with driver interface, will try scripts")
		scriptBatpro = checkBatpro()
		scriptFnlock = checkFnlock()
	}
	if scriptBatpro || scriptFnlock || driverGet {
		systray.Run(onReady, onExit)
	} else {
		logError.Println("Neither a supported version of Huawei-WMI driver, nor any of the required scripts are properly installed, see README.md#installation-and-setup for instructions")
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
	if scriptBatpro == false && driverGet == false {
		mStatus.Hide()
		logTrace.Println("no access to BP information, not showing it")
	} else {
		mStatus.SetTitle(getStatus())
	}
	if scriptBatpro == false && driverSet == false {
		mOff.Hide()
		mTravel.Hide()
		mOffice.Hide()
		mHome.Hide()
		logTrace.Println("no way to change BP settings, not showing the corresponding GUI")
	}
	if scriptFnlock == false && driverGet == false {
		mFnlock.Hide()
		logTrace.Println("no access to Fn-Lock setting, not showing its GUI")
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

func checkWmi() (bool, bool) {
	logTrace.Println("Checking to see if Huawei-WMI driver interface is accessible")
	val, err := ioutil.ReadFile("/sys/devices/platform/huawei-wmi/fn_lock_state")
	if err != nil {
		logInfo.Println("could not read fn_lock_state, driver interface doesn't seem to be accessible")
		logTrace.Println(err)
		return false, false
	}
	logInfo.Println("driver interface is readable")
	err = ioutil.WriteFile("/sys/devices/platform/huawei-wmi/fn_lock_state", val, 0644)
	if err != nil {
		logTrace.Println(err)
		logWarning.Println("Driver interface is readable but not writeable.")
		return true, false
	}
	logInfo.Println("successful write to driver interface")
	return true, true
}

func setThresholds(min int, max int) {
	switch {
	case driverSet:
		logTrace.Println("setting thresholds using driver interface...")
		if err := ioutil.WriteFile("/sys/devices/platform/huawei-wmi/charge_thresholds", []byte(strconv.Itoa(min)+" "+strconv.Itoa(max)), 0644); err != nil {
			logError.Println("Failed to set thresholds")
		}
	case scriptBatpro:
		cmd := exec.Command("/usr/bin/sudo", "-n", "batpro", "custom", strconv.Itoa(min), strconv.Itoa(max))
		if err := cmd.Run(); err != nil {
			logError.Println("Failed to set thresholds")
		}
	default:
		logWarning.Println("Something unexpected happened, there's a bug somewhere")
	}
}

func setBatproOff() {
	switch {
	case driverSet:
		setThresholds(0, 100)
	case scriptBatpro:
		cmd := exec.Command("/usr/bin/sudo", "-n", "batpro", "off")
		if err := cmd.Run(); err != nil {
			logError.Println("Failed to turn off battery protection")
		}
	default:
		logWarning.Println("Something unexpected happened, there's a bug somewhere")
	}
}

func toggleFnlock() {
	switch {
	case driverSet:
		val, err := ioutil.ReadFile("/sys/devices/platform/huawei-wmi/fn_lock_state")
		if err != nil {
			logError.Println("Could not read fn_lock_state from driver interface")
			logTrace.Println(err)
		}
		value := strings.TrimSpace(string(val))
		switch value {
		case "0":
			logTrace.Println("got fn-lock 0 from driver, setting to 1")
			value = "1"
		case "1":
			logTrace.Println("got fn-lock 1 from driver, setting to 0")
			value = "0"
		default:
			logWarning.Println("Something unexpected happened, there's a bug somewhere")
		}
		err = ioutil.WriteFile("/sys/devices/platform/huawei-wmi/fn_lock_state", []byte(value), 0644)
		if err != nil {
			logTrace.Println(err)
			logWarning.Println("Could not set Fn-Lock status through driver interface")
		}
		logTrace.Println("successful write to driver interface")
	case scriptBatpro:
		cmd := exec.Command("/usr/bin/sudo", "-n", "fnlock", "toggle")
		if err := cmd.Run(); err != nil {
			logError.Println("Failed to toggle Fn-Lock")
		}
	case driverGet:
		// there can be legitimate cases for this function to be called when no script is installed and driver
		// interface is readable but not writeable
		logTrace.Println("driver interface not writable, ignoring request to toggle Fn-Lock")
	default:
		logWarning.Println("Something unexpected happened, there's a bug somewhere")
	}
}

func getStatus() string {
	var (
		state    string
		min, max int
		err      error
	)
	r := "ERROR"
	switch {
	case driverGet:
		var val []byte
		val, err = ioutil.ReadFile("/sys/devices/platform/huawei-wmi/charge_thresholds")
		if err != nil {
			logError.Println("Could not read charge thresholds from driver interface")
			logTrace.Println(err)
		}
		values := strings.Split(strings.TrimSpace(string(val)), " ")
		logTrace.Println("got values from interface:", values)
		if len(values) == 2 {
			min, err = strconv.Atoi(values[0])
			if err != nil {
				logTrace.Println(err)
				min = 0
			}
			max, err = strconv.Atoi(values[1])
			if err != nil {
				logTrace.Println(err)
				max = 0
			}
			logTrace.Printf("interpreted values: min %d%%, max %d%%\n", min, max)
			switch {
			case min == 0 && max <= 100:
				state = "off"
			case 0 < min && min < 100 && 0 < max && max <= 100:
				state = "on"
			}
		} else {
			logError.Println("Can not make sence of driver interface value", val)
		}
	case scriptBatpro:
		cmd := exec.Command("/usr/bin/sudo", "-n", "batpro", "status")
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			logError.Println("Failed to get battery protection status from script")
			return "BP status unavailable"
		}
		state, min, max = parseStatus(out.String())
	default:
		logWarning.Println("Something unexpected happened, there's a bug somewhere")
	}
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
			logWarning.Printf("BP thresholds don't make sense: min %d%%, max %d%%\n", min, max)
			r = "ON, but thresholds make no sense."
		}
	default:
		r = "ERROR: can not get BP status!"
	}
	return r
}

func getFnlockStatus() string {
	r := "ERROR"
	var state string
	switch {
	case driverGet:
		val, err := ioutil.ReadFile("/sys/devices/platform/huawei-wmi/fn_lock_state")
		if err != nil {
			logError.Println("Could not read Fn-Lock state from driver interface")
			logTrace.Println(err)
		}
		value := strings.TrimSpace(string(val))
		switch value {
		case "0":
			state = "off"
		case "1":
			state = "on"
		default:
			logWarning.Println("Fn-lock state reported by driver doesn't make sense")
		}
	case scriptFnlock:
		cmd := exec.Command("/usr/bin/sudo", "-n", "fnlock", "status")
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			logError.Println("Failed to get fnlock status from script")
			return "Fn-Lock status unavailable"
		}
		state = parseOnOffStatus(out.String())
	default:
		logWarning.Println("Something unexpected happened, there's a bug somewhere")
	}
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
