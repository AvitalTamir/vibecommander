<p align="center">
  <img src="https://img.shields.io/badge/built%20with-go-00ADD8?style=flat-square" alt="Built with Go">
  <img src="https://img.shields.io/badge/vibes-immaculate-ff69b4?style=flat-square" alt="Vibes: Immaculate">
  <img src="https://img.shields.io/badge/tabs-who%20needs%20em-purple?style=flat-square" alt="Tabs: Who needs em">
</p>

<img width="1512" alt="Vibe Commander Screenshot" src="docs/screenshot.png" />

# Vibe Commander

A terminal-based IVE (Interactive Vibe Environment) for AI coding assistants. Norton Commander vibes, modern AI workflows.

## Why?

Built for developers who live in their terminal with AI assistants like Claude Code, Gemini CLI, or Codex. Instead of jumping between terminal tabs to check files, view diffs, or stage commits—do it all in one place while your AI does the heavy lifting.

## Quick Start

### Download Binary

Grab the latest binary for your platform from the [Releases page](https://github.com/avitaltamir/vibecommander/releases/latest).

### Install with Go

```bash
go install github.com/avitaltamir/vibecommander/cmd/vc@latest
```

### Run

```bash
vc
```

Press `Alt+A` to launch your AI assistant, or `Alt+S` to choose between Claude, Gemini, Codex, or a custom command.

## Features

### File Tree
- Browse your project with vim-style navigation
- Git status indicators (green=staged, yellow=modified, purple=untracked)
- Fuzzy search with `/`
- Stage/unstage files with `Space`
- Compact indent toggle with `Alt+I`

### Code Viewer
- Syntax highlighting
- Regex search (`/`, then `n`/`p` for next/prev)
- Inline diff view for modified files

### Git Panel
- Toggle with `Alt+G` to see staged/unstaged changes
- Stage/unstage files with `Space`
- Commit with `c` (supports GPG signing)

### AI Integration
- Supports Claude Code, Gemini CLI, Codex, or any custom command
- AI selection persists across sessions
- Full terminal emulation—your AI has complete control

### Layout
- Resizable panels with `Alt+[` and `Alt+]`
- Fullscreen content with `Alt+2` (toggle)
- Panel sizes persist across sessions

## Keybindings

### Panels
| Key | Action |
|-----|--------|
| `Alt+1` | Focus file tree |
| `Alt+2` | Focus content (again for fullscreen) |
| `Alt+3` | Toggle terminal |
| `Alt+G` | Toggle git panel |
| `Alt+[` / `Alt+]` | Resize panels |

### Navigation
| Key | Action |
|-----|--------|
| `↑/k` `↓/j` | Move up/down |
| `←/h` `→/l` | Collapse/expand |
| `Enter` | Open file / toggle directory |
| `PgUp/PgDn` | Page scroll |
| `Home/g` `End/G` | Jump to top/bottom |

### Search
| Key | Action |
|-----|--------|
| `/` | Search (file tree: filter, viewer: regex) |
| `n` / `p` | Next/prev match |
| `Esc` | Clear search |

### Git
| Key | Action |
|-----|--------|
| `Space` | Stage/unstage file |
| `c` | Commit (in git panel) |

### Actions
| Key | Action |
|-----|--------|
| `Alt+A` | Launch AI assistant |
| `Alt+S` | Select AI assistant |
| `Alt+T` | Cycle theme |
| `Alt+I` | Toggle compact indent |
| `Ctrl+H` | Toggle help |
| `Ctrl+Q` | Quit |

> **Mac users:** Option key works as Alt—no terminal config needed.

## Themes

Cycle through with `Alt+T`:

| Theme | Vibe |
|-------|------|
| **Midnight Miami** | Neon pink + cyan on deep purple. Synthwave dreams. |
| **Piña Colada** | Tropical sunset. Vacation mode activated. |
| **Lobster Boy** | Fresh from the seafood shack. Classy crustacean. |
| **Feral Jungle** | Deep rainforest. Touch grass, but make it terminal. |
| **Vampire Weekend** | Gothic but make it indie. Dark academia core. |

## Requirements

- 256-color terminal
- A [Nerd Font](https://www.nerdfonts.com/) for file icons (optional but recommended)

## Building from Source

```bash
git clone https://github.com/avitaltamir/vibecommander.git
cd vibecommander
make build
./bin/vc
```

---

100% Vibes, 0% Human Slop.

MIT License
