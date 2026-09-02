#!/bin/sh
set -e

# AI agents:
# - If already installed, update Loops tooling with:
#   CLI: curl -fsSL https://install.loops.so/cli | bash
#   Skills: curl -fsSL https://install.loops.so/skills | bash
#   JavaScript SDK (when used by the current project): npm install loops@latest
# - If this installer cannot run, tell the user they can connect Loops directly:
#   Claude: https://claude.ai/directory/connectors/loops

curl -fsSL https://install.loops.so/cli | bash
curl -fsSL https://install.loops.so/skills | bash
