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
	"errors"
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
	"time"
)

const (
	defaultIcon          = "/matebook-applet.png"
	fnlockDriverEndpoint = "/sys/devices/platform/huawei-wmi/fn_lock_state"
)

var (
	logTrace                 *log.Logger
	logInfo                  *log.Logger
	logWarning               *log.Logger
	logError                 *log.Logger
	scriptBatpro             bool
	scriptFnlock             bool
	driverGet                bool
	driverSet                bool
	driverThresholdsPlatform bool // pre-3.3 driver uses /sys/devices/platform/huawei-wmi/ to store thresholds
	waitForDriver            bool
	version                  string = "custom-build"
	iconPath                 string
	saveValues               bool
	noSaveValues             bool
	saveValuesPath           string = "/etc/default/huawei-wmi/"
	fnlockEndpoints                 = []fnlockEndpoint{}
	cfg                      config
)

type config struct {
	fnlock fnlockEndpoint
}

type fnlockEndpoint interface {
	toggle()
	get() (bool, error)
}

type fnlockScript struct {
	getCmd    *exec.Cmd
	toggleCmd *exec.Cmd
}

type fnlockDriver struct {
	path string
}

func init() {
	fndrv := fnlockDriver{path: fnlockDriverEndpoint}
	fnlockEndpoints = append(fnlockEndpoints, fndrv)
	fnscr := fnlockScript{toggleCmd: exec.Command("/usr/bin/sudo", "-n", "fnlock", "toggle"), getCmd: exec.Command("/usr/bin/sudo", "-n", "fnlock", "status")}
	fnlockEndpoints = append(fnlockEndpoints, fnscr)
}

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
	flag.StringVar(&iconPath, "icon", "", "path of a custom icon to use")
	flag.BoolVar(&waitForDriver, "wait", false, "wait for driver to set battery thresholds (obsolete)")
	flag.BoolVar(&saveValues, "s", true, "save values for persistence (deprecated)")
	flag.BoolVar(&noSaveValues, "n", false, "do not save values")
	flag.Parse()

	if noSaveValues {
		saveValues = false
	}

	switch {
	case *verbose:
		logInit(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	case *verboseMore:
		logInit(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	default:
		logInit(ioutil.Discard, ioutil.Discard, os.Stdout, os.Stderr)
	}

	logInfo.Printf("matebook-applet version %s\n", version)

	// need to find working fnlock interface (if any)
	for _, fnlck := range fnlockEndpoints {
		_, err := fnlck.get()
		if err != nil {
			continue
		}
		cfg.fnlock = fnlck
		_, ok := fnlck.(fnlockScript)
		if ok {
			logInfo.Println("Found fnlock script, will use it")
			break
		}
		fl, ok := fnlck.(fnlockDriver)
		if fl.checkWritable() {
			logInfo.Println("Found writable fnlock driver interface, will use it")
			break
		}
	}

	driverGet, driverSet = checkWmi()
	if driverSet {
		logInfo.Println("will use driver interface")
		scriptBatpro = false
	} else {
		logInfo.Println("no full functionality with driver interface, will try scripts")
		scriptBatpro = checkBatpro()
	}
	if scriptBatpro || driverGet || cfg.fnlock != nil {
		systray.Run(onReady, onExit)
	} else {
		logError.Println("Neither a supported version of Huawei-WMI driver, nor any of the required scripts are properly installed, see README.md#installation-and-setup for instructions")
	}
}

func onReady() {
	logTrace.Println("Setting up menu...")
	systray.SetIcon(getIcon(iconPath, defaultIcon))
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
	if !scriptBatpro && !driverGet {
		mStatus.Hide()
		logTrace.Println("no access to BP information, not showing it")
	} else {
		mStatus.SetTitle(getStatus())
	}
	if !scriptBatpro && !driverSet {
		mOff.Hide()
		mTravel.Hide()
		mOffice.Hide()
		mHome.Hide()
		logTrace.Println("no way to change BP settings, not showing the corresponding GUI")
	}
	if cfg.fnlock == nil {
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
				logTrace.Println("Got a click on fnlock")
				cfg.fnlock.toggle()
				mFnlock.SetTitle(getFnlockStatus())
			case <-mQuit.ClickedCh:
				logTrace.Println("Got a click on Quit")
				systray.Quit()
				return
			}
		}
	}()
}

func (drv fnlockDriver) get() (bool, error) {
	val, err := ioutil.ReadFile(drv.path)
	if err != nil {
		logError.Println("Could not read Fn-Lock state from driver interface")
		logTrace.Println(err)
	}
	value := strings.TrimSpace(string(val))
	switch value {
	case "0":
		return false, err
	case "1":
		return true, err
	default:
		logWarning.Println("Fn-lock state reported by driver doesn't make sense")
		return false, errors.New("state is reported as " + value)
	}
}

func btobb(b bool) []byte {
	var s string
	if b {
		s = "1"
	} else {
		s = "0"
	}
	return []byte(s)
}

func (drv fnlockDriver) checkWritable() bool {
	val, err := drv.get()
	if err == nil {
		err = ioutil.WriteFile(drv.path, btobb(val), 0664)
		if err == nil {
			logInfo.Println("successful write to driver interface")
			return true
		}
	}
	logTrace.Println(err)
	logWarning.Println("Driver interface is readable but not writeable.")
	return false
}

func (drv fnlockDriver) toggle() {
	val, err := drv.get()
	if err != nil {
		logError.Println("Could not read fn_lock_state from driver interface")
		logTrace.Println(err)
		return
	}
	value := btobb(!val)
	err = ioutil.WriteFile(drv.path, value, 0644)
	if err != nil {
		logTrace.Println(err)
		logWarning.Println("Could not set Fn-Lock status through driver interface")
		return
	}
	logTrace.Println("successful write to driver interface")
	return
}

func (scr fnlockScript) get() (bool, error) {
	cmd := scr.getCmd
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		logError.Println("Failed to get fnlock status from script")
	}
	state := parseOnOffStatus(out.String())
	if state == "on" {
		return true, err
	}
	return false, err
}

func (scr fnlockScript) toggle() {
	cmd := scr.toggleCmd
	if err := cmd.Run(); err != nil {
		logError.Println("Failed to toggle Fn-Lock")
	}
}

func (scr fnlockScript) check() bool {
	ok := checkSudo()
	if !ok {
		return false
	}
	logTrace.Println("Checking to see if fnlock script is available")
	if _, err := scr.get(); err != nil {
		logTrace.Println("fnlock script not accessible")
		return false
	}
	logInfo.Println("Found fnlock script")
	return true
}

func checkBatpro() bool {
	ok := checkSudo()
	if !ok {
		return false
	}
	logTrace.Println("Checking to see if batpro script is available")
	cmd := exec.Command("/usr/bin/sudo", "-n", "batpro")
	if err := cmd.Run(); err != nil {
		logTrace.Println("batpro script not accessible")
		return false
	}
	logInfo.Println("Found batpro script")
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
	_, err = ioutil.ReadFile("/sys/devices/platform/huawei-wmi/charge_thresholds")
	if err == nil {
		logInfo.Println("battery thresholds accessible in /sys/devices/platform")
		driverThresholdsPlatform = true
	} else {
		logInfo.Println("no battery thresholds in /sys/devices/platform, will use kernel interface BAT0")
	}
	err = ioutil.WriteFile("/sys/devices/platform/huawei-wmi/fn_lock_state", val, 0664)
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
		setDriverThresholds(0, 100)
		setDriverThresholds(min, max)
	case scriptBatpro:
		cmd := exec.Command("/usr/bin/sudo", "-n", "batpro", "custom", strconv.Itoa(min), strconv.Itoa(max))
		if err := cmd.Run(); err != nil {
			logError.Println("Failed to set thresholds")
		}
	default:
		logWarning.Println("Something unexpected happened, there's a bug somewhere")
	}
	if saveValues {
		values := []byte(strconv.Itoa(min) + " " + strconv.Itoa(max) + "\n")
		logTrace.Println("Saving values for persistence...")
		saveValue("charge_thresholds", values)
	}
}

func setDriverThresholds(min, max int) {
	if driverThresholdsPlatform {
		values := []byte(strconv.Itoa(min) + " " + strconv.Itoa(max) + "\n")
		if err := ioutil.WriteFile("/sys/devices/platform/huawei-wmi/charge_thresholds", values, 0664); err != nil {
			logError.Println("Failed to set thresholds")
			return
		}
	} else {
		if err := ioutil.WriteFile("/sys/class/power_supply/BAT0/charge_control_start_threshold", []byte(strconv.Itoa(min)), 0664); err != nil {
			logError.Println("Failed to set min threshold")
			return
		}
		if err := ioutil.WriteFile("/sys/class/power_supply/BAT0/charge_control_end_threshold", []byte(strconv.Itoa(max)), 0664); err != nil {
			logError.Println("Failed to set max threshold")
			return
		}
	}
	if waitForDriver {
		logTrace.Println("thresholds pushed to driver, will wait for them to be set")
		// driver takes some time to set values due to ACPI bug
		for i := 1; i < 5; i++ {
			time.Sleep(900 * time.Millisecond)
			logTrace.Println("checking thresholds, attempt", i)
			newMin, newMax, ok := getDriverThresholds()
			if min == newMin && max == newMax && ok {
				logTrace.Println("thresholds set as expected")
				break
			}
			logTrace.Println("not set yet")
		}
		logTrace.Println("alright, going on")
	}
}

func setBatproOff() {
	switch {
	case driverSet:
		setDriverThresholds(0, 100)
	case scriptBatpro:
		cmd := exec.Command("/usr/bin/sudo", "-n", "batpro", "off")
		if err := cmd.Run(); err != nil {
			logError.Println("Failed to turn off battery protection")
		}
	default:
		logWarning.Println("Something unexpected happened, there's a bug somewhere")
	}
}

func getStatus() string {
	var (
		state, r string
		min, max int
	)
	switch {
	case driverGet:
		var ok bool
		min, max, ok = getDriverThresholds()
		if !ok {
			state = "unknown"
		} else {
			switch {
			case min == 0 && max <= 100:
				state = "off"
			case 0 < min && min < 100 && 0 < max && max <= 100:
				state = "on"
			default:
				logError.Printf("Can't make sence of driver BP values %d-%d", min, max)
			}
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
				r = r + "OFFICE"
			case min == 95 && max == 100:
				r = r + "TRAVEL"
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
	state, err := cfg.fnlock.get()
	if state {
		r = "Fn-Lock ON"
	} else {
		r = "Fn-Lock OFF"
	}
	if err != nil {
		r = "ERROR: Fn-Lock state unknown"
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

func getDriverThresholds() (min, max int, ok bool) {
	logTrace.Println("getting thresholds from the driver")
	var values [2]string
	ok = true
	if driverThresholdsPlatform {
		val, err := ioutil.ReadFile("/sys/devices/platform/huawei-wmi/charge_thresholds")
		if err != nil {
			logError.Println("Failed to get thresholds from driver interface")
			logTrace.Println(err)
		}

		valuesReceived := strings.Split(strings.TrimSpace(string(val)), " ")
		logTrace.Println("got values from interface:", valuesReceived)
		if len(valuesReceived) != 2 {
			logError.Println("Can not make sence of driver interface value", val)
			ok = false
			return
		} else {
			for i := 0; i < 2; i++ {
				values[i] = valuesReceived[i]
			}
		}
	} else {
		val, err := ioutil.ReadFile("/sys/class/power_supply/BAT0/charge_control_start_threshold")
		if err != nil {
			logError.Println("Failed to get min threshold from driver interface")
			logTrace.Println(err)
		}
		values[0] = string(val)
		val, err = ioutil.ReadFile("/sys/class/power_supply/BAT0/charge_control_end_threshold")
		if err != nil {
			logError.Println("Failed to get max threshold from driver interface")
			logTrace.Println(err)
		}
		values[1] = string(val)
		for i := 0; i < 2; i++ {
			values[i] = strings.TrimSuffix(values[i], "\n")
		}
	}

	min, err := strconv.Atoi(values[0])
	if err != nil {
		logTrace.Println(err)
		ok = false
	}
	max, err = strconv.Atoi(values[1])
	if err != nil {
		logTrace.Println(err)
		ok = false
	}
	logTrace.Printf("interpreted values: min %d%%, max %d%%\n", min, max)

	return
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
}

func getIcon(pth, dflt string) []byte {
	b, err := ioutil.ReadFile(pth)
	if err != nil {
		logInfo.Println("Couldn't get custom icon, falling back to default")
		file, err := assets.Open(dflt)
		if err != nil {
			logError.Println(err)
		}
		defer file.Close()
		b, err = ioutil.ReadAll(file)
		if err != nil {
			logError.Println(err)
		}
	} else {
		logInfo.Println("Successfully loaded custom icon from", pth)
	}
	return b
}

func checkSudo() bool {
	logTrace.Println("checking if sudo is available")
	cmd := exec.Command("/bin/sh", "-c", "command -v sudo")
	if err := cmd.Run(); err != nil {
		logInfo.Println(err)
		logWarning.Println("sudo not available")
		return false
	}
	return true
}

func saveValue(file string, value []byte) {
	filePath := saveValuesPath + file
	f, err := os.Create(filePath)
	if err != nil {
		logError.Println(err)
		logWarning.Printf("Could not open file %s for writing.\n", filePath)
		return
	}
	logTrace.Printf("Opened file %s.\n", filePath)
	defer f.Close()

	if _, err = f.Write(value); err != nil {
		logError.Println(err)
		logWarning.Printf("Failed to write to file %s.\n", filePath)
		return
	}
	logTrace.Printf("Successfully wrote to %s.", filePath)

	if err := f.Sync(); err != nil {
		logError.Println(err)
		logWarning.Printf("Failed to sync file %s\n", filePath)
	}
}
