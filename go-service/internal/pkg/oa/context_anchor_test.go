package oa

import "testing"

func TestIsAnchorStale(t *testing.T) {
	stored := OAContextAnchor{LastReturnLogID: 100, FlowRevision: 150, ContentFingerprint: "sha256:aaa"}
	currentSame := OAContextAnchor{LastReturnLogID: 100, FlowRevision: 150, ContentFingerprint: "sha256:aaa"}
	if IsAnchorStale(stored, currentSame) {
		t.Fatal("expected same anchor not stale")
	}

	currentNewReturn := OAContextAnchor{LastReturnLogID: 101, FlowRevision: 160}
	if !IsAnchorStale(stored, currentNewReturn) {
		t.Fatal("expected new return to be stale")
	}

	currentNewRevision := OAContextAnchor{LastReturnLogID: 100, FlowRevision: 151}
	if !IsAnchorStale(stored, currentNewRevision) {
		t.Fatal("expected higher flow revision to be stale")
	}

	currentNewFingerprint := OAContextAnchor{LastReturnLogID: 100, FlowRevision: 150, ContentFingerprint: "sha256:bbb"}
	if !IsAnchorStale(stored, currentNewFingerprint) {
		t.Fatal("expected fingerprint change to be stale")
	}
}
