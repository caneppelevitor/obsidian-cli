# Obsidian CLI

A terminal-based interface for managing daily notes in Obsidian vaults. Built with Go and the [Charm](https://charm.sh) ecosystem (Bubble Tea, Lipgloss, Glamour).

## Features

- **Daily Notes** — Create and edit daily notes with section-based organization (Tasks, Ideas, Questions, Insights)
- **Eisenhower Tasks** — Tag tasks with `#do`, `#delegate`, `#schedule`, `#eliminate` and view them in an interactive matrix
- **File Browser** — Navigate your full Obsidian vault with directory browsing and global fuzzy search (`/`)
- **Inline Editing** — View files with Glamour-rendered markdown, edit with a built-in textarea editor
- **Central Logging** — All tasks, ideas, questions, and insights logged to separate central files with backlinks
- **Adaptive Colors** — Catppuccin palette with automatic light/dark terminal detection

## Install

### From source (requires Go 1.25+)

```bash
git clone https://github.com/caneppelevitor/obsidian-cli-go.git
cd obsidian-cli-go

# Option A: go install (puts binary in ~/go/bin/)
go install .

# Option B: make install (puts binary in /usr/local/bin/)
make install
```

### Verify

```bash
obsidian --help
```

## Setup

```bash
obsidian init
```

This will prompt you for:
1. **Daily notes path** — where daily notes are created (e.g. `~/vault/daily-notes`)
2. **Vault root path** — root directory for the file browser (e.g. `~/vault`)

Configuration is stored in `~/.obsidian-cli/config.yaml`.

## Usage

```bash
obsidian              # Opens today's daily note (default)
obsidian daily        # Same as above
obsidian tasks        # View and manage tasks from CLI
obsidian config       # View config status
obsidian config --edit  # Edit config in $EDITOR
```

### Quick Input Prefixes

In the daily note input bar, use these prefixes to route content:

| Prefix | Section | Example |
|--------|---------|---------|
| `[]` | Tasks | `[] Fix the auth bug #do` |
| `-` | Ideas | `- Try using Redis for caching` |
| `?` | Questions | `? How does the billing system work?` |
| `!` | Insights | `! The API timeout was causing the issue` |

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Tab` | Switch between tabs |
| `e` | Enter edit mode |
| `Esc` | Exit edit mode / go back |
| `Ctrl+S` | Save (in edit mode) |
| `j/k` | Navigate tasks |
| `Enter` | Complete task / open file / enter folder |
| `/` | Global fuzzy search (Files tab) |
| `?` | Toggle help |
| `Ctrl+C` | Quit |

## Configuration

Generate a sample config:

```bash
obsidian init --sample-config
```

Key settings in `~/.obsidian-cli/config.yaml`:

```yaml
vault:
  defaultPath: "/path/to/your/vault/daily-notes"
  rootPath: "/path/to/your/vault"  # File browser root

dailyNotes:
  sections: ["Daily Log", "Tasks", "Ideas", "Questions", "Insights", "Links to Expand"]
  tags: ["#daily", "#inbox"]

interface:
  eisenhowerTags:
    "#do": "131"
    "#delegate": "180"
    "#schedule": "66"
    "#eliminate": "145"
```

## Development

```bash
make build          # Build binary
make test           # Run tests
make lint           # Run go vet
make install        # Build and install to /usr/local/bin
make uninstall      # Remove from /usr/local/bin
make clean          # Remove local binary
```

## License

MIT
