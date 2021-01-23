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

// +build linux

package main

import (
	"os/exec"
	"strconv"
)

func initEndpoints() {
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
