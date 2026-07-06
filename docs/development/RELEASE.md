# Release Process

Tag-based releases. Push version tag → automated builds → GitHub Release.

## Prepare release content (before tagging)

1. **`CHANGELOG.md`**: add the new `[vX.Y.Z] - YYYY-MM-DD` entry (Keep a
   Changelog format, matching the existing entries).
2. **`RELEASE_NOTES.md`**: rewrite for this release only. `build.yml` uploads
   it verbatim as the GitHub release body (`body_path`). Required sections:
   - A short narrative of the headline change, with highlights.
   - **Installation and upgrade (ALWAYS included).** The install one-liners
     (Linux/macOS `install.sh`, Windows `install.ps1`), the manual-download
     pointer, and a `zcp version` verify line showing the new version. Copy the
     section from the previous release notes and bump the version.
   - Fixed / Added details with copy-paste examples.

## Steps

```bash
# 1. Create release branch
git checkout main && git pull origin main
git checkout -b release/<version>
git push -u origin release/<version>

# 2. Tag (triggers build + release)
git tag <version>
git push origin <version>

# 3. Merge back
git checkout main
git merge release/<version>
git push origin main
git branch -d release/<version>
```

## Update the public docs changelog

The documentation site carries a unified, user-facing changelog at
`zcp-docs/src/content/docs/changelog/index.md` and its French mirror
`zcp-docs/src/content/docs/fr/changelog/index.md`. It is **maintained by hand** (not
generated). After cutting a release, add the new version to the **CLI (`zcp`)** section of
**both** files:

1. Add a `### vX.Y.Z - <Month DD, YYYY>` entry at the top of the CLI section, summarizing
   the **user-facing** highlights only. Skip internal struct/JSON-tag/test changes.
2. **Keep it vendor-neutral.** Public Cloud docs must not name internal backends (Ceph,
   RGW, CloudStack, etc.). Use "S3-compatible", "the platform API", and the like.
3. In `zcp-docs`, run `pnpm fmt && pnpm build` (the build validates internal links).

> **Future automation (gated).** Auto-generating the docs CLI section from this repo's
> `CHANGELOG.md` is intentionally **not** wired up. The prerequisite is making **this
> `CHANGELOG.md` vendor-neutral at the source**. Until it carries no internal backend
> names, the docs changelog must stay a curated hand-written mirror. Treat "neutral source
> changelog" as the gate before building any pull-from-GitHub generation.

## Version Format

`v<major>.<minor>.<patch>` (semver, **with** the `v` prefix: the release
workflow only triggers on tags matching `v[0-9]*`, and `install.sh`/version
injection expect it).

Examples: `v0.0.22`, `v0.1.0`, `v1.0.0`

## Build Artifacts

| Platform      | Binary                  |
| ------------- | ----------------------- |
| Linux amd64   | `zcp-linux-amd64`       |
| Linux arm64   | `zcp-linux-arm64`       |
| macOS amd64   | `zcp-darwin-amd64`      |
| macOS arm64   | `zcp-darwin-arm64`      |
| Windows amd64 | `zcp-windows-amd64.exe` |
| Windows arm64 | `zcp-windows-arm64.exe` |

Also included in the release:

- `install.sh`: Unix one-liner installer
- `install.ps1`: Windows one-liner installer
- Archives (`.tar.gz` for Unix, `.zip` for Windows)
- `checksums.txt`: SHA256 checksums for all artifacts

## Troubleshooting

```bash
# Check workflow status
gh run list
gh run view <run-id> --log

# Local test build
go build ./cmd/zcp && ./zcp version
```
