package main

import (
	"fmt"
	"github.com/getlantern/systray"
	"io/ioutil"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(getIcon("assets/Iconsmind-Outline-Battery-Charge.ico"))
	systray.SetTitle("Protected or not")
	systray.SetTooltip("Protection details")
}

func onExit() {
	// cleanup (maybe TODO)
}

func getIcon(s string) []byte {
	b, err := ioutil.ReadFile(s)
	if err != nil {
		fmt.Print(err)
	}
	return b
}
