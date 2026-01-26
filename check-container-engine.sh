#!/bin/bash

# Container Engine Detection and Validation Script
# This script helps users understand which container engine will be used

set -e

echo "🔍 Container Engine Detection Report"
echo "=================================="

# Check for Podman
if command -v podman >/dev/null 2>&1; then
    echo "✅ Podman found: $(podman --version)"
    PODMAN_AVAILABLE=true
else
    echo "❌ Podman not found"
    PODMAN_AVAILABLE=false
fi

# Check for Docker
if command -v docker >/dev/null 2>&1; then
    echo "✅ Docker found: $(docker --version)"
    DOCKER_AVAILABLE=true
else
    echo "❌ Docker not found"
    DOCKER_AVAILABLE=false
fi

echo ""

# Determine which engine would be used
if [ "$DOCKER_AVAILABLE" = true ] && [ "$PODMAN_AVAILABLE" = true ]; then
    echo "🎯 Both container engines are available!"
    echo "   The Makefile will automatically use: Docker (default)"
    echo "   To use Podman instead: CONTAINER_ENGINE=podman make <target>"
elif [ "$PODMAN_AVAILABLE" = true ]; then
    echo "🎯 Only Podman is available - it will be used automatically"
elif [ "$DOCKER_AVAILABLE" = true ]; then
    echo "🎯 Only Docker is available - it will be used automatically"
else
    echo "❌ No container engine found!"
    echo "   Please install either Docker or Podman to continue"
    exit 1
fi

echo ""
echo "📋 Quick Commands:"
echo "   Check current detection: make container-engine"
echo "   Validate your engine:    make validate-engine"
echo "   Deploy with auto-detect: make local/deploy"
echo "   Deploy with Docker:      CONTAINER_ENGINE=docker make local/deploy"
echo "   Deploy with Podman:      CONTAINER_ENGINE=podman make local/deploy"

if [ "$PODMAN_AVAILABLE" = true ]; then
    echo ""
    echo "💡 Podman Tips:"
    echo "   - If you encounter permission issues, try:"
    echo "     ADDITIONAL_CONTAINER_ENGINE_PARAMS=\"--privileged\" make local/deploy"
    echo "   - For SELinux systems, volume mounts include :z labels automatically"
fi

echo ""
echo "✅ Environment check complete!"
