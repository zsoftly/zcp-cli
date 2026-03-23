# Release Process

Tag-based releases. Push version tag → automated builds → GitHub Release.

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

## Version Format

`<major>.<minor>.<patch>` (semver, no `v` prefix)

Examples: `0.0.1`, `0.1.0`, `1.0.0`

## Build Artifacts

| Platform      | Binary                    |
| ------------- | ------------------------- |
| Linux amd64   | `zcp-linux-amd64`         |
| Linux arm64   | `zcp-linux-arm64`         |
| macOS amd64   | `zcp-darwin-amd64`        |
| macOS arm64   | `zcp-darwin-arm64`        |
| Windows amd64 | `zcp-windows-amd64.exe`   |
| Windows arm64 | `zcp-windows-arm64.exe`   |

Also included in the release:

- `install.sh` — Unix one-liner installer
- `install.ps1` — Windows one-liner installer
- Archives (`.tar.gz` for Unix, `.zip` for Windows)
- `checksums.txt` — SHA256 checksums for all artifacts

## Troubleshooting

```bash
# Check workflow status
gh run list
gh run view <run-id> --log

# Local test build
go build ./cmd/zcp && ./zcp version
```
