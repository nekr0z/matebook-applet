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
	"github.com/andlabs/ui"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"io/ioutil"
	"os"
)

const (
	defaultIcon = "/matebook-applet.png"
)

var (
	appQuit = make(chan struct{})
)

func onReady() {
	logTrace.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "PreparingTray", Other: "Setting up menu..."}}))
	systraySetIcon(getIcon(iconPath, defaultIcon))
	mStatus := systrayAddMenuItem("", "")
	systrayAddSeparator()
	mOff := systrayAddMenuItem(localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "StatusOff"}), "Switch off battery protection")
	mTravel := systrayAddMenuItem(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "DoTravel", Other: "TRAVEL (95%-100%)"}}), "Set battery protection to TRAVEL")
	mOffice := systrayAddMenuItem(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "DoOffice", Other: "OFFICE (70%-90%)"}}), "Set battery protection to OFFICE")
	mHome := systrayAddMenuItem(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "DoHome", Other: "HOME (40%-70%)"}}), "Set battery protection to HOME")
	mCustom := systrayAddMenuItem(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "DoCustom", Other: "CUSTOM"}}), "Set custom battery protection thresholds")
	systrayAddSeparator()
	mFnlock := systrayAddMenuItem("", "")
	systrayAddSeparator()
	mQuit := systrayAddMenuItem(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "Quit", Other: "Quit"}}), "Quit the applet")
	if config.thresh == nil {
		mStatus.Hide()
		logTrace.Println("no access to BP information, not showing it")
	} else {
		mStatus.SetTitle(getStatus())
	}
	if config.thresh == nil || !config.thresh.isWritable() {
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
				systrayQuit()
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
	logInfo.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "AppletExit", Other: "Exiting the applet..."}}))
	os.Exit(0)
}

func onExit() {
}

func getIcon(pth, dflt string) []byte {
	b, err := ioutil.ReadFile(pth)
	if err != nil {
		logInfo.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "NoCustomIcon", Other: "Couldn't get custom icon, falling back to default"}}))
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
		logInfo.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "GotCustomIcon", Other: "Successfully loaded custom icon from {{.Path}}"}, TemplateData: map[string]interface{}{"Path": pth}}))
	}
	return b
}
