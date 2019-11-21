#!/bin/bash
# Copyright 2019 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

# Usage.
if [ "$#" != 2 ] || [ "$1" == "--help" ] ; then
  echo "Usage: ./create_diff.sh [file_path] [patch_name]"
  echo "Ex: ./creatediff.sh rm/remove_histogram.xml test_diff.patch"
  echo "[file_path] should be relative to prevdata/src and testdata/src."
  echo "The patch will be generated in prevdata/[patch_name]."
  exit 1
fi

FILE_PATH="$1"
FILE_FOLDER="$(dirname "$FILE_PATH")"
FILE_NAME="$(basename "$FILE_PATH")"
PATCH_NAME="$2"

# Get the current directory of the script.
DIR="$(dirname "$(readlink -f "$0")")"

# Create a temp directory and initialize git repository for performing git diff.
tmp_dir=$(mktemp -d -t)
cd "$tmp_dir" || exit
git init --quiet

# Copy over the previous version of the file from prevdata/.
cp "$DIR/prevdata/src/$FILE_PATH" "$tmp_dir"
git add .
git commit --quiet -m "diff1"

# Copy over the current version of the file from testdata/.
cp "$DIR/testdata/src/$FILE_PATH" "$tmp_dir"

# Get the patch.
git diff --src-prefix="a/$FILE_FOLDER/" --dst-prefix="b/$FILE_FOLDER/" \
         --output="$DIR/prevdata/$PATCH_NAME" "$FILE_NAME"

# Clean up temporary directory.
rm -rf "$tmp_dir"