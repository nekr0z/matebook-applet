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
	//	"github.com/andlabs/ui"
	"github.com/getlantern/systray"
	"runtime"
)

// TODO: systray crashes on OS X
func main() {
	runtime.LockOSThread()
	systray.Run(onReady, onExit)
}

/*
func main() {
	doInit()
	if err := ui.Main(launchUI); err != nil {
		logError.Println(err)
	}
}
*/
