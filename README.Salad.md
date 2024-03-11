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
