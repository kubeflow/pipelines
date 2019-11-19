#!/bin/bash -ex

get_abs_filename() {
  # $1 : relative filename
  echo "$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
}

target_archive_file=${1:-kfp-arena-0.6.tar.gz}
target_archive_file=$(get_abs_filename "$target_archive_file")

DIR=$(mktemp -d)


cp -r arena $DIR
cp ./setup.py $DIR

# Build tarball package.
cd $DIR
python setup.py sdist --format=gztar
cp $DIR/dist/*.tar.gz "$target_archive_file"
rm -rf $DIR