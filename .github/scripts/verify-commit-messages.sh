#!/usr/bin/env bash
# Verifies non-merge commit subjects in a PR range match conventional commits.
#
# Prefer BASE_SHA + HEAD_SHA (GitHub pull_request event) for accuracy on forks.
# Fallback: BASE_REF branch name vs HEAD (local / Actions checkout).
#
# shellcheck disable=SC2207

set -euo pipefail

conv='^(feat|fix|refactor|test|docs|chore|perf)(\([a-zA-Z0-9._/-]+\))?!?: .+'
conv_no_scope='^(feat|fix|refactor|test|docs|chore|perf)!?: .+'

if [[ -n "${BASE_SHA:-}" ]] && [[ -n "${HEAD_SHA:-}" ]]; then
  mapfile -t SUBJECTS < <(git log --no-merges "${BASE_SHA}..${HEAD_SHA}" --format=%s)
elif [[ -n "${BASE_REF:-}" ]]; then
  git fetch origin "${BASE_REF}" --depth=10000 2>/dev/null || git fetch origin "${BASE_REF}"
  mapfile -t SUBJECTS < <(git log --no-merges "origin/${BASE_REF}..HEAD" --format=%s)
else
  echo "verify-commit-messages: set BASE_SHA+HEAD_SHA or BASE_REF" >&2
  exit 1
fi

if [[ ${#SUBJECTS[@]} -eq 0 ]]; then
  echo "verify-commit-messages: no non-merge commits in range (ok)"
  exit 0
fi

bad=0
for subj in "${SUBJECTS[@]}"; do
  if [[ "$subj" =~ ^Revert ]]; then
    continue
  fi
  if [[ "$subj" =~ $conv ]] || [[ "$subj" =~ $conv_no_scope ]]; then
    continue
  fi
  echo "verify-commit-messages: invalid subject: $subj" >&2
  bad=1
done

if [[ "$bad" -ne 0 ]]; then
  echo "verify-commit-messages: expected Conventional Commits on first line, e.g. feat(scope): add thing" >&2
  exit 1
fi

echo "verify-commit-messages: ok (${#SUBJECTS[@]} commit(s))"
exit 0
