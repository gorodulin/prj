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
TAP_NAME="gorodulin/tap"
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
command -v brew >/dev/null 2>&1 || error "brew is not installed"
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
info "Packaging files exist"

echo ""

# ── Dry run stops here ───────────────────────────────────────────

if $DRY_RUN; then
    echo "Dry run — would perform:"
    echo "  1. git tag $TAG && git push origin $TAG"
    echo "  2. gh release create $TAG"
    echo "  2b. make cross && gh release upload $TAG dist/prj-*"
    echo "  3. Download $TARBALL_URL"
    echo "  4. Compute checksums and patch $FORMULA and $PORTFILE"
    echo "  5. Local brew install, test, uninstall"
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

# ── Step 13: Verify patches ─────────────────────────────────────

if grep -q "PLACEHOLDER" "$FORMULA" "$PORTFILE"; then
    error "PLACEHOLDER still present in packaging files"
fi
info "No PLACEHOLDERs remain"

echo ""

# ── Step 14: Brew test ───────────────────────────────────────────

echo "Testing Homebrew formula..."

TAP_DIR="$(brew --repository)/Library/Taps/${TAP_NAME/\///homebrew-}"
if [[ ! -d "$TAP_DIR" ]]; then
    error "Local tap not found. Run: brew tap-new $TAP_NAME"
fi

cp "$FORMULA" "$TAP_DIR/Formula/prj.rb"

trap 'brew uninstall prj 2>/dev/null || true' EXIT

brew install --build-from-source "$TAP_NAME/prj"
info "Installed"

INSTALLED_VERSION="$(prj --version 2>&1 | awk '{print $NF}')"
if [[ "$INSTALLED_VERSION" != *"$VERSION"* ]]; then
    error "Version mismatch: expected $VERSION, got $INSTALLED_VERSION"
fi
info "Version matches: $INSTALLED_VERSION"

brew test "$TAP_NAME/prj"
info "brew test passed"

trap - EXIT
brew uninstall prj
info "Uninstalled"

echo ""

# ── Step 15: Commit packaging updates and push ──────────────────

git add "$FORMULA" "$PORTFILE"
git commit -m "Update packaging for v$VERSION release"
info "Committed packaging updates"

git push origin "$MAIN_BRANCH"
info "Pushed $MAIN_BRANCH"

echo ""

# ── Step 16: Summary ─────────────────────────────────────────────

echo "Done! Release $TAG is ready and pushed."
echo ""
echo "Remaining manual steps:"
echo ""
echo "  1. Push Homebrew formula:"
echo "     cd $TAP_DIR"
echo "     git add Formula/prj.rb"
echo "     git commit -m \"Update prj to $VERSION\""
echo "     git push"
echo ""
echo "  2. Submit MacPorts Portfile:"
echo "     cp $PORTFILE <macports-ports-fork>/devel/prj/Portfile"
echo "     # Open PR to macports/macports-ports"
