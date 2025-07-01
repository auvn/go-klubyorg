package tg

import "time"

var _loc *time.Location

func init() {
	var err error
	_loc, err = time.LoadLocation("Europe/Warsaw")
	if err != nil {
		panic(err)
	}
}

func timeNow() time.Time {
	return time.Now().In(_loc)
}

func timeUnix(ts int64) time.Time {
	return time.Unix(ts, 0).In(_loc)
}
