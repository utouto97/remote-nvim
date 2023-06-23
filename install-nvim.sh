#!/bin/sh

if [ "$(uname)" = "Linux" ]; then
  if [ -z $(command -v wget) ] || [ -z $(command -v tar) ]; then
    if [ -n "$(command -v apt)" ]; then
      apt update -y
      apt install -y wget tar
    fi
  fi
  if [ -z $(command -v git) ]; then
    if [ -n "$(command -v apk)" ]; then
      apk add git
    fi
  fi

  filename="nvim-linux64.tar.gz"
  version="stable"
  cd /tmp
  wget "https://github.com/neovim/neovim/releases/download/${version}/${filename}"
  tar zxf "${filename}"
  cp nvim-linux64/bin/nvim /usr/local/bin/
  cp -R nvim-linux64/share/* /usr/local/share/
fi
