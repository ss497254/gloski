#!/bin/sh
# Commit an explicit list of paths, leaving anyone else's staged work alone.
#
# A plain `git add && git commit` in a shared clone commits every staged file,
# including files staged by another agent. Use this instead when more than one
# agent is working in the same checkout.
set -eu

cd "$(dirname "$0")/.."

if [ "$#" -lt 2 ]; then
	echo "usage: $0 \"type: message\" <path> [path...]" >&2
	exit 2
fi

message=$1
shift

for path in "$@"; do
	if [ ! -e "$path" ]; then
		echo "commit-only: no such path: $path" >&2
		exit 2
	fi
done

# --only ignores the index for tracked files, but untracked paths must be in it.
git add -- "$@"
git commit --only -m "$message" -- "$@"
