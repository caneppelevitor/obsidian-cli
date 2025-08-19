# Obsidian CLI Manager

A CLI tool for managing your Obsidian vault with a Claude Code-inspired interface. Easily create, edit, and browse your markdown notes from the command line.

## Features

- 📝 **Daily Notes**: Create and edit daily notes with interactive mode
- 📂 **File Browser**: List and view all markdown files in your vault (including subdirectories)
- 🎨 **Colorized Output**: Beautiful, readable terminal interface
- ⚙️ **Configuration**: Set default vault path for convenience
- 🔍 **Interactive Mode**: Edit files with real-time preview
- 📁 **Recursive Search**: Find notes in nested folders

## Installation

### Local Development Setup

1. Clone or download this repository
2. Navigate to the project directory
3. Install dependencies:
   ```bash
   npm install
   ```

### Global Installation (Optional)

To use the CLI from anywhere:

```bash
npm install -g .
```

## Quick Start

1. **Initialize with your vault path:**
   ```bash
   npm start init
   # This sets your vault to: /Users/vitorcaneppele/Documents/Notes do Papai/zettelkasten vault
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

**Interactive Mode Commands:**
- `/view` - Show current file content with line numbers
- `/save` - Save current file
- `/files` - List all files in vault
- `/open` - Open a different file (supports file numbers)
- `/daily` - Switch back to daily note
- `/exit` - Exit interactive mode
- `/help` - Show help

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
Quick setup with your predefined vault path.

```bash
npm start init
```

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
├── jest.config.js       # Test configuration
├── .eslintrc.js        # Linting rules
├── .prettierrc         # Code formatting
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

### Development Mode

```bash
npm run dev daily
```

## Configuration

The CLI stores configuration in `~/.obsidian-cli/config.json`. You can:

- Set a default vault path to avoid using `--vault` option
- Store other preferences (future features)

## Daily Note Template

When creating a new daily note, the CLI uses this template:

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
# Initialize with your vault
npm start init

# Open today's daily note
npm start daily

# In interactive mode, type content or use commands:
# > This is a new note entry
# > /save
# > /view
# > /exit
```

### File Operations

```bash
# List all files
npm start files

# View a specific file
npm start view
# Then enter filename or number when prompted
```

### Using Different Vaults

```bash
# Use a different vault temporarily
npm start daily --vault "/path/to/other/vault"

# Set as new default
npm start config "/path/to/other/vault"
```

## Troubleshooting

### "No vault path specified" Error

Either:
1. Run `npm start init` to set up the default vault
2. Use the `--vault` option: `npm start daily --vault "/path/to/vault"`
3. Set manually: `npm start config "/path/to/vault"`

### Permission Issues

Make sure the CLI has read/write permissions to your vault directory.

### Module Not Found

Run `npm install` in the project directory.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Run tests: `npm test`
4. Run linting: `npm run lint`
5. Submit a pull request

## License

MIT