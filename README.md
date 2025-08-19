# Obsidian CLI Manager

A CLI tool for managing your Obsidian vault with an interactive interface. Create, edit, and browse markdown notes from the command line.

## Features

- Daily Notes: Create and edit daily notes with interactive mode
- File Browser: List and view all markdown files in your vault
- Interactive Interface: Edit files with real-time preview and navigation
- Line-specific Editing: Insert or replace content at specific line numbers
- Configuration: Set default vault path for convenience
- Bullet Point Auto-formatting: All text input automatically becomes bullet points

## Installation

### Local Development Setup

1. Clone or download this repository
2. Navigate to the project directory
3. Install dependencies:
   ```bash
   npm install
   ```

### Global Installation

To use the CLI from anywhere:

```bash
npm install -g .
```

## Quick Start

1. **Initialize with your vault path:**
   ```bash
   npm start init
   ```

2. **Start using the CLI:**
   ```bash
   # Open today's daily note
   npm start daily
   
   # Browse all files
   npm start files
   
   # View mode to read files
   npm start view
   ```

## Commands

### `daily`
Opens or creates today's daily note in interactive mode.

```bash
npm start daily
# or specify vault path
npm start daily --vault "/path/to/your/vault"
```

**Interactive Mode:**
- Tab: Switch between input and file view
- Enter: Submit input
- Ctrl+C: Exit

**Input Patterns:**
- `[] your text` - Add regular text as bullet point
- `[5] your text` - Replace line 5 with your text
- `[n5] your text` - Insert new line after line 5
- `/save` - Save current file
- `/exit` - Exit interactive mode

### `view`
Browse and view files in your vault.

```bash
npm start view
```

### `files`
List all markdown files in your vault.

```bash
npm start files
```

### `config`
Manage configuration settings.

```bash
# Show current default vault
npm start config

# Set new default vault
npm start config "/path/to/your/vault"
```

### `init`
Quick setup with predefined vault path.

```bash
npm start init
```

## Interactive Mode Features

- **Navigation**: Use Tab to switch between input box and file display
- **Cursor Movement**: Arrow keys, Home, End work in input box
- **Auto-scroll**: File display automatically scrolls to show latest content
- **Line Numbers**: All lines are numbered for easy reference
- **Smart Input**: Empty brackets or spaces are ignored

## Line Editing Syntax

- `[] hello world` - Adds "- hello world" to end of file
- `[12] important note` - Replaces line 12 with "- important note"
- `[n12] new task` - Inserts "- new task" as new line after line 12

## Project Structure

```
obsidian-cli/
├── src/
│   ├── cli.js           # Main CLI entry point
│   ├── obsidian-cli.js  # Core functionality
│   └── config.js        # Configuration management
├── tests/
│   └── obsidian-cli.test.js  # Unit tests
├── package.json
├── jest.config.js
└── README.md
```

## Development

### Running Tests

```bash
npm test
```

### Linting

```bash
npm run lint
```

### Code Formatting

```bash
npm run format
```

## Configuration

The CLI stores configuration in `~/.obsidian-cli/config.json` with your default vault path.

## Daily Note Template

New daily notes use this template:

```markdown
# Daily Note - YYYY-MM-DD

## Tasks
- [ ] 

## Notes


## Reflections


---
Created: [timestamp]
```

## Examples

### Basic Usage

```bash
# Initialize
npm start init

# Open daily note
npm start daily

# Type content:
# [] This becomes a bullet point
# [5] This replaces line 5
# [n10] This inserts after line 10
```

### File Operations

```bash
# List files
npm start files

# View specific file
npm start view
```

## Troubleshooting

### "No vault path specified" Error

1. Run `npm start init` to set up default vault
2. Use `--vault` option: `npm start daily --vault "/path/to/vault"`
3. Set manually: `npm start config "/path/to/vault"`

### Permission Issues

Ensure CLI has read/write permissions to your vault directory.

### Module Not Found

Run `npm install` in the project directory.

## License

MIT