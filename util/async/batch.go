package async

import "sync"

type BatchTask[T any, V any] struct {
	parallel  int
	taskPipe  chan T
	workerWg  *sync.WaitGroup
	doConsume func(T)
	results   []BatchTaskResult[V]
	resultsMu *sync.Mutex
}

// Wait until all generated tasks are completed and close pipeline channel.
func (b *BatchTask[T, V]) Wait() []BatchTaskResult[V] {
	b.workerWg.Wait()
	defer close(b.taskPipe)
	return b.results
}

// Close underlying pipeline channel without waiting.
func (b *BatchTask[T, V]) Close() {
	close(b.taskPipe)
}

func (b *BatchTask[T, V]) preHeat() {
	for i := 0; i < b.parallel; i++ {
		go func() {
			for t := range b.taskPipe {
				b.doConsume(t)
			}
		}()
	}
}

// Generate task.
func (b *BatchTask[T, V]) Generate(task T) {
	b.workerWg.Add(1)
	b.taskPipe <- task
}

type BatchTaskResult[V any] struct {
	Result V
	Err    error
}

// Create a batch of concurrent task for one time use.
func NewBatchTask[T any, V any](parallel int, bufferSize int, consumer func(T) (V, error)) *BatchTask[T, V] {
	bt := &BatchTask[T, V]{
		parallel:  parallel,
		taskPipe:  make(chan T, bufferSize),
		workerWg:  &sync.WaitGroup{},
		results:   make([]BatchTaskResult[V], 0, bufferSize),
		resultsMu: &sync.Mutex{},
	}
	bt.doConsume = func(t T) {
		defer bt.workerWg.Done()
		v, err := consumer(t)
		r := BatchTaskResult[V]{Result: v, Err: err}
		bt.resultsMu.Lock()
		defer bt.resultsMu.Unlock()
		bt.results = append(bt.results, r)
	}
	bt.preHeat()
	return bt
}
