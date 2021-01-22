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
// along tihe this program. If not, see <https://www.gnu.org/licenses/>.

// +build darwin

package main

import (
	"fmt"
)

func init() {
	threshEndpoints = append(threshEndpoints, threshDriver{splitThreshEndpoint{zeroGetter{}, errSetter{}}})
}

// splitThreshEndpoint is a wmiDriver that has really differing
// ways of reading and writing
type splitThreshEndpoint struct {
	getter threshGetter
	setter threshSetter
}

func (ste splitThreshEndpoint) write(min, max int) error {
	return ste.setter.set(min, max)
}

func (ste splitThreshEndpoint) get() (min, max int, err error) {
	return ste.getter.get()
}

// threshGetter has a way to read battery thresholds
type threshGetter interface {
	get() (min, max int, err error)
}

// threshSetter has a way to set battery thresholds
type threshSetter interface {
	set(min, max int) error
}

// zeroGetter always returns 0 100
type zeroGetter struct{}

func (_ zeroGetter) get() (min, max int, err error) {
	return 0, 100, nil
}

// errSetter always returs error
type errSetter struct{}

func (_ errSetter) set(_, _ int) error {
	return fmt.Errorf("not implemented")
}
