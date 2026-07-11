#!/usr/bin/env bash
# Render the Homebrew formula for what-was-next to stdout.
#
# Usage: REPO=owner/name scripts/render-formula.sh <version> <dist-dir>
#
# <dist-dir> must contain the release binaries what-was-next-{darwin,linux}-amd64
# and what-was-next-{darwin,linux}-arm64. The formula installs the prebuilt
# binary for the host OS and architecture; the app is pure Go
# (modernc.org/sqlite), so there is no build step on the user's machine.
set -euo pipefail

version="${1:?usage: REPO=owner/name render-formula.sh <version> <dist-dir>}"
dist="${2:?usage: REPO=owner/name render-formula.sh <version> <dist-dir>}"
repo="${REPO:?REPO must be set to owner/name}"

base="https://github.com/${repo}/releases/download/${version}"
darwin_arm_sha=$(shasum -a 256 "$dist/what-was-next-darwin-arm64" | awk '{print $1}')
darwin_amd_sha=$(shasum -a 256 "$dist/what-was-next-darwin-amd64" | awk '{print $1}')
linux_arm_sha=$(shasum -a 256 "$dist/what-was-next-linux-arm64" | awk '{print $1}')
linux_amd_sha=$(shasum -a 256 "$dist/what-was-next-linux-amd64" | awk '{print $1}')

cat <<EOF
class WhatWasNext < Formula
  desc "Terminal task manager and time tracker"
  homepage "https://github.com/${repo}"
  version "${version}"

  on_macos do
    on_arm do
      url "${base}/what-was-next-darwin-arm64"
      sha256 "${darwin_arm_sha}"
    end
    on_intel do
      url "${base}/what-was-next-darwin-amd64"
      sha256 "${darwin_amd_sha}"
    end
  end

  on_linux do
    on_arm do
      url "${base}/what-was-next-linux-arm64"
      sha256 "${linux_arm_sha}"
    end
    on_intel do
      url "${base}/what-was-next-linux-amd64"
      sha256 "${linux_amd_sha}"
    end
  end

  def install
    # Only the single OS/arch-matched binary is downloaded; install it as \`what-was-next\`.
    bin.install Dir["what-was-next-*"].first => "what-was-next"
  end

  test do
    # No TTY in the Homebrew test sandbox, so the TUI exits non-zero on startup.
    # That still exercises the binary end-to-end and confirms it's the right program.
    output = shell_output("#{bin}/what-was-next < /dev/null 2>&1", 1)
    assert_match "what-was-next:", output
  end
end
EOF
