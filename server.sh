#!/bin/sh

while true
do
  echo "nvim --headless --listen 0.0.0.0:$1"
  nvim --headless --listen 0.0.0.0:$1
done
