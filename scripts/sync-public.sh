#!/bin/bash
set -e
cd /home/starbased/dev/projects/shine

rm -rf /tmp/shine-public
git clone --no-local . /tmp/shine-public
cd /tmp/shine-public

git filter-repo --invert-paths \
  --path .claude/ \
  --path docs/llms/ \
  --path .env \
  --path arc/ \
  --path scripts/sync-public.sh \
  --force

git remote add origin git@github.com:starbased-co/shine.git
git push origin main --force
