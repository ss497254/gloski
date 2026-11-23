package checks

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"strings"
)

// StableID identifies one probe definition. Kind, target and accepted status
// codes are hashed; schedule, timeout and certificate verification are not, so
// tuning those updates the same series in check_results. The prefix puts
// probes in the Connectivity column of the fleet matrix.
func StableID(kind ProbeKind, target string, accepted []int) string {
	statuses := append([]int(nil), accepted...)
	sort.Ints(statuses)
	parts := []string{string(kind), target}
	for _, status := range statuses {
		parts = append(parts, strconv.Itoa(status))
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(strings.Join(parts, "|")))
	return fmt.Sprintf("connect_%s_%08x", kind, hash.Sum32())
}
