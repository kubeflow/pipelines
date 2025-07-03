#!/bin/bash
set -euo pipefail

# This script frees up disk space on GitHub Actions runners.
# Several GHA workflows were failing with "no space left on device" errors.
# This script is only meant to run in GitHub Actions CI environment.

# Safety check: Only run on GitHub Actions
if [[ "${GITHUB_ACTIONS:-false}" != "true" ]]; then
    echo "ERROR: This script is for GitHub Actions runners only!"
    exit 1
fi

echo "=== Initial disk usage ==="
df -h

echo "=== Freeing up disk space ==="

# Remove large directories not needed for KFP tests
sudo rm -rf /usr/share/dotnet
sudo rm -rf /opt/ghc
sudo rm -rf /usr/local/share/boost
sudo rm -rf /usr/local/lib/android
sudo rm -rf /usr/local/.ghcup
sudo rm -rf /usr/share/swift

# Selectively remove large tools from hostedtoolcache while preserving Go, Node, Python
# Remove these specific large tools that aren't needed for KFP tests
sudo rm -rf /opt/hostedtoolcache/CodeQL || true
sudo rm -rf /opt/hostedtoolcache/Java_* || true
sudo rm -rf /opt/hostedtoolcache/Ruby || true
sudo rm -rf /opt/hostedtoolcache/PyPy || true
sudo rm -rf /opt/hostedtoolcache/boost || true

# Clean package manager
sudo apt-get autoremove -y
sudo apt-get autoclean

# Clean Docker
docker system prune -af --volumes
docker image prune -af

# Clean containerd
sudo systemctl stop containerd || true
sudo rm -rf /var/lib/containerd/io.containerd.snapshotter.v1.overlayfs/snapshots/* || true
sudo systemctl start containerd || true

echo "=== Final disk usage ==="
df -h 