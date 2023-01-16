#!/bin/sh
# Based on Deno installer: Copyright 2019 the Deno authors. All rights reserved. MIT license.
# TODO(everyone): Keep this script simple and easily auditable.

set -e

os=$(uname -s)
arch=$(uname -m)
# version=${1:-latest}
# version=${1:-$(curl --silent https://api.github.com/repos/usecakework/cakeworkctl/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')}
# version_number=$(echo ${version} | sed 's/v//g')

install_dir="${CAKEWORK_INSTALL_DIR:-$HOME/.cakework/bin}"
bin="$install_dir/cakework"
fly_bin="$install_dir/fly"

if [ ! -d "$install_dir" ]; then
  mkdir -p "$install_dir"
fi

# TODO get latest release. currently just hard-coding
# https://github.com/usecakework/cakeworkctl/releases/latest/download/latest/cakeworkctl_latest_Darwin_arm64.tar.gz
# /owner/name/releases/latest/download/asset-name.zip
# release_url=$(curl --silent --write-out "%{redirect_url}\n" --output /dev/null https://cakeworkctl-downloads.s3.us-west-2.amazonaws.com/1.0.44/cakeworkctl_1.0.44_Darwin_arm64.tar.gz)



extract_dir="$(mktemp -d)"
# disabled progress bar here
curl -q --fail --location --output "$extract_dir/cakework.tar.gz" "https://cakeworkctl-downloads.s3.us-west-2.amazonaws.com/1.0.61/cakeworkctl_1.0.61_${os}_${arch}.tar.gz"
cd "$extract_dir"
tar xzf "cakework.tar.gz"
mv "$extract_dir/cakework" "$bin"
chmod +x "$bin"
rm -rf "$extract_dir"

###########
# Install fly
###########
os=$(uname -s)
arch=$(uname -m)
version=${1:-latest}

flyctl_uri=$(curl -s ${FLY_FORCE_TRACE:+ -H "Fly-Force-Trace: $FLY_FORCE_TRACE"} https://api.fly.io/app/flyctl_releases/$os/$arch/$version)
if [ ! "$flyctl_uri" ]; then
	echo "Error: Unable to find a flyctl release for $os/$arch/$version - see github.com/superfly/flyctl/releases for all versions" 1>&2
	exit 1
fi

flyctl_install="${FLYCTL_INSTALL:-$HOME/.cakework/.fly}"

bin_dir="$flyctl_install/bin"
exe="$bin_dir/flyctl"
simexe="$bin_dir/fly"

if [ ! -d "$bin_dir" ]; then
 	mkdir -p "$bin_dir"
fi

curl -q --fail --location --progress-bar --output "$exe.tar.gz" "$flyctl_uri"
cd "$bin_dir"
tar xzf "$exe.tar.gz"
chmod +x "$exe"
rm "$exe.tar.gz"

ln -sf $exe $simexe

if [ "${1}" = "prerel" ] || [ "${1}" = "pre" ]; then
	"$exe" version -s "shell-prerel"
else
	"$exe" version -s "shell"
fi

if ! command -v flyctl >/dev/null; then
	case $SHELL in
	/bin/zsh) shell_profile=".zshrc" ;;
	*) shell_profile=".bash_profile" ;;
	esac
fi


echo "cakework CLI was installed successfully to $bin"
if command -v cakework >/dev/null; then
  echo "Run 'cakework --help' to get started"
else
  case $SHELL in
  /bin/zsh) shell_profile=".zshrc" ;;
  *) shell_profile=".bash_profile" ;;
  esac
  echo "Manually add the directory to your \$HOME/$shell_profile (or similar)"
  echo "  export CAKEWORK_INSTALL_DIR=\"$install_dir\""
  echo "  export PATH=\"\$CAKEWORK_INSTALL_DIR:\$PATH\""
  echo "Run '$bin --help' to get started"
fi

