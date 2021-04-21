#!/bin/bash
set -x
set -o errexit
set -o pipefail

# Get a clean checkout
git fetch origin || true
git checkout origin/master || true
git reset HEAD || true
git checkout -- || true

# Resolve pending metadata
./main

# Create a PR
git checkout -b resolve-daily-conflict
git add .
git commit -m 'Resolve daily merge conflicts'
git push