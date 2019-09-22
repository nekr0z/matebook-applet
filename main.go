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
	"flag"
	"github.com/andlabs/ui"
	"github.com/getlantern/systray"
	"io"
	"io/ioutil"
	"log"
	"os"
)

var (
	logTrace            *log.Logger
	logInfo             *log.Logger
	logWarning          *log.Logger
	logError            *log.Logger
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
	flag.BoolVar(&config.wait, "wait", false, "wait for driver to set battery thresholds (obsolete)")
	flag.BoolVar(&saveValues, "s", true, "save values for persistence (deprecated)") // TODO: remove in v3
	flag.BoolVar(&noSaveValues, "n", false, "do not save values")
	flag.BoolVar(&config.useScripts, "r", true, "use fnlock and batpro scripts if all else fails") // TODO: default to false in v3
	flag.BoolVar(&config.windowed, "w", false, "windowed mode")
	flag.Parse()

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
