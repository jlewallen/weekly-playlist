package main

import (
	"fmt"
	"strings"
	"time"
)

type DateOnly struct {
	time.Time
}

const doLayout = "2006-01-02"

func (do *DateOnly) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		do.Time = time.Time{}
		return
	}
	do.Time, err = time.Parse(doLayout, s)
	return
}

func (do *DateOnly) MarshalJSON() ([]byte, error) {
	if do.Time.UnixNano() == nilTime {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", do.Time.Format(doLayout))), nil
}

var nilTime = (time.Time{}).UnixNano()

func (do *DateOnly) IsSet() bool {
	return do.UnixNano() != nilTime
}
