// Package atom is a package for time processing.
//
// The core type in this package is [Time]. [Time] is an enhanced wrapper of [time.Time].
//
// [Time] implements [sql.Scanner] and [driver.Valuer] for database values and [json.Marshaler] and [json.Unmarshaler] for json processing.
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
// Marshaling and unmarshaling behaviour are fully customizable.
package atom
