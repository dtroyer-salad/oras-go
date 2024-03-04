# Salad Extension to ORAS Go Library

## TL;DR

The cost of failed downloads being restarted by deleting partially downloaded layers
is a high one to pay in the Salad network especially when some of these layers may exceed
10GB.  Resuming partial downloads is an important part of a robust and resilient and
performant distributed compute node with limited bandwidth.

This repository is a fork of https://github.com/oras-project/oras-go/ just after the v2.5.0
tag.  The branch `resume` contains Salad's download changes.  The only changes required to
build the ORAS CLI (`oras`) (https://github.com/oras-project/oras) are to use this replacement
for `oras-go`.

## Summary

### Changes

* `Annotations` key constants (`internal/spec/artifact.go`)
  * AnnotationResume* - the keys used in the Annotations[] map
    * The Annotations field of the Descriptor is used to pass state around during the request handling.  This avoids changing the public API via interfaces or structs.
    * Salad-specific keys are defined in `internal/spec/artifact.go` using constants with names beginning with `AnnotationResume`.

* `remote.FetcherHead` (`registry/remote/repository.go`)
  * interface defining `FetchHead()`

* `remote.BlobStoreHead` (`registry/remote/repository.go`)
  * interface combining `registry.BlobStore` with `FetcherHead`

* `remote.Repository.FetchHead()` (new) (`registry/remote/repository.go`)
  * call `FetchHead()` when `BlobStoreHead` is implemented

* `remote.blobStore` (`registry/remote/repository.go`)
  * `blobStore.Fetch()`
    * call `FetchHead()` to check for the `Range` header support from the server
      * FALSE:
        * reset resume flag and proceed as usual
      * TRUE:
        * Set `Range` header
    * after GET request to remote repository if in resume
      * `StatusPartialContent`:
        * check response `ContentLength` against `target.Size - ingestSize`
      * `StatusOK`:
        * check response `ContentLength` against `target.Size`
  * `blobStore.FetchHead()` (new)
    * do HEAD call to src
      * `StatusOK`:
        * check response `ContentLength` against `target.Size`
        * check response header `Accept-Ranges` has value `bytes`
          * TRUE:
            * Set resume flag

* `content.Storage.Push()` (`content/oci/storage.go`)
  * call `Storage.ingest()` as usual

* `content.Storage.ingest()` (`content/oci/storage.go`)
  * if resume conditions are all met
    * TRUE:
      * open existing ingest file
      * seek to 0 in ingest file
      * create a new Hash to contain the current hash of the ingest file
      * save encoded Hash to `Annotations[hash]`
    * FALSE:
      * if not found: `CreateTemp()` a new ingest file as usual
  * if `0 <= ingest size < content-length`
    * TRUE:
      * call `ioutil.CopyBuffer()` as usual

* `content.NewVerifyReader()` (`content/reader.go`)
  * Add `resume` field to `VerifyReader` struct
  * if `Annotations[offset]` > 0
    * TRUE:
      * decode `Annotations[Hash]`
      * create a new `content.hashVerifier` with the new `Hash` and the original `desc.Digest`
    * FALSE:
      * create a new `digest.hashVerifier` from `desc.Digest`

* `content.hashVerifier` (new) (`content/verifiers.go`)
  * `digest.hashVerifier` is copied here from `opencontainers/go-digest/blob/master/verifiers.go`
    because it is private and we need to construct a verifier with our new `Hash` and the original `Digest`.

* `content.Resumer` (new) (`content.storage.go`)
  * Interface to get ingest filenames, also used to determine support for resumable downloads

* `content.Store.IngestFile()` (new) (`content/oci/storage.go`)
  * Provide access to `content.Store.storage.IngestFile()`

* `content.Storage.IngestFile()` (new) (`content/oci/storage.go`)
  * Locate and return the first matching ingest file, if any
