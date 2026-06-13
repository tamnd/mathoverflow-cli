---
title: "Installation"
description: "Install mo from a release, with go install, or from source."
weight: 20
---

## Prebuilt binaries

Every [release](https://github.com/tamnd/mathoverflow-cli/releases) carries archives for Linux, macOS,
and Windows on amd64 and arm64, plus deb, rpm, and apk packages for Linux.
Download, unpack, put `mo` on your `PATH`, done. The `checksums.txt`
on each release is signed with keyless [cosign](https://docs.sigstore.dev/) if
you want to verify before running.

## With Go

```bash
go install github.com/tamnd/mathoverflow-cli/cmd/mo@latest
```

That puts `mo` in `$(go env GOPATH)/bin`, which is `~/go/bin` unless
you moved it. Make sure that directory is on your `PATH`.

## From source

```bash
git clone https://github.com/tamnd/mathoverflow-cli
cd mathoverflow-cli
make build        # produces ./bin/mo
./bin/mo version
```

## Container image

```bash
docker run --rm ghcr.io/tamnd/mo:latest --help
```

## Checking the install

```bash
mo version
```

prints the version and exits.
