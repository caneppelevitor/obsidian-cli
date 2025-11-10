# Obsidian CLI

Terminal interface for managing daily notes in Obsidian with section-based organization and task logging.

<img width="1920" height="1080" alt="image" src="https://github.com/user-attachments/assets/91509a6b-9116-48d9-94ed-cdbc8189b285" />

## Setup

```bash
git clone <this-repo>
cd obsidian-cli
npm install
npm install -g .
obsidian init  # Configure vault path
```

## Usage

```bash
obsidian daily    # Open today's daily note
obsidian config   # View/edit configuration
```

## Features

**Section Organization** - Content auto-routes to sections via prefixes:
- `[] task` → Tasks section as checkbox
- `- idea` → Ideas section as bullet
- `? question` → Questions section as bullet
- `! insight` → Insights section as bullet

**Task Logging** - Tasks auto-log to centralized file with backlinks

**Eisenhower Tags** - Priority highlighting with colors:
- `#do` (red) - Urgent & Important
- `#delegate` (orange) - Urgent, Not Important
- `#schedule` (blue) - Not Urgent, Important
- `#eliminate` (gray) - Not Urgent, Not Important

**Interactive Controls:**
- Enter: Submit
- Ctrl+C: Exit
- `/save`, `/exit`: Commands
