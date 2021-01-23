// Copyright (C) 2021 Evgeny Kuznetsov (evgeny@kuznetsov.md)
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

// +build darwin

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func init() {
	threshEndpoints = append(threshEndpoints, threshDriver{splitThreshEndpoint{ioioGetter{}, errSetter}}, threshDriver{splitThreshEndpoint{zeroGetter{}, errSetter{}}})
}

// splitThreshEndpoint is a wmiDriver that has really differing
// ways of reading and writing
type splitThreshEndpoint struct {
	getter threshGetter
	setter threshSetter
}

func (ste splitThreshEndpoint) write(min, max int) error {
	return ste.setter.set(min, max)
}

func (ste splitThreshEndpoint) get() (min, max int, err error) {
	return ste.getter.get()
}

// threshGetter has a way to read battery thresholds
type threshGetter interface {
	get() (min, max int, err error)
}

// threshSetter has a way to set battery thresholds
type threshSetter interface {
	set(min, max int) error
}

// zeroGetter always returns 0 100
type zeroGetter struct{}

func (_ zeroGetter) get() (min, max int, err error) {
	return 0, 100, nil
}

// errSetter always returs error
type errSetter struct{}

func (_ errSetter) set(_, _ int) error {
	return fmt.Errorf("not implemented")
}

// ioioGetter is a crude and suboptimal threshGetter using ioio and system log
type ioioGetter struct{}

func (_ ioioGetter) get() (min, max int, err error) {
	cmdLog := exec.Command("log", "stream", "--predicate", "senderImagePath contains \"ACPIDebug\"")
	var out bytes.Buffer
	cmdLog.Stdout = &out
	cmdIoio := exec.Command("ioio", "-s", "org_rehabman_ACPIDebug", "dbg4", "0")

	err := cmdLog.Start()
	if err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CantReadBatteryIoio", Other: "Failed to get battery protection status from the system"}}))
		return
	}
	defer cmdLog.Process.Signal(os.Kill)
	time.Sleep(200 * time.Millisecond)
	err = cmdIoio.Run()
	if err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CantReadBatteryIoio", Other: "Failed to get battery protection status from the system"}}))
		return
	}
	time.Sleep(500 * time.Millisecond)

	logTrace.Printf("Read from the log: %s", out.String())

	return 0, 100, nil
}
