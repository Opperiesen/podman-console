package domain

import (
	"reflect"
	"testing"
)

func TestImageDisplayHelpersPreferNonEmptyReferencesAndDigests(t *testing.T) {
	image := ImageSummary{
		References: []string{" ", "registry.example/app:latest"},
		Digests:    []string{"", "sha256:abc"},
		Digest:     "sha256:fallback",
	}
	if got := image.PrimaryReference(); got != "registry.example/app:latest" {
		t.Fatalf("PrimaryReference() = %q", got)
	}
	if got := image.DisplayDigest(); got != "sha256:abc" {
		t.Fatalf("DisplayDigest() = %q", got)
	}
	if empty := (ImageSummary{}).PrimaryReference(); empty != "" {
		t.Fatalf("empty PrimaryReference() = %q", empty)
	}
}

func TestImagePullEventKindsAndStatusesAreStable(t *testing.T) {
	if !reflect.DeepEqual([]ImagePullEventKind{ImagePullProgress, ImagePullSuccess, ImagePullError, ImagePullCancelled, ImagePullDone}, []ImagePullEventKind{
		"progress", "success", "error", "cancelled", "done",
	}) {
		t.Fatal("image pull event vocabulary changed unexpectedly")
	}
	if ImageOperationConfirming == ImageOperationRunning || ImageOperationSucceeded == ImageOperationFailed {
		t.Fatal("image operation statuses are not distinct")
	}
}
