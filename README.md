# Vibe Commander

### IVE - Integrated Vibe Environment

> Your AI pair programming command center. One screen. Zero context switching. Pure vibes.

<p align="center">
  <img src="https://img.shields.io/badge/built%20with-go-00ADD8?style=flat-square" alt="Built with Go">
  <img src="https://img.shields.io/badge/vibes-immaculate-ff69b4?style=flat-square" alt="Vibes: Immaculate">
  <img src="https://img.shields.io/badge/tabs-who%20needs%20em-purple?style=flat-square" alt="Tabs: Who needs em">
</p>

<img width="1712" height="1115" alt="Xnapper-2025-12-03-21 57 15" src="https://github.com/user-attachments/assets/03605818-ce17-4ddb-b013-ce5ad4019d0e" />

## Why?

Have been building this as a tool to bring my flow from 99% there to 100%.

I nowadays do pretty much everything using Claude Code and only ever hop into other terminal tabs to view the occasional file or run some git commands.

Vision was to have these minimal facilities in a familiar IDE style layout (Integrated Vibe Environment?) that evokes old time Norton Commander memories.

## Quick Start

```bash
# Install
go install github.com/avitaltamir/vibecommander/cmd/vc@latest

# Run
vc

# Launch with Claude (or your AI of choice)
# Press Alt+A once IVE is running
```

## The Layout

| Panel | What it does |
|-------|-------------|
| **File Tree** | Browse your project. Git status baked in—green for staged, yellow for modified, purple for untracked. |
| **Viewer** | Syntax-highlighted code. Regex search. Diff view for modified files. |
| **Terminal** | A real shell. Run tests, git commands, or fire up your AI assistant. |

## Keybindings

### Panel Control
| Key | Action |
|-----|--------|
| `Alt+1` | Jump to file tree |
| `Alt+2` | Jump to viewer (again for fullscreen) |
| `Alt+3` | Toggle terminal |
| `Alt+A` | Launch AI assistant |

### Navigation
| Key | Action |
|-----|--------|
| `↑/k` `↓/j` | Move up/down |
| `←/h` `→/l` | Collapse/expand |
| `Enter` | Open file / toggle dir |
| `g/G` | Top/bottom |
| `/` | Search (regex) |
| `n/p` | Next/prev match |

### System
| Key | Action |
|-----|--------|
| `Alt+T` | Cycle theme |
| `Ctrl+H` | Help |
| `Ctrl+Q` | Quit |

## Themes

Because your terminal should match your aesthetic.

| Theme | Vibe |
|-------|------|
| **Midnight Miami** | Neon pink + cyan on deep purple. Synthwave dreams. |
| **Piña Colada** | Tropical sunset. Vacation mode activated. |
| **Lobster Boy** | Fresh from the seafood shack. Classy crustacean. |
| **Feral Jungle** | Deep rainforest. Touch grass, but make it terminal. |
| **Vampire Weekend** | Gothic but make it indie. Dark academia core. |

Cycle through with `Alt+T`.

## Requirements

- Go 1.24+
- 256-color terminal
- A [Nerd Font](https://www.nerdfonts.com/) for file icons (optional but recommended)

## Building from Source

```bash
git clone https://github.com/avitaltamir/vibecommander.git
cd vibecommander
go build -o vc ./cmd/vc
./vc
```

---

Built with pure vibes.

MIT License
