// Package for time processing.
//
// The core type in this package is [Time]. [Time] is an enhanced wrapper of [time.Time]. You can use [Time] directly in your codebase
// or only use it as a tool for [time.Time] processing, e.g.,
//
//	var monday time.Time = atom.WrapTime(time.Now()).StartOfWeek(time.Monday).Unwrap()
//
// [Time] implements [sql.Scanner] and [driver.Valuer] for database values and [json.Marshaler] and [json.Unmarshaler] for json processing.
//
// Marshaling and unmarshaling behaviours are fully customizable.
//
// Use [SetTimeMarshalFormat] to change the marshal format, by default [Time] is marshalled as millseconds since unix epoch.
//
// [Time] can be unmarshaled from various formats, e.g,
//   - [time.RFC3339]
//   - [time.RFC3339Nano]
//   - `2006-01-02 15:04:05.999999`
//   - `2006-01-02`
//   - `2006-01-02T15:04:05.999999`
//   - `millseconds since unix epoch`
//   - `seconds since unix epoch`
//
// You can add extra unmarshalling formats using [AddTimeParseFormat], or overwrite the unmarshalling formats entirely using [SetTimeParseFormat].
package atom
