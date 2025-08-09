package util

import (
	"sync"
	"testing"
)

func BenchmarkGetIdPerf(b *testing.B) {
	// SnowflakeId
	// BenchmarkGetIdPerf-8    16170626                73.32 ns/op           40 B/op          2 allocs/op
	//
	// ulid
	// BenchmarkGetIdPerf-8    12000014                99.03 ns/op           48 B/op          2 allocs/op
	b.SetBytes(int64(len(GenId())))
	for range b.N {
		_ = GenId()
	}
}

func TestGetId(t *testing.T) {
	var set Set[string] = NewSet[string]()
	var mu sync.Mutex

	threadCnt := 50
	loopCnt := 1000

	var wg sync.WaitGroup
	for j := 0; j < threadCnt; j++ {
		wg.Add(1)
		go func(idSet *Set[string], threadId int) {
			defer wg.Done()
			for i := 0; i < loopCnt; i++ {
				id := SnowflakeId()

				mu.Lock()
				if idSet.Has(id) {
					t.Errorf("[%d] Map already contains id: %s", threadId, id)
					return
				} else {
					idSet.Add(id)
					t.Logf("[%d] id: %s", threadId, id)
				}
				mu.Unlock()
			}
		}(&set, j)
	}

	wg.Wait()
}
