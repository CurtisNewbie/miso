package common

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

var (
	/** 1 January 2022 00:00:00 */
	startTime int64 = 1640995200000
	/** Machine Code for current instance */
	machineCode string = RandNum(6)
	/** Max number of bits for sequenceNo (0~16383) */
	maxSeqNoBits int64 = 14
	/** Mask for seqNo, 1|1 -> 1|100000000000000 -> 0|011111111111111 */
	seqNoMask int64 = ^(-1 << maxSeqNoBits)
	/** previous timestamp */
	timestamp int64 = 0
	/** previous seqNo */
	seqNo int64 = 0
	mu    sync.Mutex
)

// Overwrite the randomly generated machine code, machine code must be between 0 and 999999, at most 6 digits.
func SetMachineCode(code int) error {
	if code < 0 || code > 999999 {
		return fmt.Errorf("machindCode must be between 0 and 999999")
	}

	mu.Lock()
	defer mu.Unlock()
	machineCode = PadNum(code, 6)
	return nil
}

/*
Generate Id with prefix

The id consists of [64 bits long] + [6 digits machine_code]
The 64 bits long consists of: [sign bit (1 bit)] + [timestamp (49 bits, ~1487.583 years)] + [sequenceNo (14 bits, 0~16383)]

# The max value of Long is 9223372036854775807, which is a string with 19 characters, so the generated id will be of at most 25 characters

This func is thread-safe
*/
func GenIdP(prefix string) (id string) {
	id = prefix + GenId()
	return id
}

/*
Generate Id

The id consists of [64 bits long] + [6 digits machine_code]
The 64 bits long consists of: [sign bit (1 bit)] + [timestamp (49 bits, ~1487.583 years)] + [sequenceNo (14 bits, 0~16383)]

# The max value of Long is 9223372036854775807, which is a string with 19 characters, so the generated id will be of at most 25 characters

This func is thread-safe
*/
func GenId() (id string) {
	mu.Lock()
	defer mu.Unlock()

	currTimestamp := time.Now().UnixMilli()
	if currTimestamp == timestamp {
		seqNo = (seqNo + 1) & seqNoMask

		// we just assume that the concurrency wouldn't be so high, that it reaches 16383 within a millisecond
		// but if that happens, we will have to wait for the next timestamp
		if seqNo == 0 {
			for {
				// keep looping until next timestamp
				if currTimestamp = time.Now().UnixMilli(); currTimestamp != timestamp {
					resetSeqNo(currTimestamp)
					return fmtId()
				}
			}
		}
	} else {
		resetSeqNo(currTimestamp)
	}
	return fmtId()
}

func resetSeqNo(currTimestamp int64) {
	seqNo = 0
	timestamp = currTimestamp
}

func fmtId() (id string) {
	return strconv.FormatInt((((timestamp-startTime)<<maxSeqNoBits)|seqNo), 10) + machineCode
}
