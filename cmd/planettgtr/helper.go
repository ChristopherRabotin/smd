package main

import (
	"fmt"
	"time"
)

type positionsStatus uint8

const (
	leading positionsStatus = iota + 1
	trailing
)

func (p positionsStatus) String() string {
	switch p {
	case leading:
		return "leading"
	default:
		return "trailing"
	}
}

type result struct {
	succeeded bool
	status    positionsStatus
	launchDT  time.Time
	arrivalDT time.Time
}

func (r result) String() string {
	var status string
	if r.status == leading {
		status = "leading"
	} else {
		status = "trailing"
	}
	return fmt.Sprintf("ok? %+v\t%s\nlaunch: %s\tarrival: %s", r.succeeded, status, r.launchDT, r.arrivalDT)
}
