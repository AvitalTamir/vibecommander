# Vibe Commander

### IVE - Integrated Vibe Environment

> Your AI pair programming command center. One screen. Zero context switching. Pure vibes.

<p align="center">
  <img src="https://img.shields.io/badge/built%20with-go-00ADD8?style=flat-square" alt="Built with Go">
  <img src="https://img.shields.io/badge/vibes-immaculate-ff69b4?style=flat-square" alt="Vibes: Immaculate">
  <img src="https://img.shields.io/badge/tabs-who%20needs%20em-purple?style=flat-square" alt="Tabs: Who needs em">
</p>

```
┌─ FILES ─────────┬─ Claude ──────────────────── 47% ─┐
│ 📁 src/         │                                   │
│  ├─ 📄 main.go  │  I've updated the function to     │
│  └─ 📁 utils/   │  handle edge cases. The changes   │
│ 📁 tests/       │  are in utils/parser.go...        │
│ 📄 go.mod       │                                   │
├─ TERMINAL ● ────────────────────────────────────────┤
│ $ go test ./...                                     │
│ ok   myproject/utils  0.042s                        │
└─────────────────────────────────────────────────────┘
```

## Why Vibe Commander?

You're pair programming with an AI. You need to:
- Show it a file → *switch tab*
- Check what changed → *switch tab*
- Run a command → *switch tab*
- Go back to the conversation → *switch tab*

**Stop. Switching. Tabs.**

Vibe Commander gives you everything in one view. Browse files, read code with syntax highlighting, watch your git status, run commands, and chat with your AI—all without your fingers leaving the keyboard.

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
