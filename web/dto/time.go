package dto

import (
	"strings"
	"time"
)

/* same as time.Time but will be serialized using format '02/01/2006 15:04' */
type WTime time.Time

func (s WTime) MarshalJSON() ([]byte, error) {
	t := time.Time(s)

	/*
		Why they format time like this, it's frustrating :D
		https://stackoverflow.com/questions/20234104/how-to-format-current-time-using-a-yyyymmddhhmmss-format
	*/
	return []byte(t.Format(`"02/01/2006 15:04"`)), nil
}

/* same as time.Time but will be serialized/deserialized using format '2006-01-02 15:04:05' */
type TTime time.Time

func (t TTime) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)

	// yyyy/mm/dd hh:mm:ss
	return []byte(tt.Format(`"2006-01-02 15:04:05"`)), nil
}

func (t *TTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")

	pt, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
	if err != nil {
		return err
	}
	*t = TTime(pt)
	return nil
}
