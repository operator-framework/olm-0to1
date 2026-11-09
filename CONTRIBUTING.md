# Contributing to olm-0to1

`olm-0to1` is an Apache 2.0-licensed repository for the OLMv0-to-OLMv1
migration command line tools. Contributions through GitHub issues and pull
requests are welcome. For substantial changes to migration behavior, open an
issue or draft PR first. You can also discuss them in the
[#olm-dev Slack channel](https://kubernetes.slack.com/archives/C0181L6JYQ2) or
the [OLM working group](https://github.com/operator-framework/community#operator-lifecycle-manager-working-group).

You do not need Operator Framework organization membership to contribute. If
you want to become a member, follow the
[contributor ladder](https://github.com/operator-framework/community/blob/master/contributor-ladder.md).

## Build and test

Use the Go version declared in [`go.mod`](go.mod). The command packages are in
`cmd/`; reusable migration logic is a dependency on
`github.com/operator-framework/library-olm`.

```sh
make build                 # build both commands into bin/
make test                  # run both command packages' unit tests
go vet ./cmd/...           # check the command packages
```

Run `gofmt` on changed Go files. Include unit tests for changed command
behavior. Changes that need reusable migration logic or cluster-level tests
should also be made in
[library-olm](https://github.com/operator-framework/library-olm), where the
migration packages and E2E suites live. Update the `library-olm` dependency in
`go.mod` when taking those changes into this repository, then run `go mod tidy`
and `make test`.

## Submit a pull request

Keep PRs focused. Explain the behavior, risk, and validation commands, and link
related issues or upstream changes. Call out cluster-level tests that were not
run. Avoid adding dependencies or raising the Go version without explaining
why they are needed.

By contributing, you certify the
[Developer Certificate of Origin](https://developercertificate.org/). Sign every
commit with `git commit --signoff` and preserve existing trailers when rewriting
commits. Use a short, descriptive subject such as `fix: reject unsafe migration
target` or `test: cover catalog CLI output`.
