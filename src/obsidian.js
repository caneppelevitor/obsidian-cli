#!/usr/bin/env node

const fs = require('fs').promises;
const path = require('path');
const blessed = require('blessed');
const config = require('./config');

class ObsidianInterface {
  constructor(vaultPath) {
    this.vaultPath = vaultPath;
    this.currentFile = null;
    this.currentContent = '';
    this.screen = null;
    this.fileBox = null;
    this.inputBox = null;
  }

  getTodayDate() {
    return new Date().toISOString().split('T')[0];
  }

  processTemplate(template) {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const day = String(now.getDate()).padStart(2, '0');
    
    return template.replace(/\{\{date:YYYY-MM-DD\}\}/g, `${year}-${month}-${day}`);
  }

  async processContentInput(input) {
    const trimmed = input.trim();
    
    if (trimmed.startsWith('[]')) {
      const content = trimmed.slice(2).trim();
      await this.addToSection('Tasks', `- [ ] ${content}`);
    } else if (trimmed.startsWith('-')) {
      const content = trimmed.slice(1).trim();
      await this.addToSection('Ideas', `- ${content}`);
    } else if (trimmed.startsWith('?')) {
      const content = trimmed.slice(1).trim();
      await this.addToSection('Questions', `- ${content}`);
    } else if (trimmed.startsWith('!')) {
      const content = trimmed.slice(1).trim();
      await this.addToSection('Insights', `- ${content}`);
    } else {
      await this.addContent(trimmed);
    }
  }

  async addToSection(sectionName, content) {
    const lines = this.currentContent.split('\n');
    const sectionIndex = this.findSectionIndex(lines, sectionName);
    
    if (sectionIndex === -1) {
      await this.addContent(content);
      return;
    }
    
    let insertIndex = sectionIndex + 1;
    let lastContentLine = sectionIndex;
    let hasContent = false;
    
    while (insertIndex < lines.length && !lines[insertIndex].startsWith('## ')) {
      if (lines[insertIndex].trim() !== '') {
        lastContentLine = insertIndex;
        hasContent = true;
      }
      insertIndex++;
    }
    
    if (!hasContent) {
      lines.splice(sectionIndex + 1, 0, content);
    } else {
      lines.splice(lastContentLine + 1, 0, content);
    }
    
    this.currentContent = lines.join('\n');
    await this.saveFileContent();
    this.updateFileDisplay();
  }

  findSectionIndex(lines, sectionName) {
    for (let i = 0; i < lines.length; i++) {
      if (lines[i].startsWith('## ') && lines[i].includes(sectionName)) {
        return i;
      }
    }
    return -1;
  }

  getDailyNotePath() {
    return path.join(this.vaultPath, `${this.getTodayDate()}.md`);
  }

  async ensureDailyNote() {
    const dailyNotePath = this.getDailyNotePath();

    try {
      await fs.access(this.vaultPath);
    } catch (error) {
      await fs.mkdir(this.vaultPath, { recursive: true });
    }

    try {
      await fs.access(dailyNotePath);
    } catch (error) {
      const template = `# {{date:YYYY-MM-DD}}

##  Insights

## Tasks

## Ideas

## Questions

## Links to Expand

## Tags
#daily #inbox
`;
      const processedTemplate = this.processTemplate(template);
      await fs.writeFile(dailyNotePath, processedTemplate);
    }

    this.currentFile = dailyNotePath;
    await this.loadFileContent();
  }

  async loadFileContent() {
    if (this.currentFile) {
      this.currentContent = await fs.readFile(this.currentFile, 'utf-8');
    }
  }

  async saveFileContent() {
    if (this.currentFile) {
      await fs.writeFile(this.currentFile, this.currentContent);
    }
  }

  setupUI() {
    this.screen = blessed.screen({
      smartCSR: true,
      title: 'Obsidian CLI',
      dockBorders: true,
      ignoreDockContrast: true
    });

    this.fileBox = blessed.box({
      top: 0,
      left: 0,
      width: '100%',
      height: '85%',
      border: {
        type: 'line',
        fg: 'cyan'
      },
      style: {
        fg: 'white',
        border: {
          fg: 'cyan'
        }
      },
      scrollable: true,
      alwaysScroll: true,
      mouse: true,
      keys: true,
      tags: true
    });

    this.inputBox = blessed.textbox({
      bottom: 0,
      left: 0,
      width: '100%',
      height: 3,
      border: {
        type: 'line',
        fg: 'yellow'
      },
      style: {
        fg: 'white',
        border: {
          fg: 'yellow'
        }
      },
      input: true,
      keys: true,
      mouse: true,
      cursor: 'line',
      cursorBlink: true,
      inputOnFocus: true,
      vi: false
    });

    this.screen.append(this.fileBox);
    this.screen.append(this.inputBox);

    this.resetInput();

    this.inputBox.on('submit', async () => {
      const text = this.inputBox.getValue();
      const content = this.extractContent(text);

      if (content.trim()) {
        await this.processInput(content);
      }

      this.resetInput();
    });

    this.inputBox.on('keypress', (ch, key) => {
      if (key && (key.name === 'left' || key.name === 'right' ||
                  key.name === 'home' || key.name === 'end')) {
        return;
      }

      setTimeout(() => {
        const current = this.inputBox.getValue();
        if (!current.startsWith('> ')) {
          this.resetInput();
        }
      }, 10);
    });


    this.screen.key(['escape', 'C-c'], () => {
      process.exit(0);
    });

    this.inputBox.key(['escape'], () => {
      process.exit(0);
    });

    this.inputBox.focus();

    this.screen.render();
  }

  resetInput() {
    this.inputBox.clearValue();
    this.inputBox.setValue('> ');
    this.inputBox.focus();
    this.screen.render();
  }

  extractContent(text) {
    return text.replace(/^>\s*/, '');
  }

  updateFileDisplay() {
    const lines = this.currentContent.split('\n');
    const numberedLines = lines.map((line, index) => {
      const lineNum = (index + 1).toString().padStart(3, ' ');
      const styledLine = this.styleLineContent(line);
      return `${lineNum} │ ${styledLine}`;
    });

    this.fileBox.setLabel(` ${path.basename(this.currentFile)} `);
    this.fileBox.setContent(numberedLines.join('\n'));

    this.fileBox.scrollTo(this.fileBox.getScrollHeight());
    this.screen.render();
  }

  styleLineContent(line) {
    if (line.match(/^##\s+/)) {
      return `{cyan-fg}{bold}${line}{/bold}{/cyan-fg}`;
    }
    
    if (line.match(/^#\s+/)) {
      return `{magenta-fg}{bold}${line}{/bold}{/magenta-fg}`;
    }
    
    if (line.match(/^\s*-\s+\[[ x]\]\s+/)) {
      return `{green-fg}${line}{/green-fg}`;
    }
    
    if (line.match(/^\s*-\s+/)) {
      return `{yellow-fg}${line}{/yellow-fg}`;
    }
    
    if (line.match(/^#\w+/)) {
      return `{blue-fg}${line}{/blue-fg}`;
    }
    
    return line;
  }

  async processInput(input) {
    const trimmed = input.trim();

    if (trimmed === '/exit' || trimmed === '/quit') {
      process.exit(0);
    }

    if (trimmed === '/clear') {
      const template = `# {{date:YYYY-MM-DD}}\n\n##  Insights\n\n## Tasks\n\n## Ideas\n\n## Questions\n\n## Links to Expand\n\n## Tags\n#daily #inbox\n`;
      this.currentContent = this.processTemplate(template);
      await this.saveFileContent();
      this.updateFileDisplay();
      return;
    }

    if (trimmed === '/files') {
      await this.showFiles();
      return;
    }

    if (trimmed.startsWith('/open ')) {
      const filename = trimmed.slice(6);
      await this.openFile(filename);
      return;
    }


    if (trimmed !== '') {
      await this.processContentInput(trimmed);
    }
  }

  async addContent(content, targetLine = null) {
    const lines = this.currentContent.split('\n');

    if (targetLine !== null) {
      if (targetLine <= lines.length) {
        lines[targetLine - 1] = content;
      } else {
        while (lines.length < targetLine - 1) {
          lines.push('');
        }
        lines.push(content);
      }
    } else {
      if (lines[lines.length - 1] === '') {
        lines[lines.length - 1] = content;
      } else {
        lines.push(content);
      }
    }

    this.currentContent = lines.join('\n');
    await this.saveFileContent();
    this.updateFileDisplay();
  }

  async insertNewLine(content, afterLine) {
    const lines = this.currentContent.split('\n');

    if (afterLine <= lines.length) {
      lines.splice(afterLine, 0, content);
    } else {
      while (lines.length < afterLine) {
        lines.push('');
      }
      lines.push(content);
    }

    this.currentContent = lines.join('\n');
    await this.saveFileContent();
    this.updateFileDisplay();
  }

  async start() {
    await this.ensureDailyNote();
    this.setupUI();
    this.updateFileDisplay();
  }

  async showFiles() {
    try {
      const files = await this.getMarkdownFiles(this.vaultPath);
      let fileList = 'Files:\n\n';
      files.slice(0, 15).forEach((file, index) => {
        fileList += `  ${index + 1}. ${file}\n`;
      });
      if (files.length > 15) {
        fileList += `\n... and ${files.length - 15} more`;
      }
      fileList += '\n\nType: /open filename';

      this.fileBox.setContent(fileList);
      this.fileBox.setLabel(' Files ');
      this.screen.render();
    } catch (error) {
      this.fileBox.setContent('Error listing files');
      this.screen.render();
    }
  }

  async openFile(filename) {
    try {
      let filePath;

      if (filename.includes('/') || filename.includes('.')) {
        filePath = path.join(this.vaultPath, filename);
      } else {
        const files = await this.getMarkdownFiles(this.vaultPath);
        const match = files.find(f =>
          f.toLowerCase().includes(filename.toLowerCase()) ||
          path.basename(f, '.md').toLowerCase() === filename.toLowerCase()
        );

        if (match) {
          filePath = path.join(this.vaultPath, match);
        } else {
          return;
        }
      }

      await fs.access(filePath);
      this.currentFile = filePath;
      await this.loadFileContent();
      this.updateFileDisplay();

    } catch (error) {
    }
  }

  async getMarkdownFiles(dir, allFiles = []) {
    const items = await fs.readdir(dir);

    for (const item of items) {
      const fullPath = path.join(dir, item);
      const stat = await fs.stat(fullPath);

      if (stat.isDirectory() && !item.startsWith('.')) {
        await this.getMarkdownFiles(fullPath, allFiles);
      } else if (item.endsWith('.md')) {
        const relativePath = path.relative(this.vaultPath, fullPath);
        allFiles.push(relativePath);
      }
    }

    return allFiles.sort();
  }
}

async function main() {
  try {
    const vaultPath = await config.getVaultPath();

    if (!vaultPath) {
      console.log('No vault configured. Run: npm start init');
      process.exit(1);
    }

    const obsidian = new ObsidianInterface(vaultPath);
    await obsidian.start();

  } catch (error) {
    console.log('Error:', error.message);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = ObsidianInterface;