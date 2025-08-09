package util

// Generate Snowflake Id with extra prefix.
//
// The id consists of 64 bits long + 6 digits machine_code.
// The 64 bits long consists of: sign bit (1 bit) + timestamp (49 bits, ~1487.583 years) + sequenceNo (14 bits, 0~16383).
//
// The max value of Long is 9223372036854775807, which is a string with 19 characters, so the generated id will be of at most 25 characters.
//
// This func is thread-safe.
//
// Deprecated: Use [idutil.Id] instead.
func GenIdP(prefix string) (id string) {
	return prefix + GenId()
}

// Generate Snowflake Id.
//
// The id consists of 64 bits long + 6 digits machine_code.
// The 64 bits long consists of: sign bit (1 bit) + timestamp (49 bits, ~1487.583 years) + sequenceNo (14 bits, 0~16383).
//
// The max value of Long is 9223372036854775807, which is a string with 19 characters, so the generated id will be of at most 25 characters.
//
// This func is thread-safe.
//
// Deprecated: Use [idutil.New] instead.
func GenId() (id string) {
	return SnowflakeId()
}
