# Obsidian CLI

Terminal interface for managing daily notes in Obsidian with section-based organization and task logging.

<img width="1920" height="1080" alt="image" src="https://github.com/user-attachments/assets/9ed19ea9-5f00-4899-a9f9-456ae3e457ab" />

<img width="1920" height="1080" alt="image" src="https://github.com/user-attachments/assets/ae2fb52c-fadf-4b57-93f5-8c5ffe647fab" />


## Setup

```bash
git clone <this-repo>
cd obsidian-cli
npm install
npm install -g .
```

Then point it to your Obsidian vault:

```bash
obsidian init --vault /path/to/your/vault
```

Or run `obsidian init` without flags for an interactive prompt.

**Tip:** Add a tmux shortcut to open it instantly:
```bash
# ~/.tmux.conf
bind-key -r o new-window "obsidian"
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
