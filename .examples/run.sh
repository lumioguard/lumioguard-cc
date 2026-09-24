#!/usr/bin/env sh
# Checks each worked example before and after its refactor, comparing through a scratch Git repo.
# Usage from the repository root: sh .examples/run.sh [path-to-binary]
set -u

BINARY=${1:-bin/lumioguard-cc}
if [ ! -x "$BINARY" ] && [ -x "$BINARY.exe" ]; then
    BINARY="$BINARY.exe"
fi
if [ ! -x "$BINARY" ]; then
    echo "build the CLI first: go build -o bin/lumioguard-cc ./cmd/lumioguard-cc" >&2
    exit 2
fi
BINARY=$(cd "$(dirname "$BINARY")" && pwd)/$(basename "$BINARY")
ROOT=$(pwd)

HAVE_GIT=1
command -v git >/dev/null 2>&1 || HAVE_GIT=0

# One scratch directory for the whole run, removed on any exit including an
# interrupt, so a cancelled demo leaves nothing behind.
SCRATCH=$(mktemp -d) || exit 2
trap 'rm -rf "$SCRATCH"' EXIT HUP INT TERM

# refactored_against_git_base NAME: commit before/, swap in after/, check against that commit.
refactored_against_git_base() {
    name=$1
    work="$SCRATCH/$name"
    mkdir -p "$work" || return 2
    cp -R "$ROOT/.examples/$name/before/." "$work/"

    # Empty template and hooks path keep the host's global Git hooks out of this fixture.
    empty="$SCRATCH/empty-template"
    mkdir -p "$empty"
    git -C "$work" init -q --template="$empty"
    git="git -C $work -c core.hooksPath=$empty -c core.autocrlf=false"
    $git add -A -f
    $git -c user.email=example@example.invalid -c user.name=WorkedExample \
        commit -q -m "the deliberately bad version"

    # Replace the sources with the refactored ones, keeping the configuration.
    find "$work" -mindepth 1 -maxdepth 1 ! -name .git ! -name .lumioguard-cc.json -exec rm -rf {} +
    cp -R "$ROOT/.examples/$name/after/." "$work/"

    "$BINARY" check --base HEAD --root "$work"
    code=$?
    rm -rf "$work"
    return $code
}

status=0
for name in typescript-order-service python-inventory java-billing c-sensor-pipeline cpp-shipping-quotes; do
    printf '=== %s: before ===\n' "$name"
    "$BINARY" check --root ".examples/$name/before"
    before=$?
    printf 'exit %d (1 is expected: the bad version must fail)\n\n' "$before"
    if [ "$before" -ne 1 ]; then
        status=1
    fi

    if [ "$HAVE_GIT" -eq 0 ]; then
        printf '=== %s: after, compared with the bad version ===\nskipped: git is not installed\n\n' "$name"
        continue
    fi
    printf '=== %s: after, compared with the bad version through git ===\n' "$name"
    refactored_against_git_base "$name"
    after=$?
    printf 'exit %d (0 is expected: the refactor must pass)\n\n' "$after"
    if [ "$after" -ne 0 ]; then
        status=1
    fi
done

if [ "$status" -eq 0 ]; then
    echo "All examples behaved as documented."
else
    echo "An example did not behave as documented." >&2
fi
exit "$status"
