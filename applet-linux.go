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

// +build !darwin

package main

import (
	"github.com/andlabs/ui"
	"github.com/getlantern/systray"
)

func guiThread(mQuit, mCustom, mStatus *systray.MenuItem) {
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
}
