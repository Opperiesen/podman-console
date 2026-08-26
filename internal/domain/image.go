package domain

import (
	"strings"
	"time"
)

// ImageSummary is one image row returned by the active Podman host.
// References and Digests intentionally remain slices: an image can have
// several tags, several repository digests, or none of either.
type ImageSummary struct {
	ID         string
	References []string
	Digests    []string
	Digest     string
	Size       uint64
	CreatedAt  time.Time
	Containers int
	Dangling   bool
	ReadOnly   bool
}

// ImageDetails is the authoritative inspect result for one image.
type ImageDetails struct {
	ImageSummary
	ParentID     string
	Architecture string
	OS           string
	Labels       map[string]string
}

// PrimaryReference returns the most useful stable display name for an image.
// The empty result is meaningful for dangling or otherwise untagged images.
func (i ImageSummary) PrimaryReference() string {
	for _, reference := range i.References {
		if reference = strings.TrimSpace(reference); reference != "" {
			return reference
		}
	}
	return ""
}

// DisplayDigest returns one digest suitable for a compact inventory row.
func (i ImageSummary) DisplayDigest() string {
	for _, digest := range i.Digests {
		if digest = strings.TrimSpace(digest); digest != "" {
			return digest
		}
	}
	return strings.TrimSpace(i.Digest)
}

// ImagePullEventKind identifies an ordered observation from a pull stream.
type ImagePullEventKind string

const (
	ImagePullProgress  ImagePullEventKind = "progress"
	ImagePullSuccess   ImagePullEventKind = "success"
	ImagePullError     ImagePullEventKind = "error"
	ImagePullCancelled ImagePullEventKind = "cancelled"
	ImagePullDone      ImagePullEventKind = "done"
)

// ImagePullEvent is delivered in arrival order for one pull operation.
type ImagePullEvent struct {
	Target    string
	Reference string
	Kind      ImagePullEventKind
	Text      string
	ImageIDs  []string
}

// ImageOperationStatus describes the state of a pull or removal in the TUI.
type ImageOperationStatus string

const (
	ImageOperationIdle       ImageOperationStatus = "idle"
	ImageOperationConfirming ImageOperationStatus = "confirming"
	ImageOperationRunning    ImageOperationStatus = "running"
	ImageOperationSucceeded  ImageOperationStatus = "succeeded"
	ImageOperationFailed     ImageOperationStatus = "failed"
	ImageOperationCancelled  ImageOperationStatus = "cancelled"
	ImageOperationRefreshing ImageOperationStatus = "refreshing"
)
