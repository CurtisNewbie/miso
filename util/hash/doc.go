// Hash based data structures and tool set.
//
// [RWMap] is map with [sync.RWMutex] embedded.
//
// [StrRWMap] is a string based, partitioned map, internally managing multiple [RWMap] to future enhance parallel performance.
//
// [Set] is a HashSet data structure backed by a map.
package hash
