package main

import (
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var messages = []*i18n.Message{
	&i18n.Message{
		ID:    "StatusOn",
		Other: "ON",
	},
	&i18n.Message{
		ID:    "StatusOff",
		Other: "OFF",
	},
	&i18n.Message{
		ID:    "StatusHome",
		Other: "HOME",
	},
	&i18n.Message{
		ID:    "StatusOffice",
		Other: "OFFICE",
	},
	&i18n.Message{
		ID:    "StatusTravel",
		Other: "TRAVEL",
	},
	&i18n.Message{
		ID:    "StatusCustom",
		Other: "CUSTOM ({{.Min}}%-{{.Max}}%)",
	},
	&i18n.Message{
		ID:    "CantReadFnlock",
		Other: "could not read Fn-Lock state from driver interface",
	},
	&i18n.Message{
		ID:    "CantSetBattery",
		Other: "failed to set thresholds",
	},
}
