package core

import (
	"sync"
	"testing"
	"time"
)

func TestGetIdPerf(t *testing.T) {
	total := 10_000_000
	start := time.Now().UnixMilli()
	for i := 0; i < total; i++ {
		GenId()
	}
	end := time.Now().UnixMilli()
	t.Logf("time: %dms, total: %d id, perf: %.5fms each", end-start, total, float64(end-start)/float64(total))
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
				id := GenId()

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
