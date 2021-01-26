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

// +build linux

package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

const (
	fnlockDriverEndpoint = "/sys/devices/platform/huawei-wmi/fn_lock_state"
	threshKernelPath     = "/sys/class/power_supply/BAT"
	threshKernelMin      = "/charge_control_start_threshold"
	threshKernelMax      = "/charge_control_end_threshold"
)

var (
	threshDriver1 = threshDriver{threshDriverSingle{path: "/sys/devices/platform/huawei-wmi/charge_thresholds"}}
	threshDriver2 = threshDriver{threshDriverSingle{path: "/sys/devices/platform/huawei-wmi/charge_control_thresholds"}}
)

func initEndpoints() {
	fndrv := fnlockDriver{path: fnlockDriverEndpoint}
	fnlockEndpoints = append(fnlockEndpoints, fndrv)

	threshEndpoints = append(threshEndpoints, threshDriver2, threshDriver1)
	for i := 0; i < 10; i++ {
		min := threshKernelPath + strconv.Itoa(i) + threshKernelMin
		max := threshKernelPath + strconv.Itoa(i) + threshKernelMax
		threshEndpoints = append(threshEndpoints, threshDriver{threshDriverMinMax{pathMin: min, pathMax: max}})
	}

	if config.useScripts {
		sudo := "/usr/bin/sudo"
		fnscr := fnlockScript{toggleCmd: exec.Command(sudo, "-n", "fnlock", "toggle"), getCmd: exec.Command(sudo, "-n", "fnlock", "status")}
		fnlockEndpoints = append(fnlockEndpoints, fnscr)

		cmdLine := []string{sudo, "-n", "batpro"}
		batpro := threshScript{
			getCmd: exec.Command(sudo, append(cmdLine, "status")...),
			setCmd: exec.Command(sudo, append(cmdLine, "custom")...),
			offCmd: exec.Command(sudo, append(cmdLine, "off")...),
		}
		threshEndpoints = append(threshEndpoints, batpro)
	}
}

// fnlockDriver is Huawei-WMI-style fnlockEndpoint
type fnlockDriver struct {
	path string
}

func (drv fnlockDriver) toggle() {
	val, err := drv.get()
	if err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "CantReadFnlock"}))
		logTrace.Println(err)
		return
	}
	value := btobb(!val)
	err = ioutil.WriteFile(drv.path, value, 0644)
	if err != nil {
		logTrace.Println(err)
		logWarning.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CantSetFnlockDriver", Other: "Could not set Fn-Lock status through driver interface"}}))
		return
	}
	logTrace.Println("successful write to driver interface")
}

func (drv fnlockDriver) get() (bool, error) {
	if _, err := os.Stat(drv.path); err != nil {
		logTrace.Printf("Couldn't access %q.", drv.path)
		return false, err
	}

	val, err := ioutil.ReadFile(drv.path)
	if err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "CantReadFnlock"}))
		logTrace.Println(err)
	}
	value := strings.TrimSpace(string(val))
	switch value {
	case "0":
		logTrace.Println("Fn-lock is OFF")
		return false, err
	case "1":
		logTrace.Println("Fn-lock is ON")
		return true, err
	default:
		logWarning.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "StrangeFnlock", Other: "Fn-lock state reported by driver doesn't make sense"}}))
		return false, errors.New("state is reported as " + value)
	}
}

func (drv fnlockDriver) isWritable() bool {
	logTrace.Println("Checking if the fnlock endpoint is writable...")
	return drv.checkWritable()
}

func (drv fnlockDriver) checkWritable() bool {
	val, err := drv.get()
	if err == nil {
		err = ioutil.WriteFile(drv.path, btobb(val), 0664)
		if err == nil {
			logTrace.Println("successful write to driver interface")
			return true
		}
	}
	logTrace.Println(err)
	logWarning.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "ReadOnlyDriver", Other: "Driver interface is readable but not writeable."}}))
	return false
}

// fnlockScript is a command-based fnlockEndpoint
type fnlockScript struct {
	getCmd    *exec.Cmd
	toggleCmd *exec.Cmd
}

func (scr fnlockScript) toggle() {
	cmd := scr.toggleCmd
	if err := cmd.Run(); err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CantToggleFnlock", Other: "Failed to toggle Fn-Lock"}}))
	}
}

func (scr fnlockScript) get() (bool, error) {
	cmd := scr.getCmd
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CantReadFnlockScript", Other: "Failed to get fnlock status from script"}}))
	}
	state := parseOnOffStatus(out.String())
	if state == "on" {
		return true, err
	}
	return false, err
}

func (scr fnlockScript) isWritable() bool {
	return true
}

// threshDriverMinMax is a wmiDriver that is two separate endpoints for min and max
type threshDriverMinMax struct {
	pathMin string
	pathMax string
}

func (drv threshDriverMinMax) write(min, max int) error {
	err := drv.writeDo(0, 100)
	if err != nil {
		return err
	}
	err = drv.writeDo(min, max)
	if err == nil {
		logTrace.Println("successful write to driver interface")
	}
	return err
}

func (drv threshDriverMinMax) get() (min, max int, err error) {
	if _, err = os.Stat(drv.pathMin); err != nil {
		logTrace.Printf("Couldn't access %q.", drv.pathMin)
		return
	}

	var values [2]string
	val, err := ioutil.ReadFile(drv.pathMin)
	if err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CantReadBatteryMin", Other: "Failed to get min threshold from driver interface"}}))
		logTrace.Println(err)
		return
	}
	values[0] = string(val)
	val, err = ioutil.ReadFile(drv.pathMax)
	if err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CantReadBatteryMax", Other: "Failed to get max threshold from driver interface"}}))
		logTrace.Println(err)
		return
	}
	values[1] = string(val)
	for i := 0; i < 2; i++ {
		values[i] = strings.TrimSuffix(values[i], "\n")
	}

	min, max, err = valuesAtoi(values[0], values[1])
	return

}

func (drv threshDriverMinMax) writeDo(min, max int) error {
	if err := ioutil.WriteFile(drv.pathMin, []byte(strconv.Itoa(min)), 0664); err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CantSetBatteryMin", Other: "Failed to set min threshold"}}))
		return err
	}
	if err := ioutil.WriteFile(drv.pathMax, []byte(strconv.Itoa(max)), 0664); err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CantSetBatteryMax", Other: "Failed to set max threshold"}}))
		return err
	}
	return nil
}

// threshScript is a command-based threshEndpoint
type threshScript struct {
	getCmd *exec.Cmd
	setCmd *exec.Cmd
	offCmd *exec.Cmd
}

func (scr threshScript) set(min, max int) {
	var cmd *exec.Cmd
	if min == 0 && max == 100 {
		cmd = scr.offCmd
	} else {
		cmd = scr.setCmd
		cmd.Args = append(cmd.Args, strconv.Itoa(min), strconv.Itoa(max))
	}
	if err := cmd.Run(); err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "CantSetBattery"}))
	}
}

func (scr threshScript) get() (min, max int, err error) {
	cmd := scr.getCmd
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CantReadBatteryScript", Other: "Failed to get battery protection status from script"}}))
		return
	}
	state, min, max := parseStatus(out.String())
	if state == "on" {
		min = 0
		max = 100
	}
	return
}

func (scr threshScript) isWritable() bool {
	return true
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

func btobb(b bool) []byte {
	var s string
	if b {
		s = "1"
	} else {
		s = "0"
	}
	return []byte(s)
}
