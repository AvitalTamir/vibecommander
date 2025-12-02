# Vibe Commander

A minimal TUI for working with AI coding assistants. View files, check git status, and run your AI assistant—all without jumping between terminal tabs.

Built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## The Problem

When working with AI assistants like Claude, you constantly need to:
- Check file contents to give context
- Look at git status to see what's changed
- Run terminal commands
- Switch back to the AI conversation

That's a lot of `Cmd+Tab` or terminal tab switching. Vibe Commander puts everything in one place.

## What It Does

```
┌─────────────┬──────────────────────────────────────┐
│ FILES       │ VIEWER                               │
│             │                                      │
│ ├── src/    │ (file contents with syntax          │
│ │   ├── ... │  highlighting and search)           │
│ │   └── ... │                                      │
│ └── ...     │                                      │
├─────────────┴──────────────────────────────────────┤
│ TERMINAL                                           │
│ (integrated shell / AI assistant)                  │
└────────────────────────────────────────────────────┘
```

- **File tree** on the left - browse and select files
- **Viewer** on the right - see file contents with syntax highlighting
- **Terminal** at the bottom - run commands or your AI assistant
- **Git status** in the status bar and file tree

## Installation

```bash
go install github.com/avitaltamir/vibecommander/cmd/vibecommander@latest
```

Or build from source:

```bash
git clone https://github.com/avitaltamir/vibecommander.git
cd vibecommander
go build -o vibecommander ./cmd/vibecommander
```

## Keybindings

### Panels
| Key | Action |
|-----|--------|
| `Alt+1` | Focus file tree |
| `Alt+2` | Focus viewer (press again for fullscreen) |
| `Alt+3` | Toggle terminal |
| `Alt+A` | Launch AI assistant (runs `claude` command) |

### Navigation
| Key | Action |
|-----|--------|
| `↑/k`, `↓/j` | Move up/down |
| `←/h`, `→/l` | Collapse/expand directories |
| `Enter` | Open file or toggle directory |
| `PgUp/PgDn` | Page up/down |
| `g/G` | Go to top/bottom |

### Search (in viewer)
| Key | Action |
|-----|--------|
| `/` | Open search (regex supported) |
| `Enter` | Execute search |
| `n/p` | Next/previous match |
| `Esc` | Clear search |

### General
| Key | Action |
|-----|--------|
| `Alt+T` | Cycle theme |
| `Ctrl+H` | Help |
| `Ctrl+Q` | Quit |

## Themes

Press `Alt+T` to cycle through themes:

- **Midnight Miami** - Neon pink and cyan on deep purple
- **Piña Colada** - Tropical sunset vibes
- **Lobster Boy** - Fresh from the seafood shack
- **Feral Jungle** - Deep in the rainforest
- **Vampire Weekend** - Gothic but make it indie

## Git Integration

The file tree shows git status at a glance:
- **Green** - staged
- **Yellow** - modified (unstaged)
- **Purple** - untracked

The status bar shows your current branch and ahead/behind status.

## Requirements

- Go 1.24+
- Terminal with 256 color support
- A nerd font for file icons (optional)

## License

MIT
