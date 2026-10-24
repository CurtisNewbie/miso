# Miso Async Context

The async utility package gives services a way to run work off the caller's goroutine and collect its result later. Its vocabulary separates the work, the handle to the result, and the group that awaits several results.

## Language

**Task**:
A function that produces a value and an error when run off the caller's goroutine. _Avoid_: job, work item, closure.

**Future**:
The handle to a task's eventual result. It is completed exactly once, and every reader observes that one result. _Avoid_: promise, result handle.

**Completion**:
The single moment a future publishes its value and error and wakes every waiter and callback. _Avoid_: resolution, settle.

**Await group**:
A collection of futures awaited together. It is sealed on the first await, after which no new tasks join it. _Avoid_: batch, future list.

**Dropped task**:
A task a pool refuses to run because it is full or stopped and its fallback discards work. A dropped task never produces a result of its own, so its future reports a drop error. _Avoid_: rejected task, lost task.

**Stop barrier**:
The guarantee that stopping a pool blocks new submissions, drains the tasks already queued, and only then lets the stop call return. _Avoid_: shutdown, teardown.
