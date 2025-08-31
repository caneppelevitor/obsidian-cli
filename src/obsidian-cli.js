const fs = require('fs').promises;
const path = require('path');
const readline = require('readline');
const chalk = require('chalk');
const blessed = require('blessed');
const config = require('./config');

class ObsidianCLI {
  constructor(vaultPath) {
    this.vaultPath = vaultPath;
    this.currentFile = null;
    this.currentContent = '';
    this.lastInsertedLine = null;
  }

  getTodayDate() {
    return new Date().toISOString().split('T')[0];
  }

  async getTaskLogPath() {
    const taskLogFile = await config.getTaskLogFile();
    return path.join(this.vaultPath, taskLogFile);
  }

  async logTaskToCentralFile(taskContent) {
    const taskLogPath = await this.getTaskLogPath();
    const sourceFile = this.currentFile ? path.basename(this.currentFile, '.md') : 'unknown';
    
    const logEntry = `- [ ] ${taskContent} *[[${sourceFile}]]*`;
    
    try {
      let existingContent = '';
      try {
        existingContent = await fs.readFile(taskLogPath, 'utf-8');
      } catch (error) {
        const header = `# Task Log\n\nCentralized log of all tasks created across daily notes.\n\n`;
        existingContent = header;
      }
      
      const lines = existingContent.split('\n');
      
      let insertIndex = -1;
      for (let i = 0; i < lines.length; i++) {
        if (lines[i].trim().startsWith('- [ ]') || lines[i].trim().startsWith('- [x]')) {
          insertIndex = i;
          break;
        }
      }
      
      if (insertIndex === -1) {
        lines.push('', logEntry);
      } else {
        lines.splice(insertIndex, 0, logEntry);
      }
      
      const updatedContent = lines.join('\n');
      await fs.writeFile(taskLogPath, updatedContent);
      
    } catch (error) {
      console.error('Error logging task to central file:', error.message);
    }
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
      await this.logTaskToCentralFile(content);
      return true;
    } else if (trimmed.startsWith('-')) {
      const content = trimmed.slice(1).trim();
      await this.addToSection('Ideas', `- ${content}`);
      return true;
    } else if (trimmed.startsWith('?')) {
      const content = trimmed.slice(1).trim();
      await this.addToSection('Questions', `- ${content}`);
      return true;
    } else if (trimmed.startsWith('!')) {
      const content = trimmed.slice(1).trim();
      await this.addToSection('Insights', `- ${content}`);
      return true;
    } else {
      return await this.addContent(trimmed);
    }
  }

  async addToSection(sectionName, content) {
    if (!this.currentFile) {
      return false;
    }

    const lines = this.currentContent.split('\n');
    const sectionIndex = this.findSectionIndex(lines, sectionName);
    
    if (sectionIndex === -1) {
      return await this.addContent(content);
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
    
    let actualInsertLine;
    if (!hasContent) {
      lines.splice(sectionIndex + 1, 0, content);
      actualInsertLine = sectionIndex + 1;
    } else {
      lines.splice(lastContentLine + 1, 0, content);
      actualInsertLine = lastContentLine + 1;
    }
    
    this.currentContent = lines.join('\n');
    await this.saveCurrentFileContent();
    
    this.lastInsertedLine = actualInsertLine + 1;
    
    return true;
  }

  findSectionIndex(lines, sectionName) {
    for (let i = 0; i < lines.length; i++) {
      if (lines[i].startsWith('## ') && lines[i].includes(sectionName)) {
        return i;
      }
    }
    return -1;
  }

  getDailyNoteFilename() {
    return `${this.getTodayDate()}.md`;
  }

  getDailyNotePath() {
    return path.join(this.vaultPath, this.getDailyNoteFilename());
  }

  async openDailyNote() {
    const dailyNotePath = this.getDailyNotePath();
    const dailyNoteFilename = this.getDailyNoteFilename();

    try {
      await fs.access(this.vaultPath);
    } catch (error) {
      await fs.mkdir(this.vaultPath, { recursive: true });
    }

    try {
      await fs.access(dailyNotePath);
      console.log(chalk.blue(`Opening existing daily note: ${dailyNoteFilename}`));
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
      console.log(chalk.green(`Created new daily note: ${dailyNoteFilename}`));
    }

    this.currentFile = dailyNotePath;
    await this.loadCurrentFileContent();
    await this.startInteractiveMode();
  }

  async loadCurrentFileContent() {
    if (this.currentFile) {
      this.currentContent = await fs.readFile(this.currentFile, 'utf-8');
    }
  }

  async saveCurrentFileContent() {
    if (this.currentFile) {
      const lines = this.currentContent.split('\n');
      const hasMetadata = lines.some(line => line.includes('updated_at:'));

      if (!hasMetadata && lines.length > 0) {
        const now = new Date();
        const metadata = [
          '---',
          `updated_at: ${now.toISOString()}`,
          '---'
        ];

        if (lines[0].startsWith('#')) {
          lines.splice(1, 0, ...metadata);
          this.currentContent = lines.join('\n');
        }
      } else if (hasMetadata) {
        const updatedLines = lines.map(line => {
          if (line.includes('updated_at:')) {
            return `updated_at: ${new Date().toISOString()}`;
          }
          return line;
        });
        this.currentContent = updatedLines.join('\n');
      }

      await fs.writeFile(this.currentFile, this.currentContent);
    }
  }

  displayFileContent() {
    if (!this.currentContent) {
      console.log(chalk.yellow('File is empty'));
      return;
    }

    console.log('\n' + chalk.gray('─'.repeat(80)));
    console.log(chalk.cyan(`File: ${path.basename(this.currentFile)}`));
    console.log(chalk.gray('─'.repeat(80)));

    const lines = this.currentContent.split('\n');
    lines.forEach((line, index) => {
      const lineNum = chalk.gray((index + 1).toString().padStart(3, ' '));
      console.log(`${lineNum} │ ${line}`);
    });

    console.log(chalk.gray('─'.repeat(80)) + '\n');
  }

  async listFiles() {
    try {
      const files = await this.getMarkdownFiles(this.vaultPath);

      if (files.length === 0) {
        console.log(chalk.yellow('No markdown files found in vault'));
        return;
      }

      console.log('\n' + chalk.cyan('Markdown files in vault:'));
      console.log(chalk.gray('─'.repeat(50)));

      for (let i = 0; i < files.length; i++) {
        const filePath = path.join(this.vaultPath, files[i]);
        const stats = await fs.stat(filePath);
        const modifiedDate = stats.mtime.toLocaleDateString();

        console.log(`${chalk.gray((i + 1).toString().padStart(2, ' '))}. ${files[i]} ${chalk.gray(`(${modifiedDate})`)}`);
      }
      console.log(chalk.gray('─'.repeat(50)) + '\n');

      return files;
    } catch (error) {
      console.error(chalk.red('Error listing files:'), error.message);
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

    return allFiles;
  }

  async viewFile(filename) {
    const filePath = path.join(this.vaultPath, filename);

    try {
      const content = await fs.readFile(filePath, 'utf-8');

      console.log('\n' + chalk.gray('═'.repeat(80)));
      console.log(chalk.cyan(`Viewing: ${filename}`));
      console.log(chalk.gray('═'.repeat(80)));

      const lines = content.split('\n');
      lines.forEach((line, index) => {
        const lineNum = chalk.gray((index + 1).toString().padStart(3, ' '));
        console.log(`${lineNum} │ ${line}`);
      });

      console.log(chalk.gray('═'.repeat(80)) + '\n');

    } catch (error) {
      console.error(chalk.red(`Error reading file ${filename}:`), error.message);
    }
  }

  async addContent(content, insertionMode = 'append') {
    if (!this.currentFile) {
      return false;
    }

    const lines = this.currentContent.split('\n');
    let insertLine;
    
    switch (insertionMode) {
      case 'append':
        this.currentContent += '\n' + content;
        insertLine = lines.length + 1;
        break;
      case 'prepend':
        this.currentContent = content + '\n' + this.currentContent;
        insertLine = 1;
        break;
      case 'replace':
        this.currentContent = content;
        insertLine = 1;
        break;
    }

    this.lastInsertedLine = insertLine;

    await this.saveCurrentFileContent();
    return true;
  }

  async insertContentAtLine(content, lineNumber) {
    if (!this.currentFile) {
      return false;
    }

    const lines = this.currentContent.split('\n');
    
    if (lineNumber >= 0 && lineNumber <= lines.length) {
      lines.splice(lineNumber, 0, content);
      this.currentContent = lines.join('\n');
      
      this.lastInsertedLine = lineNumber + 1;
      
      await this.saveCurrentFileContent();
      return true;
    }
    
    return false;
  }

  async replaceContentAtLine(content, lineNumber) {
    if (!this.currentFile) {
      return false;
    }

    const lines = this.currentContent.split('\n');
    
    if (lineNumber > 0 && lineNumber <= lines.length) {
      lines[lineNumber - 1] = content;
      this.currentContent = lines.join('\n');
      await this.saveCurrentFileContent();
      return true;
    }
    
    return false;
  }

  createCustomInput() {
    let inputBuffer = '';
    let cursorPos = 0;
    let isExiting = false;

    process.stdin.setRawMode(true);
    process.stdin.resume();
    process.stdin.setEncoding('utf8');

    const drawPrompt = () => {
      const prompt = chalk.green('> ');
      const beforeCursor = inputBuffer.slice(0, cursorPos);
      const atCursor = inputBuffer.slice(cursorPos, cursorPos + 1) || ' ';
      const afterCursor = inputBuffer.slice(cursorPos + 1);

      process.stdout.write('\r\x1b[K');
      process.stdout.write(prompt + beforeCursor + chalk.inverse(atCursor) + afterCursor);
    };

    const handleInput = async (key) => {
      if (isExiting) return;

      switch (key) {
        case '\u0003':
          console.log('\n' + chalk.yellow('Goodbye!'));
          process.stdin.setRawMode(false);
          process.exit(0);
          break;

        case '\r':
        case '\n':
          process.stdout.write('\n');
          if (inputBuffer.trim()) {
            await this.processInput(inputBuffer);
          }
          inputBuffer = '';
          cursorPos = 0;
          drawPrompt();
          break;

        case '\u007f':
          if (cursorPos > 0) {
            inputBuffer = inputBuffer.slice(0, cursorPos - 1) + inputBuffer.slice(cursorPos);
            cursorPos--;
            drawPrompt();
          }
          break;

        case '\u001b[D':
          if (cursorPos > 0) {
            cursorPos--;
            drawPrompt();
          }
          break;

        case '\u001b[C':
          if (cursorPos < inputBuffer.length) {
            cursorPos++;
            drawPrompt();
          }
          break;

        default:
          if (key >= ' ' && key <= '~') {
            inputBuffer = inputBuffer.slice(0, cursorPos) + key + inputBuffer.slice(cursorPos);
            cursorPos++;
            drawPrompt();
          }
          break;
      }
    };

    return { handleInput, drawPrompt, cleanup: () => {
      isExiting = true;
      process.stdin.setRawMode(false);
    }};
  }

  async processInput(input) {
    if (input.startsWith('/')) {
      const command = input.slice(1).toLowerCase().trim();

      switch (command) {
        case 'view':
          return false;

        case 'save':
          await this.saveCurrentFileContent();
          return false;

        case 'files':
          return false;

        case 'open':
          const files = await this.getMarkdownFiles(this.vaultPath);
          if (files && files.length > 0) {
            return false;
          }
          break;

        case 'daily':
          await this.openDailyNote();
          return true;

        case 'exit':
          process.exit(0);
          break;

        case 'help':
          return false;

        default:
          if (command.startsWith('open ')) {
            const target = command.slice(5).trim();
            const files = await this.getMarkdownFiles(this.vaultPath);

            let filename;
            const num = parseInt(target);

            if (!isNaN(num) && num > 0 && num <= files.length) {
              filename = files[num - 1];
            } else {
              filename = target;
            }

            if (files.includes(filename)) {
              this.currentFile = path.join(this.vaultPath, filename);
              await this.loadCurrentFileContent();
              return true;
            }
            return false;
          }
          return false;
      }
    } else {
      return await this.processContentInput(input);
    }
  }

  async startInteractiveMode() {
    return this.createClaudeStyleInterface();
  }

  createClaudeStyleInterface() {
    const screen = blessed.screen({
      smartCSR: true,
      title: 'Obsidian CLI',
      autoPadding: false,
      warnings: false
    });

    const notesDisplay = blessed.text({
      parent: screen,
      top: 1,
      left: 1,
      width: '100%-2',
      height: '80%',
      border: {
        type: 'line'
      },
      style: {
        fg: 'white'
      },
      content: '',
      scrollable: true,
      alwaysScroll: true,
      focusable: true,
      mouse: true,
      clickable: true,
      keyable: true,
      keys: true,
      tags: true,
      label: {
        text: ` ${path.basename(this.currentFile || 'No file')} `,
        side: 'left',
        style: {
          fg: 'cyan',
          bold: true
        }
      }
    });

    const statusLine = blessed.text({
      parent: screen,
      top: 0,
      left: 0,
      width: '100%',
      height: 1,
      content: ' Obsidian CLI - Press Tab to navigate | Ctrl+C to exit | Type to add content | Use /commands for actions',
      style: {
        fg: 'white',
        inverse: true
      }
    });

    const inputContainer = blessed.box({
      parent: screen,
      top: '85%',
      left: 1,
      width: '100%-2',
      height: '12%',
      border: {
        type: 'line'
      },
      style: {
        fg: 'cyan'
      }
    });

    const promptSymbol = blessed.text({
      parent: inputContainer,
      top: 1,
      left: 1,
      width: 2,
      height: 1,
      content: '>',
      style: {
        fg: 'cyan',
        bold: true
      }
    });

    const inputBox = blessed.textbox({
      parent: inputContainer,
      top: 1,
      left: 3,
      width: '100%-4',
      height: 1,
      style: {
        fg: 'cyan'
      },
      inputOnFocus: true,
      censor: false
    });

    const setPlaceholder = () => {
      if (!inputBox.value || inputBox.value.trim() === '') {
        inputBox.setValue('');
        setTimeout(() => {
          inputBox.screen.program.cup(inputBox.atop + inputBox.itop + 1, inputBox.aleft + inputBox.ileft + 1);
          screen.render();
        }, 10);
      }
    };

    const styleLineContent = (line) => {
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
    };

    const updateNotesDisplay = () => {
      if (this.currentContent) {
        const lines = this.currentContent.split('\n');
        const numberedLines = lines.map((line, index) => {
          const lineNum = (index + 1).toString().padStart(3, ' ');
          const styledLine = styleLineContent(line);
          return `${lineNum} │ ${styledLine}`;
        });
        notesDisplay.setContent(numberedLines.join('\n'));
        notesDisplay.setLabel(` ${path.basename(this.currentFile || 'No file')} `);
        
        if (this.lastInsertedLine) {
          const totalLines = lines.length;
          const displayHeight = notesDisplay.height - 2;
          
          const currentScrollTop = notesDisplay.getScroll();
          const currentScrollBottom = currentScrollTop + displayHeight;
          
          const insertedLineIndex = this.lastInsertedLine - 1;
          
          if (insertedLineIndex < currentScrollTop || insertedLineIndex >= currentScrollBottom) {
            const targetScrollTop = Math.max(0, insertedLineIndex - Math.floor(displayHeight / 2));
            notesDisplay.scrollTo(targetScrollTop);
          }
          
          this.lastInsertedLine = null;
        } else {
          notesDisplay.scrollTo(notesDisplay.getScrollHeight());
        }
      } else {
        notesDisplay.setContent('File is empty');
      }
      screen.render();
    };


    inputBox.on('submit', async (value) => {
      const shouldIgnoreInput = (input) => {
        return !input || !input.trim();
      };
      
      if (!shouldIgnoreInput(value)) {
        const success = await this.processInput(value);
        if (success !== false) {
          updateNotesDisplay();
        }
      }
      
      inputBox.clearValue();
      setPlaceholder();
      inputBox.focus();
      screen.render();
    });

    inputBox.key('C-c', () => {
      process.exit(0);
    });

    inputBox.key(['left'], () => {
      const currentValue = inputBox.value || '';
      const cursorPos = inputBox.screen.program.x - inputBox.aleft - inputBox.ileft;
      
      if (cursorPos > 0) {
        inputBox.screen.program.cup(
          inputBox.atop + inputBox.itop + 1, 
          inputBox.aleft + inputBox.ileft + cursorPos - 1
        );
      }
      screen.render();
    });

    inputBox.key(['right'], () => {
      const currentValue = inputBox.value || '';
      const cursorPos = inputBox.screen.program.x - inputBox.aleft - inputBox.ileft;
      
      if (cursorPos < currentValue.length) {
        inputBox.screen.program.cup(
          inputBox.atop + inputBox.itop + 1, 
          inputBox.aleft + inputBox.ileft + cursorPos + 1
        );
      }
      screen.render();
    });

    inputBox.key(['home'], () => {
      inputBox.screen.program.cup(
        inputBox.atop + inputBox.itop + 1, 
        inputBox.aleft + inputBox.ileft
      );
      screen.render();
    });

    inputBox.key(['end'], () => {
      const currentValue = inputBox.value || '';
      inputBox.screen.program.cup(
        inputBox.atop + inputBox.itop + 1, 
        inputBox.aleft + inputBox.ileft + currentValue.length
      );
      screen.render();
    });


    screen.key('tab', () => {
      if (inputBox.focused) {
        notesDisplay.focus();
      } else {
        inputBox.focus();
      }
      screen.render();
    });

    screen.key(['escape', 'q', 'C-c'], () => {
      process.exit(0);
    });

    inputBox.on('focus', () => {
      inputContainer.style.border.fg = 'green';
      notesDisplay.style.border.fg = 'white';
      setPlaceholder();
      screen.render();
    });

    notesDisplay.on('focus', () => {
      notesDisplay.style.border.fg = 'green';
      inputContainer.style.border.fg = 'white';
      screen.render();
    });

    process.on('exit', () => {
      process.stdout.write('\x1b[0m');
      process.stdout.write('\x1b[?25h');
    });

    inputBox.focus();
    setPlaceholder();

    updateNotesDisplay();
    screen.render();

    return screen;
  }

  async viewMode() {
    const files = await this.getMarkdownFiles(this.vaultPath);
    if (!files || files.length === 0) return;

    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout
    });

    console.log('\nAvailable files:');
    files.forEach((file, index) => {
      console.log(`${chalk.gray((index + 1).toString().padStart(2, ' '))}. ${file}`);
    });

    rl.question(chalk.yellow('Enter filename or number to view (or press Enter to cancel): '), async (input) => {
      if (input.trim()) {
        let filename;
        const num = parseInt(input);

        if (!isNaN(num) && num > 0 && num <= files.length) {
          filename = files[num - 1];
        } else {
          filename = input.trim();
        }

        if (files.includes(filename)) {
          await this.viewFile(filename);
        } else {
          console.log(chalk.red('File not found'));
        }
      }
      rl.close();
    });
  }
}

module.exports = ObsidianCLI;