#!/bin/sh
# Add shadcn primitives without losing our local conventions.
#
# The CLI has three habits that break this app:
#   1. it imports `cn` from an npm package instead of @/shared/lib/utils,
#   2. it rewrites button.tsx and input.tsx, dropping our local tweaks,
#   3. generated files that export a cva variant fail the fast-refresh lint rule.
#
# Use this instead of calling `bunx shadcn@latest add` by hand. If a generated
# file exports a variant, move it into its own file (see toggle-variants.ts).
set -eu

cd "$(dirname "$0")/.."

if [ "$#" -eq 0 ]; then
	echo "usage: $0 <component> [component...]" >&2
	exit 2
fi

cd web

bunx shadcn@latest add "$@" --yes --overwrite

# 1. Point every generated file at our own cn helper.
perl -pi -e "s/import \{ cn \} from [\"']cn[\"']/import { cn } from '\@\/shared\/lib\/utils'/" src/ui/*.tsx

# The CLI keeps re-adding the package; remove it if present.
if grep -q '"cn":' package.json; then
	bun remove cn >/dev/null 2>&1 || true
fi

# 2. Restore the local Button default-variant shadow after an overwrite.
perl -pi -e 's/bg-primary text-primary-foreground hover:bg-primary\/90/bg-primary text-primary-foreground shadow-sm hover:bg-primary\/90 hover:shadow-md/' src/ui/button.tsx

# 3. Catch missing imports, overwritten customisations and variant exports.
bun run type-check
bun run lint
sh ../scripts/review-checks.sh

echo "Added: $*"
