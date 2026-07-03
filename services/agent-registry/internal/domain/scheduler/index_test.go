// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func cand(key string, capIDs ...string) Candidate {
	caps := make([]Capability, 0, len(capIDs))
	for _, id := range capIDs {
		caps = append(caps, Capability{ID: id})
	}
	return Candidate{Key: key, Name: key, Endpoint: key + ".default.svc:50061", Ready: true, Capabilities: caps}
}

func TestIndex_UpsertAndCandidates(t *testing.T) {
	idx := NewIndex()
	idx.Upsert(cand("default/b", "review"))
	idx.Upsert(cand("default/a", "review", "summarize"))

	got := idx.Candidates("review")
	if len(got) != 2 {
		t.Fatalf("candidates(review) = %d, want 2", len(got))
	}
	// Deterministic key order.
	if got[0].Key != "default/a" || got[1].Key != "default/b" {
		t.Errorf("order = %s,%s want default/a,default/b", got[0].Key, got[1].Key)
	}
	if n := len(idx.Candidates("summarize")); n != 1 {
		t.Errorf("candidates(summarize) = %d, want 1", n)
	}
	if idx.Candidates("unknown") != nil {
		t.Error("candidates(unknown) should be nil")
	}
	if idx.Len() != 2 {
		t.Errorf("Len = %d, want 2", idx.Len())
	}
}

// TestIndex_UpsertReplacesStaleLinks covers the capability-set change on
// re-registration: links from the previous version must be dropped.
func TestIndex_UpsertReplacesStaleLinks(t *testing.T) {
	idx := NewIndex()
	idx.Upsert(cand("default/a", "review"))
	idx.Upsert(cand("default/a", "summarize")) // capability changed

	if got := idx.Candidates("review"); got != nil {
		t.Errorf("stale link survived: candidates(review) = %v", got)
	}
	if n := len(idx.Candidates("summarize")); n != 1 {
		t.Errorf("candidates(summarize) = %d, want 1", n)
	}
	if idx.Len() != 1 {
		t.Errorf("Len = %d, want 1", idx.Len())
	}
}

func TestIndex_Delete(t *testing.T) {
	idx := NewIndex()
	idx.Upsert(cand("default/a", "review"))
	idx.Upsert(cand("default/b", "review"))
	idx.Delete("default/a")

	got := idx.Candidates("review")
	if len(got) != 1 || got[0].Key != "default/b" {
		t.Fatalf("after delete: %v", got)
	}
	// Deleting the last holder of a capability empties the index entry.
	idx.Delete("default/b")
	if idx.Candidates("review") != nil {
		t.Error("capability entry should be gone after last delete")
	}
	// Deleting an unknown key is a no-op.
	idx.Delete("default/ghost")
	if idx.Len() != 0 {
		t.Errorf("Len = %d, want 0", idx.Len())
	}
}

// TestIndex_ResyncRebuild is the stateless-restart property (ADR-039 §2): a
// fresh index fed the same List yields the identical projection.
func TestIndex_ResyncRebuild(t *testing.T) {
	listed := []Candidate{
		cand("default/a", "review"),
		cand("default/b", "review", "summarize"),
		cand("team-x/c", "compile"),
	}
	first := NewIndex()
	for _, c := range listed {
		first.Upsert(c)
	}
	// "Restart": brand-new index, replay the API-server List.
	second := NewIndex()
	for _, c := range listed {
		second.Upsert(c)
	}
	if !reflect.DeepEqual(first.Snapshot(), second.Snapshot()) {
		t.Fatal("resync produced a different projection")
	}
	if !reflect.DeepEqual(first.Candidates("review"), second.Candidates("review")) {
		t.Fatal("resync produced different candidates(review)")
	}
}

func TestCandidate_CapabilityByID(t *testing.T) {
	c := cand("default/a", "review", "summarize")
	if got, ok := c.CapabilityByID("summarize"); !ok || got.ID != "summarize" {
		t.Errorf("CapabilityByID(summarize) = %v %v", got, ok)
	}
	if _, ok := c.CapabilityByID("missing"); ok {
		t.Error("CapabilityByID(missing) should be false")
	}
}

// TestIndex_ConcurrentAccess exercises writer/reader interleaving under -race.
func TestIndex_ConcurrentAccess(t *testing.T) {
	idx := NewIndex()
	var wg sync.WaitGroup
	for w := range 4 {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for n := range 50 {
				key := fmt.Sprintf("default/agent-%d-%d", w, n)
				idx.Upsert(cand(key, "review"))
				_ = idx.Candidates("review")
				idx.Delete(key)
			}
		}(w)
	}
	wg.Wait()
	if idx.Len() != 0 {
		t.Errorf("Len = %d, want 0 after balanced upsert/delete", idx.Len())
	}
}
