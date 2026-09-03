package memoryservice

import (
	"slices"
	"testing"

	"github.com/runethread/core/internal/memory"
)

func TestLegalOperationsAdvertiseResolveOnlyForUnresolvedOpenLoops(t *testing.T) {
	cases := []struct {
		name        string
		status      string
		wantResolve bool
	}{
		{name: "open", status: "open", wantResolve: true},
		{name: "blocked", status: "blocked", wantResolve: true},
		{name: "deferred", status: "deferred", wantResolve: true},
		{name: "resolved", status: "resolved", wantResolve: false},
		{name: "cancelled", status: "cancelled", wantResolve: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := tc.status
			got := legalOperations([]Document{{Memory: memory.Memory{
				Type:           "open_loop",
				Lifecycle:      "active",
				OpenLoopStatus: &status,
			}}})
			if slices.Contains(got, "resolve") != tc.wantResolve {
				t.Fatalf("legal operations = %v, resolve presence want %v", got, tc.wantResolve)
			}
		})
	}
}
