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

func TestCompareContextAnchorsClassifiesOrdinaryApproval(t *testing.T) {
	stored := OAContextAnchor{
		LastLogID:          100,
		LastReturnLogID:    20,
		LastResubmitLogID:  21,
		CurrentNodeID:      7,
		ContentFingerprint: "sha256:data",
	}
	current := stored
	current.LastLogID = 101
	current.LastLogType = "0"
	current.CurrentNodeID = 8

	changes := CompareContextAnchors(stored, current)
	if !changes.FlowChanged || !changes.CurrentNodeChanged {
		t.Fatalf("expected flow and node changes, got %+v", changes)
	}
	if changes.DataChanged || changes.AttachmentChanged || changes.ReturnResubmitChanged {
		t.Fatalf("ordinary approval must not be classified as business data change: %+v", changes)
	}
}

func TestComputeAttachmentFingerprintsTracksVersion(t *testing.T) {
	before, beforeFields := ComputeAttachmentFingerprints([]AttachmentVersionAnchor{
		{FieldKey: "FJ", DocID: "10", VersionID: 1, ImageFileID: "100", FileName: "a.pdf"},
	})
	after, afterFields := ComputeAttachmentFingerprints([]AttachmentVersionAnchor{
		{FieldKey: "fj", DocID: "10", VersionID: 2, ImageFileID: "101", FileName: "a.pdf"},
	})
	if before == "" || after == "" || before == after {
		t.Fatalf("attachment version change must update fingerprint: before=%s after=%s", before, after)
	}
	if beforeFields["fj"] == afterFields["fj"] {
		t.Fatal("field fingerprint must change with version")
	}
}
