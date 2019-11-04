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
	"github.com/BurntSushi/toml"
	"github.com/andlabs/ui"
	"github.com/cloudfoundry/jibber_jabber"
	"github.com/getlantern/systray"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var (
	logTrace     *log.Logger
	logInfo      *log.Logger
	logWarning   *log.Logger
	logError     *log.Logger
	version      = "custom-build"
	iconPath     string
	saveValues   bool
	noSaveValues bool
	localizer    *i18n.Localizer
	config       struct {
		fnlock     fnlockEndpoint
		thresh     threshEndpoint
		threshPers threshEndpoint
		wait       bool
		useScripts bool
		windowed   bool
	}
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
	flag.BoolVar(&saveValues, "s", false, "save values for persistence (deprecated)") // TODO: remove in v3
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

	// i18n init
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	if tr, err := assets.Open("/translations"); err == nil {
		if files, err := tr.Readdir(-1); err == nil {
			for _, file := range files {
				if ok, err := filepath.Match("active.*.toml", file.Name()); ok && err == nil {
					f, err := assets.Open(filepath.Join("/translations", file.Name()))
					if err == nil {
						b, err := ioutil.ReadAll(f)
						if err == nil {
							_, err := bundle.ParseMessageFileBytes(b, file.Name())
							if err != nil {
								logError.Printf("error reading translation file %s: %s", file.Name(), err)
							}
						}
						f.Close()
					}
				}
			}
		}
		tr.Close()
	}

	lang, err := jibber_jabber.DetectIETF()
	if err != nil {
		logWarning.Println("could not detect locale")
	}

	localizer = i18n.NewLocalizer(bundle, lang)
	logTrace.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "DetectedLocale", Other: "detected locale: {{.Locale}}"}, TemplateData: map[string]interface{}{"Locale": lang}}))

	logInfo.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "AppletVersion", Other: "matebook-applet version {{.Version}}"}, TemplateData: map[string]interface{}{"Version": version}}))

	// need to find working fnlock interface (if any)
	for _, fnlck := range fnlockEndpoints {
		_, err := fnlck.get()
		if err != nil {
			continue
		}
		config.fnlock = fnlck
		if fnlck.isWritable() {
			logInfo.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "FoundFnlock", Other: "Found writable fnlock endpoint, will use it"}}))
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
			logInfo.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "FoundBattery", Other: "Found writable battery thresholds endpoint, will use it"}}))
			break
		}
	}

	if saveValues {
		logWarning.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "OptionSDeprecated", Other: "-s option is deprecated, applet is now saving values for persistence by default"}}))
	}

	if !noSaveValues {
		logTrace.Println("looking for endpoint to save thresholds to...")
		for _, ep := range threshSaveEndpoints {
			_, _, err := ep.get()
			if err == nil {
				logInfo.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "FoundBatteryPers", Other: "Persistence thresholds values endpoint found."}}))
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
		logError.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "NothingToWorkWith", Other: "Neither a supported version of Huawei-WMI driver, nor any of the required scripts are properly installed, see README.md#installation-and-setup for instructions"}}))
	}
}
