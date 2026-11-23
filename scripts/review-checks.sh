#!/bin/sh
# Deterministic checks for the reviewer subagent and for human pre-commit use.
# Read-only: it never edits, formats, or commits anything.
set -u

cd "$(dirname "$0")/.." || exit 1

failed=0
section() { printf '\n[review] %s\n' "$1"; }
note() { printf '[review]   %s\n' "$1"; }
show() { printf '%s\n' "$1" | tail -30 | sed 's/^/    /'; }

staged_mode=${GLOSKI_CHECK_STAGED:-0}

changed() {
	if [ "$staged_mode" = "1" ]; then
		# The hook's temporary index contains exactly the committed paths.
		git diff --cached --name-only --diff-filter=ACM | sort -u
	else
		{
			git diff --name-only --diff-filter=ACM
			git diff --cached --name-only --diff-filter=ACM
			git ls-files --others --exclude-standard
		} | sort -u
	fi
}

diff_patch() {
	if [ "$staged_mode" = "1" ]; then
		git diff --cached -U0 "$@"
	else
		git diff -U0 "$@"
	fi
}

name_only() {
	if [ "$staged_mode" = "1" ]; then
		git diff --cached --name-only "$@"
	else
		git diff --name-only "$@"
	fi
}

section "changed files"
changed | sed 's/^/    /'
note "$(changed | wc -l | tr -d ' ') file(s)"

# Under `git commit --only` the pre-commit hook sees a temporary index holding
# only the paths being committed. Checking the worktree there would fail on
# another agent's unrelated staged/unstaged files in the same clone.
if [ "${GLOSKI_CHECK_STAGED:-}" = "1" ]; then
	whitespace_cmd="git diff --cached --check"
	whitespace_label="git diff --cached --check (commit scope)"
else
	whitespace_cmd="git diff --check"
	whitespace_label="git diff --check"
fi
section "whitespace ($whitespace_label)"
if output=$($whitespace_cmd 2>&1) && [ -z "$output" ]; then
	note "OK"
else
	note "FAILED"
	show "$output"
	failed=$((failed + 1))
fi

section "gofmt on changed Go files"
go_files=$(changed | grep '\.go$' || true)
if [ -z "$go_files" ]; then
	note "no Go files changed"
else
	unformatted=$(printf '%s\n' "$go_files" | xargs gofmt -l 2>&1)
	if [ -z "$unformatted" ]; then
		note "OK"
	else
		note "NEEDS GOFMT"
		show "$unformatted"
		failed=$((failed + 1))
	fi
fi

section "go vet ./..."
if output=$(cd server && go vet ./... 2>&1); then
	note "OK"
else
	note "FAILED"
	show "$output"
	failed=$((failed + 1))
fi

section "go test ./... (two SSH tests fail here without an SSH server)"
if output=$(cd server && go test ./... 2>&1); then
	note "OK"
else
	unexpected=$(printf '%s\n' "$output" | grep -- '--- FAIL:' | grep -v -E 'TestInstallOverSSH|TestChangedHostKeyIsReportedAndCanBeTrusted' || true)
	if [ -z "$unexpected" ]; then
		note "KNOWN ENVIRONMENT FAILURES ONLY"
		show "$(printf '%s\n' "$output" | grep -E -- '--- FAIL:|^FAIL' | head -10)"
	else
		note "FAILED"
		show "$output"
		failed=$((failed + 1))
	fi
fi

section "frontend"
if output=$(cd web && bun run type-check 2>&1); then
	note "type-check OK"
else
	note "type-check FAILED"
	show "$output"
	failed=$((failed + 1))
fi
if output=$(cd web && bun test 2>&1); then
	note "tests OK"
else
	note "tests FAILED"
	show "$output"
	failed=$((failed + 1))
fi
if output=$(cd web && bun run build 2>&1); then
	note "build OK"
else
	note "build FAILED"
	show "$output"
	failed=$((failed + 1))
fi

section "shadcn conventions"
# A real test reports every violation at once. bun run build also runs it via
# prebuild, so `make build-web` enforces this without the git hook.
if output=$(cd web && bun test tests/ui-conventions.test.ts 2>&1); then
	note "OK (also enforced by bun run build via prebuild)"
else
	note "FAILED"
	show "$output"
	failed=$((failed + 1))
fi

section "reminders from the diff"
found=0
while IFS= read -r line; do
	case "$line" in
		+READ_AT*|+WRITE_AT*|*pkill\ -f*)
			note "$line"
			found=1
			;;
	esac
done <<EOF
$(diff_patch | grep '^+' | grep -E 'READ_AT|WRITE_AT|pkill -f' || true)
EOF
[ "$found" -eq 0 ] && note "no banned patterns added"

if name_only | grep -q '^server/proto/agentv1/'; then
	note "PROTOCOL FILE CHANGED: confirm generated code was regenerated and ProtocolVersion bumped if the wire format changed"
fi
if name_only | grep -q '^server/internal/controltower/routes.go$'; then
	note "ROUTES CHANGED: confirm docs/http-api.md and the frontend client were updated"
fi

section "summary"
if [ "$failed" -eq 0 ]; then
	note "all mechanical checks passed; reviewer still checks the hard rules"
else
	note "$failed check(s) failed"
fi
exit "$failed"
