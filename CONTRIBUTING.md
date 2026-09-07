# Contributing

Thank you for your interest in zcp-cli.

## Reporting Issues

Please use [GitHub Issues](https://github.com/zsoftly/zcp-cli/issues) to report bugs or request features.

When filing a bug report, include:

- The `zcp` version (`zcp version`)
- Your operating system and architecture
- The exact command you ran
- The expected vs. actual output
- Any relevant error messages (use `--debug` for additional detail)

## Pull Requests

We welcome contributions from the community. Before opening a pull request:

1. Open an issue first to discuss the change.
2. Fork the repository and create a feature branch.
3. Follow the existing code style (run `make fmt` before committing).
4. Add or update tests for any changed behavior.
5. Run `make test-race` to confirm all tests pass.
6. Sign off every commit (`git commit -s`) — see [Developer Certificate of Origin](#developer-certificate-of-origin) below.
7. Open a pull request with a clear description of the change.

## Documentation

Four files can need updating. Which ones depend on what you changed.

| File                       | When to update it                                                        | Who                  |
| -------------------------- | ------------------------------------------------------------------------ | -------------------- |
| `docs/commands.md`         | Any time you add, change, or remove a command or a flag                  | Contributor, same PR |
| `docs/command-taxonomy.md` | Any time you add or remove a command, or change which flags are required | Contributor, same PR |
| `CHANGELOG.md`             | Any time a user could notice the change                                  | Contributor, same PR |
| `RELEASE_NOTES.md`         | At release time only                                                     | Maintainer           |

`docs/commands.md` is the authoritative command reference and the documentation
site picks up from it, so an error there propagates everywhere downstream.

Add `CHANGELOG.md` entries under a `## [Unreleased]` heading. Do not choose a
version number or a date, because those are assigned when the release is cut.
Creating a versioned heading in a pull request tends to conflict with other
branches doing the same thing, and the date is usually wrong by the time the
release ships.

Leave `RELEASE_NOTES.md` alone. It is rewritten for each release and published
as the GitHub Release body.

A quick test: if a user could notice the change, it needs a `CHANGELOG.md`
entry. If they would have to type something different, it needs a
`docs/commands.md` update as well.

## Changing an exported API

The Terraform provider consumes this repository as a Go library, so changing an
exported signature under `pkg/api/**` can break it. You do not need to fix the
provider in your pull request, but please say so in the description when you
change one, so the provider update can be sequenced behind the next release.

## Developer Certificate of Origin

All commits must be signed off, certifying the [Developer Certificate of Origin](https://developercertificate.org/): a statement that you wrote the code, or otherwise have the right to submit it under this project's license.

Sign off by adding the `-s` flag when committing:

```sh
git commit -s -m "fix: describe the change"
```

This appends a `Signed-off-by: Your Name <your@email>` line to the commit message. The name and email must match the commit author.

To sign off commits already on your branch, rebase with sign-off and force-push (replace `main` with your pull request's base branch if different):

```sh
git rebase --signoff origin/main
git push --force-with-lease
```

A DCO check runs on every pull request and must pass before merge.

## Development Setup

See [docs/development.md](docs/development.md) for the full development guide.
