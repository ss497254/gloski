#!/bin/sh
# Point this clone's Git hooks at the checked-in hooks directory.
set -eu

cd "$(dirname "$0")/.."
git config --local core.hooksPath scripts/hooks
echo "Enabled pre-commit review hook from scripts/hooks."
echo "Bypass for an emergency with GLOSKI_SKIP_REVIEW=1 git commit ..."
