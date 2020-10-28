#!/bin/sh

set -e -u

GIT_INDEX_FILE="$(mktemp)"
if [ ! -f "$GIT_INDEX_FILE" ]; then
    exit 1
fi
trap 'rm "${GIT_INDEX_FILE}"' EXIT
cp .git/index "$GIT_INDEX_FILE"

export GIT_INDEX_FILE
git add --all
git write-tree