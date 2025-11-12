package snowflake

import (
	"sync"
	"testing"

	"github.com/curtisnewbie/miso/util/hash"
)

func TestGetId(t *testing.T) {
	var set hash.Set[string] = hash.NewSet[string]()
	var mu sync.Mutex

	threadCnt := 50
	loopCnt := 1000

	var wg sync.WaitGroup
	for j := 0; j < threadCnt; j++ {
		wg.Add(1)
		go func(idSet *hash.Set[string], threadId int) {
			defer wg.Done()
			for i := 0; i < loopCnt; i++ {
				id := Id()

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
