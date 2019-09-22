package main

import (
	"github.com/andlabs/ui"
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
	batteryGroup.SetTitle(getStatus())
	batteryGroup.SetMargined(true)
	vbox.Append(batteryGroup, false)

	batteryVbox := ui.NewVerticalBox()
	batteryVbox.SetPadded(true)
	batteryGroup.SetChild(batteryVbox)

	offButton := ui.NewButton("Off")
	offButton.OnClicked(func(*ui.Button) {
		logTrace.Println("Off button clicked")
		setThresholds(0, 100)
		batteryGroup.SetTitle(getStatus())
	})
	batteryVbox.Append(offButton, false)

	travelButton := ui.NewButton("Travel")
	travelButton.OnClicked(func(*ui.Button) {
		logTrace.Println("Travel button clicked")
		setThresholds(95, 100)
		batteryGroup.SetTitle(getStatus())
	})
	batteryVbox.Append(travelButton, false)

	officeButton := ui.NewButton("Office")
	officeButton.OnClicked(func(*ui.Button) {
		logTrace.Println("Office button clicked")
		setThresholds(70, 90)
		batteryGroup.SetTitle(getStatus())
	})
	batteryVbox.Append(officeButton, false)

	homeButton := ui.NewButton("Home")
	homeButton.OnClicked(func(*ui.Button) {
		logTrace.Println("Home button clicked")
		setThresholds(40, 70)
		batteryGroup.SetTitle(getStatus())
	})
	batteryVbox.Append(homeButton, false)

	customButton := ui.NewButton("Custom")
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
	fnlockGroup.SetTitle(getFnlockStatus())
	fnlockGroup.SetMargined(true)
	vbox.Append(fnlockGroup, false)

	fnlockVbox := ui.NewVerticalBox()
	fnlockVbox.SetPadded(true)
	fnlockGroup.SetChild(fnlockVbox)

	fnlockToggle := ui.NewButton("Toggle")
	fnlockToggle.OnClicked(func(*ui.Button) {
		logTrace.Println("Fnlock toggle button clicked")
		config.fnlock.toggle()
		fnlockGroup.SetTitle(getFnlockStatus())
	})
	fnlockVbox.Append(fnlockToggle, false)

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
