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
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// +build darwin

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func initEndpoints() {
	threshEndpoints = append(threshEndpoints, threshDriver{splitThreshEndpoint{ioioGetter{}, ioioSetter{}}}, threshDriver{splitThreshEndpoint{zeroGetter{}, errSetter{}}})
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

	err = cmdLog.Start()
	if err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CantRunLog", Other: "Failed to run the \"log\" command. Is \"log\" binary not in PATH?"}}))
		return
	}
	defer cmdLog.Process.Signal(os.Kill)
	defer logTrace.Println("Killing the log output process...")
	time.Sleep(200 * time.Millisecond)
	err = cmdIoio.Run()
	if err != nil {
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CantRunIoio", Other: "Failed to run the \"ioio\" command. Is the binary not in PATH?"}}))
		return
	}
	time.Sleep(500 * time.Millisecond)

	logTrace.Printf("Read from the log: %s", out.String())

	return 0, 100, nil
}

// ioioSetter is a threshSetter that uses ioio
type ioioSetter struct{}

func (_ ioioSetter) set(min, max int) error {
	if min == 0 && max == 100 {
		logTrace.Println("Using ioio to switch battery protection off...")
		return ioioThreshOff()
	}

	logTrace.Printf("Using ioio to set battery thresholds to %d-%d", min, max)
	return ioioThreshSet(min, max)
}

func ioioThreshOff() error {
	cmd := exec.Command("ioio", "-s", "org_rehabman_ACPIDebug", "dbg0", "5")
	err := cmd.Run()
	return err
}

func ioioThreshSet(min, max int) error {
	arg := threshToHexArg(min, max)
	cmd := exec.Command("ioio", "-s", "org_rehabman_ACPIDebug", "dbg5", arg)
	err := cmd.Run()
	return err
}

func threshToHexArg(min, max int) string {
	minHex := decToHex(min)
	maxHex := decToHex(max)
	argHex := fmt.Sprintf("%s%s0000", maxHex, minHex)
	arg, err := strconv.ParseInt(argHex, 16, 32)
	if err != nil {
		return "0"
	}
	return strconv.FormatInt(arg, 10)
}

func decToHex(i int) string {
	return strconv.FormatInt(int64(i), 16)
}

func getThreshFromLog(log string) (min, max int, err error) {
	fail := fmt.Errorf("failed to parse log fragment")

	l := strings.Split(log, "Reading (hexadecimal values):")
	if len(l) < 2 {
		err = fail
		return
	}
	l = strings.Split(l[1], ",")

	val := []int{}
	for _, s := range l {
		i := strings.Index(s, "x")
		if i == -1 {
			continue
		}
		if len(s) != i+3 {
			err = fail
			return
		}
		logTrace.Printf("Found hex value:%s", s)
		var v int64
		h := s[i+1:]
		v, err = strconv.ParseInt(h, 16, 32)
		if err != nil {
			err = fail
			return
		}
		val = append(val, int(v))
	}
	if len(val) != 2 {
		err = fail
		return
	}
	return val[0], val[1], nil
}
