# Salad Extension to ORAS Go Library

## TL;DR

The cost of failed downloads being restarted by deleting partially downloaded layers
is a high one to pay in the Salad network especially when some of these layers may exceed
10GB.  Resuming partial downloads is an important part of a robust and resilient and
performant distributed compute node with limited bandwidth.

This repository is a fork of https://github.com/oras-project/oras-go/ just after the v2.4.0
tag.  The branch `resume` contains Salad's download changes.  The only changes required to
build the ORAS CLI (`oras`) (https://github.com/oras-project/oras) are to use this replacement
for `oras-go`.

## Summary

### Resumable Downloads

This resumable download implementation is contained entirely within oras-go and the code path
below oras.doCopyNode().  Attempts have been made to not alter the existing external interfaces
although some new ones have been added.  Resume download is always enabled but conditions are
carefully evaluated and falls back to the original code path when not possible. This
implementation does not include any way to force resume enabled (fail if not possible) or
disabled (do not attempt even when possible).

Resumable downloads are limited to remote registry source targets and local storage destination
targets.

In order to implement this with minimally-changed interfaces a method for passing some state
between ares is needed, the Annotations field of the Descriptor was selected using Salad-named
keys defined in `internal/spec/artifact.go` using constants with names beginning with
`AnnotationResume`.

* `Annotations` key constants (`internal/spec/artifact.go`)
  * AnnotationResume* - the keys used in the Annotations[] map

* `oras.doCopyNode()` (`copy.go`)
  * Look for files in the ingest directory that match the current `Descriptor` being downloaded
    * if found: save full filename and file size to the `Annotations` map for the `Descriptor`
    * if not found: nothing to see here, proceed as normal

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
      * make a new Hash to contain the running hash of the ingest file
      * save encoded Hash to `Annotations[hash]`
    * FALSE:
      * if not found: `CreateTemp()` a new ingest file as usual
  * if `0 <= ingest size < content-length`
    * TRUE:
      * call `ioutil.CopyBuffer()` as usual

* `ioutil.CopyBuffer()` (`internal/ioutil.io.go`)
  * call `content.NewVerifyReader()` as usual
  * handle `io.ErrUnexpectedEOF`: check `bytes read == desc.Size - ingestSize`

* `content.NewVerifyReader()` (`content/reader.go`)
  * Add resume field to `VerifyReader` struct
  * if `Annotations[offset]` > 0
    * TRUE:
      * decode `Annotations[Hash]`
      * create a new `content.hashVerifier` with the new `Hash` and the original `desc.Digest`
    * FALSE:
      * create a new `digest.hashVerifier` from `desc.Digest`

* `content.hashVerifier` (new) (`content/verifiers.go`)
  * `digest.hashVerifier` is copied here from `opencontainers/go-digest/blob/master/verifiers.go`
    because it is private and we need to construct one with our new `Hash` and the original `Digest`.

* `content.Resumer` (new) (`content.storage.go`)
  * Interface to get ingest filenames, also used to determine support for resumable downloads

* `content.Store.IngestFile()` (new) (`content/oci/storage.go`)
  * Provide access to `content.Store.storage.IngestFile()`

* `content.Storage.IngestFile()` (new) (`content/oci/storage.go`)
  * Locate and return the first matching ingest file, if any
