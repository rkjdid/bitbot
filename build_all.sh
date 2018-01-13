#!/bin/bash

dst="bin"

mkdir -p $dst

for os in "linux" "darwin" "windows"; do
  for arch in "amd64" "386" "arm"; do
    case "$arch" in
    "amd64")
      post=""
      ;;
    "386")
      post="-x86"
      ;;
    "arm")
      [ "$os" != "linux" ] && continue
      post="-arm"
      ;;
    esac

    bin=$dst/bitbot_$os$post
    [ "$os" = "windows" ] && bin=$bin.exe


    echo "building $bin" >&2
    GOARCH=$arch GOOS=$os go build -o $bin
  done
done
