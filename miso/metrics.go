package miso

import (
	"fmt"
	"regexp"
	"runtime"
	"runtime/metrics"
	"sync"
)

const (
	goGCHeapTinyAllocsObjects               = "/gc/heap/tiny/allocs:objects"
	goGCHeapAllocsObjects                   = "/gc/heap/allocs:objects"
	goGCHeapFreesObjects                    = "/gc/heap/frees:objects"
	goGCHeapFreesBytes                      = "/gc/heap/frees:bytes"
	goGCHeapAllocsBytes                     = "/gc/heap/allocs:bytes"
	goGCHeapObjects                         = "/gc/heap/objects:objects"
	goGCHeapGoalBytes                       = "/gc/heap/goal:bytes"
	goMemoryClassesTotalBytes               = "/memory/classes/total:bytes"
	goMemoryClassesHeapObjectsBytes         = "/memory/classes/heap/objects:bytes"
	goMemoryClassesHeapUnusedBytes          = "/memory/classes/heap/unused:bytes"
	goMemoryClassesHeapReleasedBytes        = "/memory/classes/heap/released:bytes"
	goMemoryClassesHeapFreeBytes            = "/memory/classes/heap/free:bytes"
	goMemoryClassesHeapStacksBytes          = "/memory/classes/heap/stacks:bytes"
	goMemoryClassesOSStacksBytes            = "/memory/classes/os-stacks:bytes"
	goMemoryClassesMetadataMSpanInuseBytes  = "/memory/classes/metadata/mspan/inuse:bytes"
	goMemoryClassesMetadataMSPanFreeBytes   = "/memory/classes/metadata/mspan/free:bytes"
	goMemoryClassesMetadataMCacheInuseBytes = "/memory/classes/metadata/mcache/inuse:bytes"
	goMemoryClassesMetadataMCacheFreeBytes  = "/memory/classes/metadata/mcache/free:bytes"
	goMemoryClassesProfilingBucketsBytes    = "/memory/classes/profiling/buckets:bytes"
	goMemoryClassesMetadataOtherBytes       = "/memory/classes/metadata/other:bytes"
	goMemoryClassesOtherBytes               = "/memory/classes/other:bytes"
)

var (
	MetricsMemoryMatcher = regexp.MustCompile(`^/memory/.*`)

	memStatsCollector *MetricsCollector
)

func init() {
	SetDefProp(PropMetricsEnableMemStatsLogJob, false)
	SetDefProp(PropMetricsMemStatsLogJobCron, "0 */1 * * * *")

	RegisterBootstrapCallback(ComponentBootstrap{
		Name:  "BootstrapMetrics",
		Order: BootstrapOrderL4,
		Condition: func(app *MisoApp, rail Rail) (bool, error) {
			return app.Config().GetPropBool(PropMetricsEnableMemStatsLogJob), nil
		},
		Bootstrap: func(app *MisoApp, rail Rail) error {
			collector := NewMetricsCollector(DefaultMetricDesc(nil))
			memStatsCollector = &collector
			return ScheduleCron(Job{
				Name:            "MetricsMemStatLogJob",
				CronWithSeconds: true,
				Cron:            GetPropStr(PropMetricsMemStatsLogJobCron),
				Run: func(r Rail) error {
					memStatsCollector.Read()
					r.Infof("\n\n%s", SprintMemStats(memStatsCollector.MemStats()))
					return nil
				},
			})
		},
	})
}

// Collector of runtime/metrics.
//
// Use NewMetricsCollector() to create a new collector, the collector is thread-safe.
// Periodically call Read() to load metrics from runtime.
// Once the metrics are loaded, you can either use Value() or MemStats() func to access the values
// that you are interested in.
type MetricsCollector struct {
	sw        sync.RWMutex
	Desc      []metrics.Description
	Samples   []metrics.Sample
	SampleMap map[string]*metrics.Sample
}

// Create new MetricsCollector.
//
// MetricsCollector only supports Uint64 metrics, those that are not Uint64 kind, are simply ignored.
func NewMetricsCollector(descs []metrics.Description) MetricsCollector {
	samples := make([]metrics.Sample, 0, len(descs))
	sampleMap := make(map[string]*metrics.Sample)
	for _, d := range descs {
		samples = append(samples, metrics.Sample{Name: d.Name})
		sampleMap[d.Name] = &samples[len(samples)-1]
	}
	// Debugf("Samples: %+v", samples)
	// Debugf("SampleMap: %+v", sampleMap)
	return MetricsCollector{Desc: descs, Samples: samples, SampleMap: sampleMap}
}

func (m *MetricsCollector) Value(name string) uint64 {
	m.sw.RLock()
	defer m.sw.RUnlock()
	if s, ok := m.SampleMap[name]; ok {
		if s.Value.Kind() != metrics.KindUint64 {
			Warnf("value %v is not of kind uint64, but %v", name, s.Value.Kind())
			return 0
		}
		return s.Value.Uint64()
	} else {
		Debugf("metrics %v is not found from sample map", name)
	}
	return 0
}

func (m *MetricsCollector) MemStats() runtime.MemStats {
	// copied from github.com/prometheus/client_golang v1.4.0

	// Currently, MemStats adds tiny alloc count to both Mallocs AND Frees.
	// The reason for this is because MemStats couldn't be extended at the time
	// but there was a desire to have Mallocs at least be a little more representative,
	// while having Mallocs - Frees still represent a live object count.
	// Unfortunately, MemStats doesn't actually export a large allocation count,
	// so it's impossible to pull this number out directly.
	ms := runtime.MemStats{}
	tinyAllocs := m.Value(goGCHeapTinyAllocsObjects)
	ms.Mallocs = m.Value(goGCHeapAllocsObjects) + tinyAllocs
	ms.Frees = m.Value(goGCHeapFreesObjects) + tinyAllocs
	ms.TotalAlloc = m.Value(goGCHeapAllocsBytes)
	ms.Sys = m.Value(goMemoryClassesTotalBytes)
	ms.Lookups = 0 // Already always zero.
	ms.HeapAlloc = m.Value(goMemoryClassesHeapObjectsBytes)
	ms.Alloc = ms.HeapAlloc
	ms.HeapInuse = ms.HeapAlloc + m.Value(goMemoryClassesHeapUnusedBytes)
	ms.HeapReleased = m.Value(goMemoryClassesHeapReleasedBytes)
	ms.HeapIdle = ms.HeapReleased + m.Value(goMemoryClassesHeapFreeBytes)
	ms.HeapSys = ms.HeapInuse + ms.HeapIdle
	ms.HeapObjects = m.Value(goGCHeapObjects)
	ms.StackInuse = m.Value(goMemoryClassesHeapStacksBytes)
	ms.StackSys = ms.StackInuse + m.Value(goMemoryClassesOSStacksBytes)
	ms.MSpanInuse = m.Value(goMemoryClassesMetadataMSpanInuseBytes)
	ms.MSpanSys = ms.MSpanInuse + m.Value(goMemoryClassesMetadataMSPanFreeBytes)
	ms.MCacheInuse = m.Value(goMemoryClassesMetadataMCacheInuseBytes)
	ms.MCacheSys = ms.MCacheInuse + m.Value(goMemoryClassesMetadataMCacheFreeBytes)
	ms.BuckHashSys = m.Value(goMemoryClassesProfilingBucketsBytes)
	ms.GCSys = m.Value(goMemoryClassesMetadataOtherBytes)
	ms.OtherSys = m.Value(goMemoryClassesOtherBytes)
	ms.NextGC = m.Value(goGCHeapGoalBytes)

	// N.B. GCCPUFraction is intentionally omitted. This metric is not useful,
	// and often misleading due to the fact that it's an average over the lifetime
	// of the process.
	// See https://github.com/prometheus/client_golang/issues/842#issuecomment-861812034
	// for more details.
	ms.GCCPUFraction = 0
	return ms
}

func (m *MetricsCollector) Read() {
	m.sw.Lock()
	metrics.Read(m.Samples)
	m.sw.Unlock()
}

func DefaultMetricDesc(matcher *regexp.Regexp) []metrics.Description {
	nameFilter := func(name string) bool {
		if matcher == nil {
			return true
		}
		return matcher.MatchString(name)
	}
	kindFilter := func(kind metrics.ValueKind) bool {
		return kind == metrics.KindUint64
	}
	return FilterMetricDesc(nameFilter, kindFilter)
}

func FilterMetricDesc(nameFilter func(name string) bool, kindFilter func(kind metrics.ValueKind) bool) []metrics.Description {
	if nameFilter == nil && kindFilter == nil {
		return metrics.All()
	}
	var descs []metrics.Description
	for _, d := range metrics.All() {
		if nameFilter != nil && !nameFilter(d.Name) {
			continue
		}
		if kindFilter != nil && !kindFilter(d.Kind) {
			continue
		}
		descs = append(descs, d)
	}
	return descs
}

func toMbStr(b uint64) string {
	return fmt.Sprintf("%.3f mb", float64(b)/1024/1024)
}

func SprintMemStats(ms runtime.MemStats) string {
	type NamedStats struct {
		Name  string
		Stats any
	}
	l := []NamedStats{
		{"Heap Alloc", toMbStr(ms.HeapAlloc)},
		{"Stack Sys (Stack In-Use + OS Stack)", toMbStr(ms.StackSys)},
		{"Allocated Heap Objects (Cumulative)", ms.Mallocs},
		{"Alive Heap Objects", ms.HeapObjects},
	}
	s := ""
	for _, v := range l {
		s += fmt.Sprintf("%-40s %v\n", v.Name, v.Stats)
	}
	return s
}
