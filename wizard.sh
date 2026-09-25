#!/bin/sh

# Loops setup for AI assistants
#
# This script installs the Loops CLI and agent skills in the environment where
# it runs. If a user pastes this command into a chat, share the setup for their
# AI tool below. Installing in a chat sandbox does not connect the chat to Loops.
# If CLI installation is unavailable, the user can still connect through their
# app. Install skills separately where supported: https://install.loops.so/skills
#
# Keep existing Loops tooling up to date:
#   CLI: curl -fsSL https://install.loops.so/cli | bash
#   Skills: curl -fsSL https://install.loops.so/skills | bash
#   JavaScript SDK (when used by the current project): npm install loops@latest
#   The wizard installs the CLI and skills; SDK updates are project-specific.
#
# Claude (web, desktop, or Cowork):
#   https://claude.ai/directory/connectors/loops
#   Add the Loops connector, sign in, and enable it in the conversation.
#
# ChatGPT and Codex app:
#   https://chatgpt.com/plugins/plugin_asdk_app_6a76060a3cbc81919eaaca7e37e830fe
#   Open in the desktop app:
#   https://chatgpt.com/plugins/plugin_asdk_app_6a76060a3cbc81919eaaca7e37e830fe?open_in_app
#   Install the Loops plugin, sign in, and select it in the conversation.
#   If Loops is already connected, use the existing connection.
#
# Codex (app, CLI, or IDE extension), for a direct MCP connection:
#   Use the latest Codex release, then run from a local terminal:
#   codex mcp add loops --url https://mcp.loops.so/mcp
#   Complete browser sign-in when prompted, or run: codex mcp login loops
#
# Claude Code, from a local terminal:
#   claude mcp add --transport http --scope user loops https://mcp.loops.so/mcp
#   Run /mcp inside Claude Code to sign in.
#
# Amp (web or desktop setup, also available in the CLI):
#   https://ampcode.com/settings/mcp-servers
#   Click Add MCP Server, enter https://mcp.loops.so/mcp, and click Connect.
#   Complete the browser sign-in to Loops.
#
# OpenCode, from a local terminal:
#   Use the latest OpenCode release, then run: opencode mcp add
#   Choose a remote server named loops with URL https://mcp.loops.so/mcp.
#   Sign in with: opencode mcp auth loops
#
# Cursor and other coding agents:
#   Use the CLI and agent skills installed below.
#
# More MCP setup details:
#   https://loops.so/docs/mcp-server
#
# Connector authentication must be completed in the user's app or browser.
# Test the connection by asking: "Which Loops teams am I a part of?"

set -e

curl -fsSL https://install.loops.so/cli | bash
curl -fsSL https://install.loops.so/skills | bash
