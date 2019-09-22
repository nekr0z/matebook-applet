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
	"github.com/andlabs/ui"
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
	threshKernelPath     = "/sys/class/power_supply/BAT"
	threshKernelMin      = "/charge_control_start_threshold"
	threshKernelMax      = "/charge_control_end_threshold"
)

var (
	logTrace            *log.Logger
	logInfo             *log.Logger
	logWarning          *log.Logger
	logError            *log.Logger
	waitForDriver       bool
	version             string = "custom-build"
	iconPath            string
	saveValues          bool
	noSaveValues        bool
	saveValuesPath      string = "/etc/default/huawei-wmi/"
	fnlockEndpoints            = []fnlockEndpoint{}
	threshEndpoints            = []threshEndpoint{}
	threshSaveEndpoints        = []threshDriver{
		threshDriver{threshDriverSingle{path: (saveValuesPath + "charge_control_thresholds")}},
		threshDriver{threshDriverSingle{path: (saveValuesPath + "charge_thresholds")}},
	}
	threshDriver1 = threshDriver{threshDriverSingle{path: "/sys/devices/platform/huawei-wmi/charge_thresholds"}}
	threshDriver2 = threshDriver{threshDriverSingle{path: "/sys/devices/platform/huawei-wmi/charge_control_thresholds"}}
	config        struct {
		fnlock     fnlockEndpoint
		thresh     threshEndpoint
		threshPers threshEndpoint
		wait       bool
		useScripts bool
		windowed   bool
	}
	appQuit      = make(chan struct{})
	customWindow *ui.Window
	mainWindow   *ui.Window
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

func init() {
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
	flag.BoolVar(&saveValues, "s", true, "save values for persistence (deprecated)") // TODO: remove in v3
	flag.BoolVar(&noSaveValues, "n", false, "do not save values")
	flag.BoolVar(&config.useScripts, "r", true, "use fnlock and batpro scripts if all else fails") // TODO: default to false in v3
	flag.BoolVar(&config.windowed, "w", false, "windowed mode")
	flag.Parse()

	config.wait = waitForDriver

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
		config.fnlock = fnlck
		if fnlck.isWritable() {
			logInfo.Println("Found writable fnlock endpoint, will use it")
			break
		}
	}

	// need to find working threshold interface (if any)
	for _, thresh := range threshEndpoints {
		_, _, err := thresh.get()
		if err != nil {
			continue
		}
		config.thresh = thresh
		if thresh.isWritable() {
			logInfo.Println("Found writable battery thresholds endpoint, will use it")
			break
		}
	}

	if config.thresh != nil || config.fnlock != nil {
		if config.windowed {
			if err := ui.Main(launchUI); err != nil {
				logError.Println(err)
			}
		} else {
			systray.Run(onReady, onExit)
		}
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
	mCustom := systray.AddMenuItem("CUSTOM", "Set custom battery protection thresholds")
	systray.AddSeparator()
	mFnlock := systray.AddMenuItem("", "")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the applet")
	if config.thresh == nil {
		mStatus.Hide()
		logTrace.Println("no access to BP information, not showing it")
	} else {
		mStatus.SetTitle(getStatus())
	}
	if !config.thresh.isWritable() {
		mOff.Hide()
		mTravel.Hide()
		mOffice.Hide()
		mHome.Hide()
		mCustom.Hide()
		logTrace.Println("no way to change BP settings, not showing the corresponding GUI")
	}
	if config.fnlock == nil {
		mFnlock.Hide()
		logTrace.Println("no access to Fn-Lock setting, not showing its GUI")
	} else {
		mFnlock.SetTitle(getFnlockStatus())
	}
	if noSaveValues {
		saveValues = false
	} else {
		logTrace.Println("looking for endpoint to save thresholds to...")
		for _, ep := range threshSaveEndpoints {
			_, _, err := ep.get()
			if err == nil {
				logInfo.Println("Persistence thresholds values endpoint found.")
				config.threshPers = ep
				break
			}
		}
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
				setThresholds(0, 100)
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
				config.fnlock.toggle()
				mFnlock.SetTitle(getFnlockStatus())
			case <-appQuit:
				logTrace.Println("Shutting down systray applet")
				systray.Quit()
				return
			}
		}
	}()
	logTrace.Println("Setting up GUI thread...")
	if err := ui.Main(func() {
		ui.OnShouldQuit(func() bool {
			customWindow.Destroy()
			logTrace.Println("ready to quit GUI thread")
			return true
		})
		go func() {
			for {
				select {
				case <-mCustom.ClickedCh:
					logTrace.Println("Got a click on BP CUSTOM")
					ch := make(chan struct{})
					ui.QueueMain(func() { customThresholds(ch) })
					<-ch
					mStatus.SetTitle(getStatus())
				case <-appQuit:
					return
				}
			}
		}()
		go func() {
			<-mQuit.ClickedCh
			logTrace.Println("Got a click on Quit")
			ui.Quit()
			close(appQuit)
		}()
	}); err != nil {
		logError.Println(err)
	}
	logInfo.Println("Exiting the applet...")
	os.Exit(0)
}

func launchUI() {
	logTrace.Println("Setting up GUI...")
	mainWindow = ui.NewWindow("matebook-applet", 640, 480, false)
	mainWindow.OnClosing(func(*ui.Window) bool {
		ui.Quit()
		return true
	})
	ui.OnShouldQuit(func() bool {
		customWindow.Destroy()
		mainWindow.Destroy()
		return true
	})

	mainWindow.SetMargined(true)
	vbox := ui.NewVerticalBox()
	vbox.SetPadded(true)
	mainWindow.SetChild(vbox)

	batteryGroup := ui.NewGroup("Battery protection thresholds")
	batteryGroup.SetMargined(true)
	vbox.Append(batteryGroup, false)

	batteryVbox := ui.NewVerticalBox()
	batteryGroup.SetChild(batteryVbox)

	customButton := ui.NewButton("Custom")
	var customButtonOnClicked func(*ui.Button)
	customButtonOnClicked = func(*ui.Button) {
		logTrace.Println("Custom button clicked")
		go func() {
			customButton.OnClicked(func(*ui.Button) {})
			ch := make(chan struct{})
			ui.QueueMain(func() { customThresholds(ch) })
			<-ch
			customButton.OnClicked(customButtonOnClicked)
		}()
	}
	customButton.OnClicked(customButtonOnClicked)
	batteryVbox.Append(customButton, false)

	mainWindow.Show()
}

func customThresholds(ch chan struct{}) {
	logTrace.Println("Launching custom thresholds window")
	min, max, err := config.thresh.get()
	if err != nil {
		logWarning.Println("Failed to get thresholds")
	}
	customWindow = ui.NewWindow("Custom battery thresholds", 640, 240, false)
	customWindow.OnClosing(func(*ui.Window) bool {
		close(ch)
		return true
	})
	customWindow.SetMargined(true)
	vbox := ui.NewVerticalBox()
	vbox.SetPadded(true)
	hbox := ui.NewHorizontalBox()
	hbox.SetPadded(true)
	customWindow.SetChild(vbox)
	minSlider := ui.NewSlider(0, 100)
	maxSlider := ui.NewSlider(0, 100)
	minSlider.OnChanged(func(*ui.Slider) {
		if minSlider.Value() > maxSlider.Value() {
			minSlider.SetValue(maxSlider.Value())
		}
	})
	maxSlider.OnChanged(func(*ui.Slider) {
		if maxSlider.Value() < minSlider.Value() {
			maxSlider.SetValue(minSlider.Value())
		}
	})
	vbox.Append(minSlider, false)
	minLabel := ui.NewLabel("MIN: the battery won't be charged unless it is lower than this level when AC is plugged")
	vbox.Append(minLabel, false)
	vbox.Append(maxSlider, false)
	maxLabel := ui.NewLabel("MAX: the battery won't be charged above this level")
	vbox.Append(maxLabel, false)
	setButton := ui.NewButton("Set")
	setButton.OnClicked(func(*ui.Button) {
		setThresholds(minSlider.Value(), maxSlider.Value())
		customWindow.Destroy()
		close(ch)
	})
	vbox.Append(hbox, false)
	hbox.Append(setButton, true)
	minSlider.SetValue(min)
	maxSlider.SetValue(max)
	customWindow.Show()
}

func (drv fnlockDriver) get() (bool, error) {
	if _, err := os.Stat(drv.path); err != nil {
		logTrace.Printf("Couldn't access %q.", drv.path)
		return false, err
	}

	val, err := ioutil.ReadFile(drv.path)
	if err != nil {
		logError.Println("Could not read Fn-Lock state from driver interface")
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
			logTrace.Println("successful write to driver interface")
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
}

func (s threshScript) isWritable() bool {
	return true
}

func (d threshDriver) isWritable() bool {
	logTrace.Println("Checking if the threshold endpoint is writable...")
	return d.checkWritable()
}

func (d fnlockDriver) isWritable() bool {
	logTrace.Println("Checking if the fnlock endpoint is writable...")
	return d.checkWritable()
}

func (s fnlockScript) isWritable() bool {
	return true
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

func (drv threshDriverSingle) get() (min, max int, err error) {
	if _, err = os.Stat(drv.path); err != nil {
		logTrace.Printf("Couldn't access %q.", drv.path)
		return
	}

	var values [2]string
	val, err := ioutil.ReadFile(drv.path)
	if err != nil {
		logError.Println("Failed to get thresholds from driver interface")
		logTrace.Println(err)
	}

	valuesReceived := strings.Split(strings.TrimSpace(string(val)), " ")
	logTrace.Println("got values from interface:", valuesReceived)
	if len(valuesReceived) != 2 {
		logError.Println("Can not make sence of driver interface value", val)
		return
	} else {
		for i := 0; i < 2; i++ {
			values[i] = valuesReceived[i]
		}
		min, max, err = valuesAtoi(values[0], values[1])
		return
	}
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
		logError.Println("Failed to set min threshold")
		return err
	}
	if err := ioutil.WriteFile(drv.pathMax, []byte(strconv.Itoa(max)), 0664); err != nil {
		logError.Println("Failed to set max threshold")
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
		logError.Println("Failed to get min threshold from driver interface")
		logTrace.Println(err)
		return
	}
	values[0] = string(val)
	val, err = ioutil.ReadFile(drv.pathMax)
	if err != nil {
		logError.Println("Failed to get max threshold from driver interface")
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
		logError.Println("Failed to set thresholds")
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
		logError.Println("Failed to get battery protection status from script")
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
		logError.Println("Failed to set thresholds")
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
	var r string
	min, max, err := config.thresh.get()
	if err != nil {
		r = "ERROR: can not get BP status!"
	} else {
		if min >= 0 && min <= 100 && max != 0 && max <= 100 && min <= max {
			r = "Battery protection mode "
			switch {
			case min == 0 && max == 100:
				r = "Battery protection OFF"
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
	}
	return r
}

func getFnlockStatus() string {
	var r string
	state, err := config.fnlock.get()
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
