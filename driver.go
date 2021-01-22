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

package main

import (
	"bytes"
	"errors"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	fnlockDriverEndpoint = "/sys/devices/platform/huawei-wmi/fn_lock_state"
	threshKernelPath     = "/sys/class/power_supply/BAT"
	threshKernelMin      = "/charge_control_start_threshold"
	threshKernelMax      = "/charge_control_end_threshold"
	saveValuesPath       = "/etc/default/huawei-wmi/"
)

var (
	fnlockEndpoints     = []fnlockEndpoint{}
	threshEndpoints     = []threshEndpoint{}
	threshSaveEndpoints = []threshDriver{
		{threshDriverSingle{path: (saveValuesPath + "charge_control_thresholds")}},
		{threshDriverSingle{path: (saveValuesPath + "charge_thresholds")}},
	}
	threshDriver1 = threshDriver{threshDriverSingle{path: "/sys/devices/platform/huawei-wmi/charge_thresholds"}}
	threshDriver2 = threshDriver{threshDriverSingle{path: "/sys/devices/platform/huawei-wmi/charge_control_thresholds"}}
)

type fnlockEndpoint interface {
	toggle()
	get() (bool, error)
	isWritable() bool
}

type threshEndpoint interface {
	set(min, max int)
	get() (min, max int, err error)
	isWritable() bool
}

type fnlockScript struct {
	getCmd    *exec.Cmd
	toggleCmd *exec.Cmd
}

type fnlockDriver struct {
	path string
}

type wmiDriver interface {
	write(min, max int) error
	get() (min, max int, err error)
}

type threshDriver struct {
	wmiDriver
}

type threshDriverSingle struct {
	path string
}

type threshDriverMinMax struct {
	pathMin string
	pathMax string
}

type threshScript struct {
	getCmd *exec.Cmd
	setCmd *exec.Cmd
	offCmd *exec.Cmd
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
			logTrace.Println("successful write to driver interface")
			return true
		}
	}
	logTrace.Println(err)
	logWarning.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "ReadOnlyDriver", Other: "Driver interface is readable but not writeable."}}))
	return false
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

func (scr threshScript) isWritable() bool {
	return true
}

func (drv threshDriver) isWritable() bool {
	logTrace.Println("Checking if the threshold endpoint is writable...")
	return drv.checkWritable()
}

func (drv fnlockDriver) isWritable() bool {
	logTrace.Println("Checking if the fnlock endpoint is writable...")
	return drv.checkWritable()
}

func (scr fnlockScript) isWritable() bool {
	return true
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

func (scr fnlockScript) toggle() {
	cmd := scr.toggleCmd
	if err := cmd.Run(); err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CantToggleFnlock", Other: "Failed to toggle Fn-Lock"}}))
	}
}

func (drv threshDriverSingle) get() (min, max int, err error) {
	if _, err = os.Stat(drv.path); err != nil {
		logTrace.Printf("Couldn't access %q.", drv.path)
		return
	}

	var values [2]string
	val, err := ioutil.ReadFile(drv.path)
	if err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CantReadBatteryDriver", Other: "Failed to get thresholds from driver interface"}}))
		logTrace.Println(err)
	}

	valuesReceived := strings.Split(strings.TrimSpace(string(val)), " ")
	logTrace.Println("got values from interface:", valuesReceived)
	if len(valuesReceived) != 2 {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CantUnderstandBattery", Other: "Can not make sense of driver interface value {{.Value}}"}, TemplateData: map[string]interface{}{"Value": val}}))
		return
	}
	for i := 0; i < 2; i++ {
		values[i] = valuesReceived[i]
	}
	min, max, err = valuesAtoi(values[0], values[1])
	return
}

func valuesAtoi(mins, maxs string) (min, max int, err error) {
	min, err = strconv.Atoi(mins)
	if err != nil {
		logTrace.Println(err)
	}
	max, err = strconv.Atoi(maxs)
	if err != nil {
		logTrace.Println(err)
	}
	logTrace.Printf("interpreted values: min %d%%, max %d%%\n", min, max)
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

func (drv threshDriverSingle) write(min, max int) error {
	values := []byte(strconv.Itoa(min) + " " + strconv.Itoa(max) + "\n")
	err := ioutil.WriteFile(drv.path, values, 0664)
	if err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "CantSetBattery"}))
	} else {
		logTrace.Println("successful write to driver interface")
	}
	return err
}

func (drv threshDriver) checkWritable() bool {
	min, max, err := drv.get()
	if err != nil {
		return false
	}
	err = drv.write(min, max)
	return (err == nil)
}

func (drv threshDriver) set(min, max int) {
	if err := drv.write(min, max); err != nil {
		return
	}
	if config.wait {
		logTrace.Println("thresholds pushed to driver, will wait for them to be set")
		// driver takes some time to set values due to ACPI bug
		for i := 1; i < 5; i++ {
			time.Sleep(900 * time.Millisecond)
			logTrace.Println("checking thresholds, attempt", i)
			newMin, newMax, err := drv.get()
			if min == newMin && max == newMax && err == nil {
				logTrace.Println("thresholds set as expected")
				break
			}
			logTrace.Println("not set yet")
		}
		logTrace.Println("alright, going on")
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

func setThresholds(min int, max int) {
	config.thresh.set(min, max)
	if config.threshPers != nil {
		logTrace.Println("Saving values for persistence...")
		config.threshPers.set(min, max)
	}
}

func getStatus() string {
	var r, status string
	min, max, err := config.thresh.get()
	if err != nil {
		r = localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "BatteryProtectionStatusError", Other: "ERROR: can not get BP status!"}})
	} else {
		if min >= 0 && min <= 100 && max >= 0 && max <= 100 && min <= max {
			switch {
			case min == 0 && (max == 100 || max == 0):
				status = localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "StatusOff"})
				r = localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "BatteryProtectionOff", Other: "Battery protection is {{.Status}}"}, TemplateData: map[string]interface{}{"Status": status}})
				return r
			case min == 40 && max == 70:
				status = localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "StatusHome"})
			case min == 70 && max == 90:
				status = localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "StatusOffice"})
			case min == 95 && max == 100:
				status = localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "StatusTravel"})
			default:
				status = localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "StatusCustom", TemplateData: map[string]interface{}{"Min": min, "Max": max}})
			}
		} else {
			logWarning.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "StrangeThresholds", Other: "BP thresholds don't make sense: min {{.Min}}%, max {{.Max}}%"}, TemplateData: map[string]interface{}{"Min": min, "Max": max}}))
			status = localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "StatusOn"})
			r = localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "BatteryProtectionStatusStrange", Other: "{{.Status}}, but thresholds make no sense."}, TemplateData: map[string]interface{}{"Status": status}})
			return r
		}
		r = localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "BatteryProtectionStatus", Other: "Battery protection mode: {{.Status}}"}, TemplateData: map[string]interface{}{"Status": status}})
	}
	return r
}

func getFnlockStatus() string {
	var r, status string
	state, err := config.fnlock.get()
	if state {
		status = localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "StatusOn"})
	} else {
		status = localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "StatusOff"})
	}
	if err != nil {
		r = localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "FnlockStatusError", Other: "ERROR: Fn-Lock state unknown"}})
	} else {
		r = localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "FnlockStatus", Other: "Fn-Lock is {{.Status}}"}, TemplateData: map[string]interface{}{"Status": status}})
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
