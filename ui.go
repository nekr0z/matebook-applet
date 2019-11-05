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
	"github.com/andlabs/ui"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var (
	customWindow *ui.Window
	mainWindow   *ui.Window
)

func launchUI() {
	logTrace.Println("Setting up GUI...")
	mainWindow = ui.NewWindow("matebook-applet", 480, 360, false)
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

	batteryGroup := ui.NewGroup("")
	batteryGroup.SetMargined(true)
	if config.thresh == nil {
		logTrace.Println("no access to BP information, not showing the corresponding UI")
	} else {
		vbox.Append(batteryGroup, false)
		batteryGroup.SetTitle(getStatus())
	}

	batteryVbox := ui.NewVerticalBox()
	batteryVbox.SetPadded(true)
	if config.thresh != nil && config.thresh.isWritable() {
		batteryGroup.SetChild(batteryVbox)
	} else {
		logTrace.Println("BP endpoint read-only, not showing BP buttons")
	}

	offButton := ui.NewButton(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "SetOff", Other: "Off"}}))
	offButton.OnClicked(func(*ui.Button) {
		logTrace.Println("Off button clicked")
		setThresholds(0, 100)
		batteryGroup.SetTitle(getStatus())
	})
	batteryVbox.Append(offButton, false)

	travelButton := ui.NewButton(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "SetTravel", Other: "Travel"}}))
	travelButton.OnClicked(func(*ui.Button) {
		logTrace.Println("Travel button clicked")
		setThresholds(95, 100)
		batteryGroup.SetTitle(getStatus())
	})
	batteryVbox.Append(travelButton, false)

	officeButton := ui.NewButton(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "SetOffice", Other: "Office"}}))
	officeButton.OnClicked(func(*ui.Button) {
		logTrace.Println("Office button clicked")
		setThresholds(70, 90)
		batteryGroup.SetTitle(getStatus())
	})
	batteryVbox.Append(officeButton, false)

	homeButton := ui.NewButton(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "SetHome", Other: "Home"}}))
	homeButton.OnClicked(func(*ui.Button) {
		logTrace.Println("Home button clicked")
		setThresholds(40, 70)
		batteryGroup.SetTitle(getStatus())
	})
	batteryVbox.Append(homeButton, false)

	customButton := ui.NewButton(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "SetCustom", Other: "Custom"}}))
	var customButtonOnClicked func(*ui.Button)
	customButtonOnClicked = func(*ui.Button) {
		logTrace.Println("Custom button clicked")
		go func() {
			customButton.OnClicked(func(*ui.Button) {})
			ch := make(chan struct{})
			ui.QueueMain(func() { customThresholds(ch) })
			<-ch
			batteryGroup.SetTitle(getStatus())
			customButton.OnClicked(customButtonOnClicked)
		}()
	}
	customButton.OnClicked(customButtonOnClicked)
	batteryVbox.Append(customButton, false)

	fnlockGroup := ui.NewGroup("")
	fnlockGroup.SetMargined(true)
	if config.fnlock == nil {
		logTrace.Println("no access to Fn-Lock setting, not showing the corresponding GUI")
	} else {
		vbox.Append(fnlockGroup, false)
		fnlockGroup.SetTitle(getFnlockStatus())
	}

	fnlockVbox := ui.NewVerticalBox()
	fnlockVbox.SetPadded(true)
	fnlockGroup.SetChild(fnlockVbox)

	fnlockToggle := ui.NewButton(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "DoToggle", Other: "Toggle"}}))
	fnlockToggle.OnClicked(func(*ui.Button) {
		logTrace.Println("Fnlock toggle button clicked")
		config.fnlock.toggle()
		fnlockGroup.SetTitle(getFnlockStatus())
	})
	if config.fnlock != nil && config.fnlock.isWritable() {
		fnlockVbox.Append(fnlockToggle, false)
	} else {
		logTrace.Println("Fn-Lock setting read-only, not showing the button")
	}

	mainWindow.Show()
}

func customThresholds(ch chan struct{}) {
	logTrace.Println("Launching custom thresholds window")
	min, max, err := config.thresh.get()
	if err != nil {
		logWarning.Println(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CantReadBattery", Other: "failed to get thresholds"}}))
	}
	customWindow = ui.NewWindow(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "CustomWindowTitle", Other: "Charging thresholds"}}), 640, 240, false)
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
	minLabel := ui.NewLabel(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "MinThresholdExplain", Other: "MIN: the battery won't be charged unless it is lower than this level when AC is plugged"}}))
	vbox.Append(minLabel, false)
	vbox.Append(maxSlider, false)
	maxLabel := ui.NewLabel(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "MaxThresholdExplain", Other: "MAX: the battery won't be charged above this level"}}))
	vbox.Append(maxLabel, false)
	setButton := ui.NewButton(localizer.MustLocalize(&i18n.LocalizeConfig{DefaultMessage: &i18n.Message{ID: "DoSet", Other: "Set"}}))
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
