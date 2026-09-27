#!/usr/bin/env bash
#
# Release script for prj.
# Usage: scripts/release.sh <VERSION> [--dry-run]
#
# VERSION must be in X.Y.Z format (without the v prefix).
# --dry-run runs all guards but does not create tags, releases, or modify files.

set -euo pipefail

# ── Constants ─────────────────────────────────────────────────────

FORMULA="packaging/homebrew/prj.rb"
PORTFILE="packaging/macports/Portfile"
WINGET_DIR="packaging/winget"
WINGET_VERSION="gorodulin.prj.yaml"
WINGET_INSTALLER="gorodulin.prj.installer.yaml"
WINGET_LOCALE="gorodulin.prj.locale.en-US.yaml"
WINGET_FORK="gorodulin/winget-pkgs"
WINGET_UPSTREAM="microsoft/winget-pkgs"
TAP_REPO="gorodulin/homebrew-tap"
TAP_FORMULA="Formula/prj.rb"
MAIN_BRANCH="main"

# ── Arguments ─────────────────────────────────────────────────────

VERSION="${1:-}"
DRY_RUN=false
if [[ "${2:-}" == "--dry-run" ]]; then
    DRY_RUN=true
fi

if [[ -z "$VERSION" ]]; then
    echo "Usage: scripts/release.sh <VERSION> [--dry-run]"
    echo "  VERSION: X.Y.Z (e.g. 0.2.0)"
    exit 1
fi

if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: VERSION must be in X.Y.Z format (got: $VERSION)"
    exit 1
fi

TAG="v$VERSION"

# Reject if not newer than the current version.
CURRENT_TAG="$(git describe --tags --abbrev=0 2>/dev/null || echo "")"
if [[ -n "$CURRENT_TAG" && "$CURRENT_TAG" != "$TAG" ]]; then
    CURRENT="${CURRENT_TAG#v}"
    if printf '%s\n%s\n' "$VERSION" "$CURRENT" | sort -V | head -1 | grep -qx "$VERSION"; then
        error "Version $VERSION is not newer than current ($CURRENT)"
    fi
fi
REPO_URL="$(git remote get-url origin | sed -e 's/\.git$//' -e 's|git@github.com:|https://github.com/|')"
REPO_PATH="${REPO_URL#https://github.com/}"
TARBALL_URL="$REPO_URL/archive/refs/tags/$TAG.tar.gz"

info()  { printf "  ✓ %s\n" "$1"; }
warn()  { printf "  → %s\n" "$1"; }
error() { printf "  ✗ %s\n" "$1" >&2; exit 1; }

# ── Step 1–5: Guards (no state changes) ──────────────────────────

echo "Checking prerequisites..."

# Required tools
command -v gh >/dev/null 2>&1 || error "gh (GitHub CLI) is not installed"
command -v go >/dev/null 2>&1 || error "go is not installed"
info "Required tools available"

# Clean working tree
if [[ -n "$(git status --porcelain)" ]]; then
    error "Working tree is not clean. Commit or stash changes first."
fi
info "Clean working tree"

# On main branch
BRANCH="$(git branch --show-current)"
if [[ "$BRANCH" != "$MAIN_BRANCH" ]]; then
    error "Not on $MAIN_BRANCH branch (on: $BRANCH)"
fi
info "On $MAIN_BRANCH branch"

# Tests pass
if ! make check; then
    error "Tests or linting failed."
fi
info "Tests pass"

# CHANGELOG mentions this version
if ! grep -q "\[$VERSION\]" CHANGELOG.md 2>/dev/null; then
    error "CHANGELOG.md does not mention [$VERSION]. Update it first."
fi
info "CHANGELOG mentions $VERSION"

# Packaging files exist
[[ -f "$FORMULA" ]] || error "Formula not found: $FORMULA"
[[ -f "$PORTFILE" ]] || error "Portfile not found: $PORTFILE"
[[ -f "$WINGET_DIR/$WINGET_VERSION" ]] || error "WinGet manifest not found: $WINGET_DIR/$WINGET_VERSION"
[[ -f "$WINGET_DIR/$WINGET_INSTALLER" ]] || error "WinGet manifest not found: $WINGET_DIR/$WINGET_INSTALLER"
[[ -f "$WINGET_DIR/$WINGET_LOCALE" ]] || error "WinGet manifest not found: $WINGET_DIR/$WINGET_LOCALE"
info "Packaging files exist"

# WinGet fork exists
gh api "repos/$WINGET_FORK" --silent 2>/dev/null || error "WinGet fork not found: $WINGET_FORK"
info "WinGet fork accessible"

# The tap's formula matches ours apart from url/sha256, so the release
# changes only those two lines.
TAP_CURRENT="$(gh api -H "Accept: application/vnd.github.raw" "repos/$TAP_REPO/contents/$TAP_FORMULA" 2>/dev/null)" \
    || error "Formula not found in tap: $TAP_REPO/$TAP_FORMULA"
strip_release_lines() { grep -vE '^  (url|sha256) "'; }
if [[ "$(strip_release_lines <<<"$TAP_CURRENT")" != "$(strip_release_lines < "$FORMULA")" ]]; then
    error "$TAP_REPO/$TAP_FORMULA differs from $FORMULA beyond url/sha256. Sync them first."
fi
info "Tap formula matches $FORMULA"

echo ""

# ── Dry run stops here ───────────────────────────────────────────

if $DRY_RUN; then
    echo "Dry run — would perform:"
    echo "  1. git tag $TAG && git push origin $TAG"
    echo "  2. gh release create $TAG"
    echo "  2b. make cross && gh release upload $TAG dist/prj-*"
    echo "  3. Download $TARBALL_URL"
    echo "  4. Compute checksums and patch $FORMULA and $PORTFILE"
    echo "  5. Build the tarball as the formula does (go build), check --version and completions"
    echo "  6. Commit packaging, push $MAIN_BRANCH, commit formula to $TAP_REPO"
    echo "  7. Patch WinGet manifests and submit PR to $WINGET_UPSTREAM"
    echo ""
    echo "All guards passed. Run without --dry-run to proceed."
    exit 0
fi

# ── Step 6: Create git tag ───────────────────────────────────────

echo "Creating release..."

if git rev-parse "$TAG" >/dev/null 2>&1; then
    TAG_COMMIT="$(git rev-parse "$TAG")"
    HEAD_COMMIT="$(git rev-parse HEAD)"
    if [[ "$TAG_COMMIT" != "$HEAD_COMMIT" ]]; then
        error "Tag $TAG already exists but points at a different commit.
       Tag:  ${TAG_COMMIT:0:12}
       HEAD: ${HEAD_COMMIT:0:12}
    Use a new version number, or delete the stale tag:
       git tag -d $TAG
       git push origin :refs/tags/$TAG"
    fi
    warn "Tag $TAG already exists at HEAD — skipping"
else
    git tag "$TAG"
    info "Created tag $TAG"
fi

# ── Step 7: Push tag ────────────────────────────────────────────

if git ls-remote --tags origin | grep -q "refs/tags/$TAG$"; then
    warn "Tag $TAG already on remote — skipping push"
else
    git push origin "$TAG"
    info "Pushed tag $TAG"
fi

# ── Step 8: Create GitHub release ────────────────────────────────

if gh release view "$TAG" >/dev/null 2>&1; then
    warn "GitHub release $TAG already exists — skipping"
else
    gh release create "$TAG" --title "$TAG" --generate-notes
    info "Created GitHub release $TAG"
fi

# ── Step 8b: Upload cross-compiled binaries ─────────────────────

make cross
info "Cross-compiled binaries"

gh release upload "$TAG" dist/prj-* --clobber
info "Uploaded binaries to release $TAG"

echo ""

# ── Step 9: Download tarball ─────────────────────────────────────

echo "Updating packaging..."

TARBALL="/tmp/prj-${VERSION}.tar.gz"
HTTP_CODE="$(curl -sL -w '%{http_code}' -o "$TARBALL" "$TARBALL_URL")"
if [[ "$HTTP_CODE" != "200" ]]; then
    error "Failed to download tarball (HTTP $HTTP_CODE): $TARBALL_URL"
fi
info "Downloaded tarball"

# ── Step 10: Compute checksums ───────────────────────────────────

SHA256="$(shasum -a 256 "$TARBALL" | cut -d' ' -f1)"
RMD160="$(openssl rmd160 "$TARBALL" 2>/dev/null | awk '{print $NF}')"
SIZE="$(wc -c < "$TARBALL" | tr -d ' ')"

info "SHA256:  $SHA256"
info "RMD160:  $RMD160"
info "Size:    $SIZE"

# ── Step 11: Patch Homebrew formula ──────────────────────────────

FORMULA_TMP="${FORMULA}.tmp"
sed \
    -e "s|url \".*\"|url \"$TARBALL_URL\"|" \
    -e "s|sha256 \".*\"|sha256 \"$SHA256\"|" \
    "$FORMULA" > "$FORMULA_TMP"
mv "$FORMULA_TMP" "$FORMULA"
info "Patched $FORMULA"

# ── Step 12: Patch MacPorts Portfile ─────────────────────────────

PORTFILE_TMP="${PORTFILE}.tmp"
sed \
    -e "s|go.setup.*|go.setup            $REPO_PATH $VERSION v|" \
    -e "s|rmd160  .*|rmd160  $RMD160 \\\\|" \
    -e "s|sha256  .*|sha256  $SHA256 \\\\|" \
    -e "s|size    .*|size    $SIZE|" \
    "$PORTFILE" > "$PORTFILE_TMP"
mv "$PORTFILE_TMP" "$PORTFILE"
info "Patched $PORTFILE"

# ── Step 12b: Patch WinGet manifests ────────────────────────────

WIN_AMD64="/tmp/prj-${VERSION}-windows-amd64.exe"
WIN_ARM64="/tmp/prj-${VERSION}-windows-arm64.exe"

curl -sSfL -o "$WIN_AMD64" "$REPO_URL/releases/download/$TAG/prj-windows-amd64.exe" \
    || error "Failed to download Windows amd64 binary"
curl -sSfL -o "$WIN_ARM64" "$REPO_URL/releases/download/$TAG/prj-windows-arm64.exe" \
    || error "Failed to download Windows arm64 binary"
info "Downloaded Windows binaries"

WIN_AMD64_SHA="$(shasum -a 256 "$WIN_AMD64" | cut -d' ' -f1 | tr '[:lower:]' '[:upper:]')"
WIN_ARM64_SHA="$(shasum -a 256 "$WIN_ARM64" | cut -d' ' -f1 | tr '[:lower:]' '[:upper:]')"

info "Windows amd64 SHA256: $WIN_AMD64_SHA"
info "Windows arm64 SHA256: $WIN_ARM64_SHA"

for f in "$WINGET_VERSION" "$WINGET_INSTALLER" "$WINGET_LOCALE"; do
    sed -e "s|^PackageVersion:.*|PackageVersion: $VERSION|" \
        "$WINGET_DIR/$f" > "$WINGET_DIR/$f.tmp"
    mv "$WINGET_DIR/$f.tmp" "$WINGET_DIR/$f"
done

sed -i.bak \
    -e "s|InstallerSha256: [A-F0-9]\{64\}|PLACEHOLDER_SHA|" \
    -e "s|InstallerUrl: .*/prj-windows-amd64\.exe|InstallerUrl: $REPO_URL/releases/download/$TAG/prj-windows-amd64.exe|" \
    -e "s|InstallerUrl: .*/prj-windows-arm64\.exe|InstallerUrl: $REPO_URL/releases/download/$TAG/prj-windows-arm64.exe|" \
    "$WINGET_DIR/$WINGET_INSTALLER"
rm -f "$WINGET_DIR/$WINGET_INSTALLER.bak"

# Replace SHA placeholders in order: first occurrence is amd64, second is arm64
awk -v amd64="$WIN_AMD64_SHA" -v arm64="$WIN_ARM64_SHA" '
    /PLACEHOLDER_SHA/ && !done_amd64 { sub(/PLACEHOLDER_SHA/, "InstallerSha256: " amd64); done_amd64=1 }
    /PLACEHOLDER_SHA/ && done_amd64  { sub(/PLACEHOLDER_SHA/, "InstallerSha256: " arm64) }
    { print }
' "$WINGET_DIR/$WINGET_INSTALLER" > "$WINGET_DIR/$WINGET_INSTALLER.tmp"
mv "$WINGET_DIR/$WINGET_INSTALLER.tmp" "$WINGET_DIR/$WINGET_INSTALLER"

info "Patched WinGet manifests"

# ── Step 13: Verify patches ─────────────────────────────────────

if grep -q "PLACEHOLDER" "$FORMULA" "$PORTFILE" "$WINGET_DIR"/*; then
    error "PLACEHOLDER still present in packaging files"
fi
info "No PLACEHOLDERs remain"

echo ""

# ── Step 14: Verify formula without Homebrew ────────────────────

echo "Verifying Homebrew formula..."

# Only url and sha256 may change; everything else was proven by earlier releases.
FORMULA_CHANGES="$(git diff --numstat -- "$FORMULA" | awk '{print $1" "$2}')"
if [[ "$FORMULA_CHANGES" != "2 2" ]]; then
    error "Expected $FORMULA to change exactly url and sha256 (got +/-: ${FORMULA_CHANGES:-none})"
fi
info "Formula changes only url and sha256"

# Build the release tarball the way the formula's install block does.
BUILD_DIR="$(mktemp -d)"
trap 'rm -rf "$BUILD_DIR"' EXIT
tar -xzf "$TARBALL" -C "$BUILD_DIR" --strip-components 1
(cd "$BUILD_DIR" && go build -trimpath \
    -ldflags "-s -w -X github.com/gorodulin/prj/cmd.version=$VERSION" -o bin/prj .)
info "Built from tarball"

BUILT_VERSION="$("$BUILD_DIR/bin/prj" --version 2>&1)"
if [[ "$BUILT_VERSION" != *"$VERSION"* ]]; then
    error "Version mismatch: expected $VERSION, got $BUILT_VERSION"
fi
info "Version matches: $BUILT_VERSION"

for shell in bash zsh fish; do
    "$BUILD_DIR/bin/prj" completion "$shell" >/dev/null || error "prj completion $shell failed"
done
info "Completions generate"

rm -rf "$BUILD_DIR"
trap - EXIT

echo ""

# ── Step 15: Commit packaging updates and push ──────────────────

git add "$FORMULA" "$PORTFILE" "$WINGET_DIR"
git commit -m "Update packaging for v$VERSION release"
info "Committed packaging updates"

git push origin "$MAIN_BRANCH"
info "Pushed $MAIN_BRANCH"

# ── Step 15b: Publish formula to the tap ────────────────────────

TAP_SHA="$(gh api "repos/$TAP_REPO/contents/$TAP_FORMULA" --jq '.sha')"
if [[ "$(gh api -H "Accept: application/vnd.github.raw" "repos/$TAP_REPO/contents/$TAP_FORMULA")" == "$(cat "$FORMULA")" ]]; then
    warn "Tap formula already at $VERSION — skipping"
else
    gh api "repos/$TAP_REPO/contents/$TAP_FORMULA" -X PUT \
        -f message="Update prj to $VERSION" \
        -f content="$(base64 < "$FORMULA")" \
        -f sha="$TAP_SHA" \
        -f branch="main" --silent \
        || error "Failed to publish formula to $TAP_REPO"
    info "Published formula to $TAP_REPO"
fi

echo ""

# ── Step 16: Submit WinGet PR ───────────────────────────────────

echo "Submitting WinGet manifest..."

gh api -X POST "repos/$WINGET_FORK/merge-upstream" -f branch=master --silent 2>/dev/null
info "Synced $WINGET_FORK with upstream"

MASTER_SHA="$(gh api "repos/$WINGET_FORK/git/refs/heads/master" --jq '.object.sha')"
WINGET_BRANCH="prj-$VERSION"

gh api "repos/$WINGET_FORK/git/refs" -X POST \
    -f ref="refs/heads/$WINGET_BRANCH" -f sha="$MASTER_SHA" --silent \
    || error "Failed to create branch $WINGET_BRANCH on $WINGET_FORK"
info "Created branch $WINGET_BRANCH"

WINGET_MANIFEST_DIR="manifests/g/gorodulin/prj/$VERSION"
for f in "$WINGET_VERSION" "$WINGET_INSTALLER" "$WINGET_LOCALE"; do
    CONTENT="$(base64 < "$WINGET_DIR/$f")"
    gh api "repos/$WINGET_FORK/contents/$WINGET_MANIFEST_DIR/$f" -X PUT \
        -f message="Add gorodulin.prj v$VERSION" \
        -f content="$CONTENT" \
        -f branch="$WINGET_BRANCH" --silent \
        || error "Failed to upload $f"
done
info "Uploaded manifests to $WINGET_FORK"

WINGET_PR_URL="$(gh api "repos/$WINGET_UPSTREAM/pulls" -X POST \
    -f title="New version: gorodulin.prj v$VERSION" \
    -f head="gorodulin:$WINGET_BRANCH" \
    -f base="master" \
    -f body="Automated submission from release script." \
    --jq '.html_url')" \
    || error "Failed to create WinGet PR"
info "WinGet PR: $WINGET_PR_URL"

echo ""

# ── Step 17: Summary ─────────────────────────────────────────────

echo "Done! Release $TAG is ready and pushed."
echo ""
echo "Remaining manual steps:"
echo ""
echo "  1. Check the tap's CI run: https://github.com/$TAP_REPO/actions"
echo ""
echo "  2. Submit MacPorts Portfile:"
echo "     cp $PORTFILE <macports-ports-fork>/devel/prj/Portfile"
echo "     # Open PR to macports/macports-ports"
echo ""
echo "  3. Monitor WinGet PR: $WINGET_PR_URL"
