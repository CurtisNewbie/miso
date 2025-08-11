package idutil

import "github.com/oklog/ulid/v2"

// Generate random id with 27 characters with extra prefix.
//
// Thread safe. Can be used in distributed environment.
func Id(prefix string) (id string) {
	return prefix + New()
}

// Generate random id with 27 characters
//
// Thread safe. Can be used in distributed environment.
func New() (id string) {
	// prefix "2" is to make sure the generated ids are always ordered after the previously used [SnowflakeId].
	// e.g., '1863535532687360327197' (snowflake) vs '2' + '01K2770MEFQ8EGE7QB750G6M5M'
	return "2" + ulid.Make().String()
}
