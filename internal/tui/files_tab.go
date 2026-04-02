package tui

import (
	"fmt"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/list"

	"github.com/caneppelevitor/obsidian-cli/internal/vault"
)

// FileItem implements list.Item and list.DefaultItem for the file browser.
type FileItem struct {
	name string
	path string
}

func (f FileItem) FilterValue() string { return f.name }
func (f FileItem) Title() string       { return f.name }
func (f FileItem) Description() string { return "" }

// newFileList creates a list.Model for file browsing.
func newFileList(files []string, width, height int) list.Model {
	items := make([]list.Item, len(files))
	for i, f := range files {
		items[i] = FileItem{name: f}
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(1)
	delegate.SetSpacing(0)
	delegate.ShowDescription = false

	l := list.New(items, delegate, width, height)
	l.Title = "Vault Files"
	l.SetShowHelp(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	return l
}

// loadFileList loads the file list asynchronously.
func (m *AppModel) loadFileList() tea.Cmd {
	return func() tea.Msg {
		files, err := vault.ListMarkdownFiles(m.vaultPath)
		if err != nil {
			return StatusMsg{Text: fmt.Sprintf("Error listing files: %v", err)}
		}
		return FileListMsg{Files: files}
	}
}

// handleFileSelection opens the selected file from the list.
func (m *AppModel) handleFileSelection() tea.Cmd {
	selected := m.fileList.SelectedItem()
	if selected == nil {
		return nil
	}

	fileItem, ok := selected.(FileItem)
	if !ok {
		return nil
	}

	filePath := filepath.Join(m.vaultPath, fileItem.name)
	m.activeTab = tabNotes
	return loadFileCmd(filePath)
}
