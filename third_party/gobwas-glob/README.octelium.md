This directory contains the source, tests, and MIT license from
[github.com/gobwas/glob v1.0.0](https://github.com/gobwas/glob/tree/v1.0.0),
commit `80c58b01f2b00664c92a1103bd9d9e328810d758`.

The upstream files are unchanged. `compat.go` restores the `Glob` interface
used by OpenTelemetry Collector while preserving the `Pattern` API required
by OPA. The collector module's replacement makes this version available to
the shared Go workspace without downgrading OPA or disabling workspace mode.

Run the upstream suite and the compatibility regression from the repository
root with `go test github.com/gobwas/glob/...`.

Remove this replacement when the collector supports the upstream `Pattern`
API or upstream glob provides the legacy interface.
