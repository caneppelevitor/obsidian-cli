# Obsidian CLI

A simple CLI tool for managing daily notes in your Obsidian vault, designed around my personal note-taking workflow with section-based organization.

<img width="1920" height="1080" alt="image" src="https://github.com/user-attachments/assets/91509a6b-9116-48d9-94ed-cdbc8189b285" />


## Installation

```bash
git clone <this-repo>
cd obsidian-cli
npm install
npm install -g .  # For global access
```

## Usage

```bash
# First time setup
npm start init

# Open today's daily note
npm start daily
```

## How It Works

This tool is built around my daily note-taking flow with predefined sections. Notes are organized with markdown headers (## Section Name) and content is automatically placed in the right section based on simple prefix commands.
¡
### Section Commands

Use these prefixes to add content to specific sections:

- `[] task description` → Adds to **Tasks** section as `- [ ] task description`
- `- idea or note` → Adds to **Ideas** section as `- idea or note`
- `? question here` → Adds to **Questions** section as `- question here`
- `! insight or reflection` → Adds to **Insights** section as `- insight or reflection`

### Task Logging

All tasks created with the `[]` prefix are automatically logged to a centralized task log file in addition to being added to your daily note. This creates a master list of all tasks across all daily notes.

**Task Log Features:**
- Includes a backlink to the daily note where it was created
- Maintains a complete history of all tasks
- Configurable log file location (default: `tasks-log.md` in vault root)

### Interactive Mode

- **Enter**: Submit input
- **Ctrl+C**: Exit
- `/save`: Save file
- `/exit`: Exit

The tool automatically finds the appropriate section header and inserts content there, or appends to the end if no matching section exists.
