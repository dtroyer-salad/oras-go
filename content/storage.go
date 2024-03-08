/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package content

import (
	"context"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	// The AnnotationResume* constants define the keys used in resumable downloads
	// in the ocispec.Descriptior.Annotations map for passing resume state around.

	// AnnotationResumeDownload is "true" when a resumable is being attempted.
	AnnotationResumeDownload = "com.salad.image.resume"

	// AnnotationResumeFilename contains the full ingest filename.
	AnnotationResumeFilename = "com.salad.image.resume.filename"

	// AnnotationResumeHash contains a hash.Hash of the existing ingest file
	// suitable for using in the new Verifier to resume download verification.
	AnnotationResumeHash = "com.salad.image.resume.hash"

	// AnnotationResumeOffset contains the offset to resume downloading, aka
	// the size of the existing ingest file.
	AnnotationResumeOffset = "com.salad.image.resume.offset"
)

// Fetcher fetches content.
type Fetcher interface {
	// Fetch fetches the content identified by the descriptor.
	Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error)
}

// Pusher pushes content.
type Pusher interface {
	// Push pushes the content, matching the expected descriptor.
	// Reader is preferred to Writer so that the suitable buffer size can be
	// chosen by the underlying implementation. Furthermore, the implementation
	// can also do reflection on the Reader for more advanced I/O optimization.
	Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error
}

// Storage represents a content-addressable storage (CAS) where contents are
// accessed via Descriptors.
// The storage is designed to handle blobs of large sizes.
type Storage interface {
	ReadOnlyStorage
	Pusher
}

// ReadOnlyStorage represents a read-only Storage.
type ReadOnlyStorage interface {
	Fetcher

	// Exists returns true if the described content exists.
	Exists(ctx context.Context, target ocispec.Descriptor) (bool, error)
}

// Deleter removes content.
// Deleter is an extension of Storage.
type Deleter interface {
	// Delete removes the content identified by the descriptor.
	Delete(ctx context.Context, target ocispec.Descriptor) error
}

// FetchAll safely fetches the content described by the descriptor.
// The fetched content is verified against the size and the digest.
func FetchAll(ctx context.Context, fetcher Fetcher, desc ocispec.Descriptor) ([]byte, error) {
	rc, err := fetcher.Fetch(ctx, desc)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return ReadAll(rc, desc)
}

// FetcherFunc is the basic Fetch method defined in Fetcher.
type FetcherFunc func(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error)

// Fetch performs Fetch operation by the FetcherFunc.
func (fn FetcherFunc) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	return fn(ctx, target)
}
