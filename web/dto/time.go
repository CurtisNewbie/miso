package dto

import (
	"errors"
	"time"
)

/* same as time.Time but will be serialized using format '02/01/2006 15:04' */
type WTime time.Time

func (s WTime) MarshalJSON() ([]byte, error) {
	t := time.Time(s)
	if y := t.Year(); y < 0 || y >= 10000 {
		return nil, errors.New("Time.MarshalJSON: year outside of range [0,9999]")
	}

	/*
		Why they format time like this, it's frustrating :D
		https://stackoverflow.com/questions/20234104/how-to-format-current-time-using-a-yyyymmddhhmmss-format
	*/
	return []byte(t.Format(`"02/01/2006 15:04"`)), nil
}
