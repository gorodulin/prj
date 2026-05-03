# Windows Distribution

How `prj` is distributed on Windows via WinGet, plus manual PowerShell fallback.

## Current status

`gorodulin.prj` is live on WinGet. Users can install with
`winget install gorodulin.prj`.

| Version | PR | Opened | Merged |
|---------|----|--------|--------|
| 0.4.1 (first submission) | [#357263](https://github.com/microsoft/winget-pkgs/pull/357263) | 2026-04-09 | 2026-04-14 |
| 0.5.0 | [#362521](https://github.com/microsoft/winget-pkgs/pull/362521) | 2026-04-19 | 2026-04-19 (~2h) |
| 0.6.0 | [#365330](https://github.com/microsoft/winget-pkgs/pull/365330) | 2026-04-26 | 2026-04-26 (~3h) |
| 0.7.0 | [#367422](https://github.com/microsoft/winget-pkgs/pull/367422) | 2026-05-01 | 2026-05-01 (~4h) |

The v0.5.0 PR was opened and merged automatically by `release.sh`,
confirming the end-to-end automation works.

**To check the latest submission status:**
`gh pr list --repo microsoft/winget-pkgs --search "gorodulin.prj" --state all`

## WinGet

### How it works

WinGet's `portable` installer type handles single-binary CLI tools:

1. Downloads the `.exe` to `%LOCALAPPDATA%\Microsoft\WinGet\Packages\`
2. Creates a symlink named `prj.exe` in `%LOCALAPPDATA%\Microsoft\WinGet\Links\`
   (already on PATH in Windows 11)
3. Registers metadata for `winget upgrade` tracking
4. `Commands: [prj]` in the manifest controls the symlink name

Users install with:

```
winget install gorodulin.prj
```

### Package ID

WinGet uses `Publisher.PackageName` format. Our ID: **`gorodulin.prj`**

### Manifest format

Three YAML files per version, stored locally in `packaging/winget/` and
submitted to `microsoft/winget-pkgs` under:

```
manifests/g/gorodulin/prj/<VERSION>/
  gorodulin.prj.yaml                 # version manifest
  gorodulin.prj.installer.yaml       # installer manifest
  gorodulin.prj.locale.en-US.yaml    # metadata manifest
```

The directory is partitioned by the first letter of the publisher name
in lowercase (`g/` for `gorodulin`).

Between releases, only version, URLs, and SHA256 hashes change.
The `release.sh` script handles this automatically by patching the
template files in `packaging/winget/` and submitting a PR via `gh` API.

### Schema headers

Every WinGet manifest file **must** start with a `yaml-language-server`
schema comment on line 1. Without it, automated validation fails with
"Schema header not found." The header tells the validator which schema
to apply:

| File | Header |
|------|--------|
| Version | `# yaml-language-server: $schema=https://aka.ms/winget-manifest.version.1.9.0.schema.json` |
| Installer | `# yaml-language-server: $schema=https://aka.ms/winget-manifest.installer.1.9.0.schema.json` |
| Locale | `# yaml-language-server: $schema=https://aka.ms/winget-manifest.defaultLocale.1.9.0.schema.json` |

The `1.9.0` in the URL matches `ManifestVersion` in the file. If you
bump `ManifestVersion`, update the schema URL to match.

#### Version manifest (`gorodulin.prj.yaml`)

```yaml
# yaml-language-server: $schema=https://aka.ms/winget-manifest.version.1.9.0.schema.json
PackageIdentifier: gorodulin.prj
PackageVersion: 0.4.1
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.9.0
```

#### Installer manifest (`gorodulin.prj.installer.yaml`)

```yaml
# yaml-language-server: $schema=https://aka.ms/winget-manifest.installer.1.9.0.schema.json
PackageIdentifier: gorodulin.prj
PackageVersion: 0.4.1
InstallerType: portable
Commands:
- prj
Installers:
- Architecture: x64
  InstallerUrl: https://github.com/gorodulin/prj/releases/download/v0.4.1/prj-windows-amd64.exe
  InstallerSha256: B7F0264A20919F792979295AE2C29665E42D137EC845F531DB941B16F9645993
- Architecture: arm64
  InstallerUrl: https://github.com/gorodulin/prj/releases/download/v0.4.1/prj-windows-arm64.exe
  InstallerSha256: 35189E3C6CEA39550813D8B6F7512FA918A3957C6B74AA6133AA6B8104163DDB
ManifestType: installer
ManifestVersion: 1.9.0
```

#### Locale manifest (`gorodulin.prj.locale.en-US.yaml`)

```yaml
# yaml-language-server: $schema=https://aka.ms/winget-manifest.defaultLocale.1.9.0.schema.json
PackageIdentifier: gorodulin.prj
PackageVersion: 0.4.1
PackageLocale: en-US
Publisher: gorodulin
PublisherUrl: https://github.com/gorodulin
PublisherSupportUrl: https://github.com/gorodulin/prj/issues
PackageName: Projector
PackageUrl: https://github.com/gorodulin/prj
License: Apache-2.0
LicenseUrl: https://github.com/gorodulin/prj/blob/main/LICENSE
ShortDescription: A cross-platform CLI tool for managing project folders, metadata, and links.
Tags:
- cli
- project-management
- productivity
ManifestType: defaultLocale
ManifestVersion: 1.9.0
```

### Submission process

Submissions go through `microsoft/winget-pkgs` as PRs:

1. Automated validation (~30 minutes):
   - YAML schema compliance
   - Installer URL reachability
   - SHA256 hash verification (downloads the binary and checks)
   - Malware scan (Microsoft Defender / SmartScreen)
2. Human review:
   - Metadata accuracy (publisher, license, description)
   - Not a duplicate of an existing package
   - Package ID follows convention
3. Merge

**Timeline:**
- First submission: 1-7 days (more scrutiny)
- Version updates: hours to 1-2 days

**Common rejection reasons:**
- Missing schema header comment on line 1 ("Schema header not found")
- SHA256 hash mismatch (if release asset was re-uploaded after manifest creation)
- URL returns 404
- Missing or inaccurate metadata

### Release automation

`release.sh` automates WinGet submission as part of the release process:

1. Computes SHA256 of the released Windows binaries (amd64 + arm64)
2. Patches `packaging/winget/` manifests with new version and hashes
3. Syncs the `gorodulin/winget-pkgs` fork with upstream
4. Creates a branch, uploads the three manifest files via GitHub API
5. Opens a PR to `microsoft/winget-pkgs`

This avoids cloning the large winget-pkgs repo. Requires `gh` CLI
with authentication (already a hard prerequisite of `release.sh`).

### SmartScreen

**Code signing is NOT required** for WinGet submission. Unsigned binaries
are accepted.

Windows SmartScreen will show a "Windows protected your PC" warning when
users first run an unsigned `.exe` with low download reputation. This
happens at first execution, not during `winget install`.

| Option | Cost | Effect |
|--------|------|--------|
| Do nothing | Free | Users click "More info" → "Run anyway". Fine for developer tools. |
| Standard (OV) code signing | ~$100/year | Builds SmartScreen reputation gradually. |
| Extended Validation (EV) | ~$400/year | Immediate SmartScreen trust. |

**Current decision:** No code signing. Target audience is developers.
Revisit if user friction becomes a real issue.

## PowerShell manual install (fallback)

For users who prefer not to use WinGet, or on systems where it is
unavailable.

```powershell
# Download
curl.exe -sSfL -o prj.exe https://github.com/gorodulin/prj/releases/latest/download/prj-windows-amd64.exe

# Move to a directory on PATH (create if needed)
New-Item -ItemType Directory -Force -Path "$env:LOCALAPPDATA\Programs\prj" | Out-Null
Move-Item -Force prj.exe "$env:LOCALAPPDATA\Programs\prj\prj.exe"
```

### Add to PATH (one-time)

```powershell
$prjDir = "$env:LOCALAPPDATA\Programs\prj"
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$prjDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$prjDir", "User")
    Write-Host "Added $prjDir to PATH. Restart your terminal."
}
```

`curl.exe` is built into Windows 10 build 17063+ and all Windows 11.
The `.exe` suffix avoids confusion with the PowerShell
`Invoke-WebRequest` alias.

The `/releases/latest/download/` URL works because the asset filename
(`prj-windows-amd64.exe`) is version-independent — GitHub redirects
`latest` to the most recent release.

**PATH conflict note:** If a user installs via PowerShell first and
later switches to `winget install`, both copies will exist. The one
earlier in PATH wins. Remove the manual copy from
`%LOCALAPPDATA%\Programs\prj\` if switching to WinGet.

## References

- WinGet manifest docs: https://learn.microsoft.com/en-us/windows/package-manager/package/manifest
- Submission guide: https://learn.microsoft.com/en-us/windows/package-manager/package/repository
- winget-pkgs repo: https://github.com/microsoft/winget-pkgs
- Our first submission: https://github.com/microsoft/winget-pkgs/pull/357263
