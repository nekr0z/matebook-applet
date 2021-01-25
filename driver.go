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

package main

import (
	"errors"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

const (
	saveValuesPath = "/etc/default/huawei-wmi/"
)

var (
	fnlockEndpoints     = []fnlockEndpoint{}
	threshEndpoints     = []threshEndpoint{}
	threshSaveEndpoints = []threshDriver{
		{threshDriverSingle{path: (saveValuesPath + "charge_control_thresholds")}},
		{threshDriverSingle{path: (saveValuesPath + "charge_thresholds")}},
	}
)

// setThresholds sets BP thresholds using configured endpoint
func setThresholds(min int, max int) {
	config.thresh.set(min, max)
	if config.threshPers != nil {
		logTrace.Println("Saving values for persistence...")
		config.threshPers.set(min, max)
	}
}

// getStatus returns human-readable BP status using configured endpoint
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

// getFnlockStatus returns human-readable FnLock status using configured endpoint
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

// fnlockEndpoint allows to control FnLock state
type fnlockEndpoint interface {
	toggle()
	get() (bool, error)
	isWritable() bool
}

// threshEndpoint allows to control BP state
type threshEndpoint interface {
	set(min, max int)
	get() (min, max int, err error)
	isWritable() bool
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

// threshDriver is Huawei-WMI-style threshEndpoint
type threshDriver struct {
	wmiDriver
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

func (drv threshDriver) isWritable() bool {
	logTrace.Println("Checking if the threshold endpoint is writable...")
	return drv.checkWritable()
}

func (drv threshDriver) checkWritable() bool {
	min, max, err := drv.get()
	if err != nil {
		return false
	}
	err = drv.write(min, max)
	return (err == nil)
}

// wmiDriver is an r/w BP endpoint
type wmiDriver interface {
	write(min, max int) error
	get() (min, max int, err error)
}

// threshDriverSingle is a wmiDriver that is a single file endpoint
type threshDriverSingle struct {
	path string
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

func btobb(b bool) []byte {
	var s string
	if b {
		s = "1"
	} else {
		s = "0"
	}
	return []byte(s)
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
