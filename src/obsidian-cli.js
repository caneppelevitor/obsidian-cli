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

  async getIdeasLogPath() {
    const ideasLogFile = await config.getIdeasLogFile();
    return path.join(this.vaultPath, ideasLogFile);
  }

  async getQuestionsLogPath() {
    const questionsLogFile = await config.getQuestionsLogFile();
    return path.join(this.vaultPath, questionsLogFile);
  }

  async getInsightsLogPath() {
    const insightsLogFile = await config.getInsightsLogFile();
    return path.join(this.vaultPath, insightsLogFile);
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
        const header = '# Task Log\n\nCentralized log of all tasks created across daily notes.\n\n';
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

  async logIdeasToCentralFile(ideaContent) {
    const ideasLogPath = await this.getIdeasLogPath();
    const sourceFile = this.currentFile ? path.basename(this.currentFile, '.md') : 'unknown';

    const logEntry = `- ${ideaContent} *[[${sourceFile}]]*`;

    try {
      let existingContent = '';
      try {
        existingContent = await fs.readFile(ideasLogPath, 'utf-8');
      } catch (error) {
        const header = '# Ideas Log\n\nCentralized log of all ideas captured across daily notes.\n\n';
        existingContent = header;
      }

      const lines = existingContent.split('\n');

      let insertIndex = -1;
      for (let i = 0; i < lines.length; i++) {
        if (lines[i].trim().startsWith('- ') && !lines[i].includes('[ ]') && !lines[i].includes('[x]')) {
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
      await fs.writeFile(ideasLogPath, updatedContent);

    } catch (error) {
      console.error('Error logging idea to central file:', error.message);
    }
  }

  async logQuestionsToCentralFile(questionContent) {
    const questionsLogPath = await this.getQuestionsLogPath();
    const sourceFile = this.currentFile ? path.basename(this.currentFile, '.md') : 'unknown';

    const logEntry = `- ${questionContent} *[[${sourceFile}]]*`;

    try {
      let existingContent = '';
      try {
        existingContent = await fs.readFile(questionsLogPath, 'utf-8');
      } catch (error) {
        const header = '# Questions Log\n\nCentralized log of all questions captured across daily notes.\n\n';
        existingContent = header;
      }

      const lines = existingContent.split('\n');

      let insertIndex = -1;
      for (let i = 0; i < lines.length; i++) {
        if (lines[i].trim().startsWith('- ') && !lines[i].includes('[ ]') && !lines[i].includes('[x]')) {
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
      await fs.writeFile(questionsLogPath, updatedContent);

    } catch (error) {
      console.error('Error logging question to central file:', error.message);
    }
  }

  async logInsightsToCentralFile(insightContent) {
    const insightsLogPath = await this.getInsightsLogPath();
    const sourceFile = this.currentFile ? path.basename(this.currentFile, '.md') : 'unknown';

    const logEntry = `- ${insightContent} *[[${sourceFile}]]*`;

    try {
      let existingContent = '';
      try {
        existingContent = await fs.readFile(insightsLogPath, 'utf-8');
      } catch (error) {
        const header = '# Insights Log\n\nCentralized log of all insights captured across daily notes.\n\n';
        existingContent = header;
      }

      const lines = existingContent.split('\n');

      let insertIndex = -1;
      for (let i = 0; i < lines.length; i++) {
        if (lines[i].trim().startsWith('- ') && !lines[i].includes('[ ]') && !lines[i].includes('[x]')) {
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
      await fs.writeFile(insightsLogPath, updatedContent);

    } catch (error) {
      console.error('Error logging insight to central file:', error.message);
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
      await this.logIdeasToCentralFile(content);
      return true;
    } else if (trimmed.startsWith('?')) {
      const content = trimmed.slice(1).trim();
      await this.addToSection('Questions', `- ${content}`);
      await this.logQuestionsToCentralFile(content);
      return true;
    } else if (trimmed.startsWith('!')) {
      const content = trimmed.slice(1).trim();
      await this.addToSection('Insights', `- ${content}`);
      await this.logInsightsToCentralFile(content);
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

  async processInput(input, currentTab = 0, tasksDisplay = null) {
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

      case 'open': {
        const files = await this.getMarkdownFiles(this.vaultPath);
        if (files && files.length > 0) {
          return false;
        }
        break;
      }

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
    } else if (currentTab === 1 && /^\d+$/.test(input.trim())) {
      const taskIndex = parseInt(input.trim()) - 1;
      const tasks = await this.readTaskLog();
      const pendingTasks = tasks.filter(task => !task.completed);
      
      if (taskIndex >= 0 && taskIndex < pendingTasks.length) {
        const originalTaskIndex = tasks.findIndex(task => 
          task.content === pendingTasks[taskIndex].content && 
          !task.completed
        );
        
        await this.completeTask(originalTaskIndex, tasks, true);
        if (tasksDisplay) {
          await this.updateTasksDisplay(tasksDisplay);
        }
        return 'task_completed';
      } else {
        return 'invalid_task_number';
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

    let currentTab = 0;
    const tabs = ['Daily Note', 'Tasks'];

    const tabBar = blessed.text({
      parent: screen,
      top: 0,
      left: 0,
      width: '100%',
      height: 1,
      style: {
        bg: 'blue',
        fg: 'white'
      },
      content: this.renderTabBar(tabs, currentTab),
      tags: true
    });

    const notesDisplay = blessed.text({
      parent: screen,
      top: 2,
      left: 1,
      width: '100%-2',
      height: '80%-1',
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

    const tasksDisplay = blessed.text({
      parent: screen,
      top: 2,
      left: 1,
      width: '100%-2',
      height: '80%-1',
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
        text: ' Pending Tasks ',
        side: 'left',
        style: {
          fg: 'yellow',
          bold: true
        }
      },
      hidden: true
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

    const inputBox = blessed.box({
      parent: inputContainer,
      top: 1,
      left: 3,
      width: '100%-4',
      height: 1,
      style: {
        fg: 'cyan'
      },
      tags: true,
      focusable: true,
      keyable: true,
      input: false
    });

    let inputBuffer = '';
    let cursorPos = 0;

    const renderInput = () => {
      const beforeCursor = inputBuffer.slice(0, cursorPos);
      const atCursor = inputBuffer.slice(cursorPos, cursorPos + 1) || ' ';
      const afterCursor = inputBuffer.slice(cursorPos + 1);
      
      const inputDisplay = beforeCursor + `{inverse}${atCursor}{/inverse}` + afterCursor;
      inputBox.setContent(inputDisplay);
      screen.render();
    };

    const clearInput = () => {
      inputBuffer = '';
      cursorPos = 0;
      renderInput();
    };

    const insertChar = (char) => {
      inputBuffer = inputBuffer.slice(0, cursorPos) + char + inputBuffer.slice(cursorPos);
      cursorPos++;
      renderInput();
    };

    const deleteChar = () => {
      if (cursorPos > 0) {
        inputBuffer = inputBuffer.slice(0, cursorPos - 1) + inputBuffer.slice(cursorPos);
        cursorPos--;
        renderInput();
      }
    };

    const deleteCharForward = () => {
      if (cursorPos < inputBuffer.length) {
        inputBuffer = inputBuffer.slice(0, cursorPos) + inputBuffer.slice(cursorPos + 1);
        renderInput();
      }
    };

    const moveCursorLeft = () => {
      if (cursorPos > 0) {
        cursorPos--;
        renderInput();
      }
    };

    const moveCursorRight = () => {
      if (cursorPos < inputBuffer.length) {
        cursorPos++;
        renderInput();
      }
    };

    const moveCursorHome = () => {
      cursorPos = 0;
      renderInput();
    };

    const moveCursorEnd = () => {
      cursorPos = inputBuffer.length;
      renderInput();
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


    let lastProcessedTime = 0;
    screen.on('keypress', async (ch, key) => {
      if (!inputBox.focused) return;

      const now = Date.now();
      if (now - lastProcessedTime < 50) {
        return;
      }
      lastProcessedTime = now;


      if (key && key.name) {
        switch (key.name) {
        case 'return':
        case 'enter':
          if (inputBuffer.trim()) {
            const success = await this.processInput(inputBuffer, currentTab, tasksDisplay);
            if (success === 'task_completed') {
              clearInput();
              return;
            } else if (success === 'invalid_task_number') {
              clearInput();
              return;
            } else if (success !== false) {
              updateNotesDisplay();
            }
          }
          clearInput();
          return;

        case 'backspace':
          deleteChar();
          return;

        case 'delete':
          deleteCharForward();
          return;

        case 'left':
          moveCursorLeft();
          return;

        case 'right':
          moveCursorRight();
          return;

        case 'home':
          moveCursorHome();
          return;

        case 'end':
          moveCursorEnd();
          return;

        case 'escape':
          clearInput();
          return;

        case 'tab':
          return;
        }
      }

      if (ch && typeof ch === 'string' && ch.length === 1) {
        const charCode = ch.charCodeAt(0);
        if (charCode >= 32 && charCode <= 126) {
          insertChar(ch);
        }
      }
    });



    screen.key(['escape', 'C-c'], () => {
      process.exit(0);
    });

    screen.key('tab', async () => {
      currentTab = (currentTab + 1) % tabs.length;
      tabBar.setContent(this.renderTabBar(tabs, currentTab));
      await this.switchTab(currentTab, notesDisplay, tasksDisplay);
      screen.render();
    });

    screen.key('q', () => {
      if (!inputBox.focused) {
        process.exit(0);
      }
    });

    inputBox.on('focus', () => {
      inputContainer.style.border.fg = 'green';
      notesDisplay.style.border.fg = 'white';
      renderInput();
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
    renderInput();

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

  async manageTasks(options) {
    try {
      const tasks = await this.readTaskLog();
      
      if (options.complete) {
        return await this.completeTask(parseInt(options.complete) - 1, tasks);
      }
      
      let filteredTasks = tasks;
      
      if (options.pending) {
        filteredTasks = tasks.filter(task => !task.completed);
      }
      
      if (options.recent) {
        const days = parseInt(options.recent);
        filteredTasks = await this.filterRecentTasks(filteredTasks, days);
      }
      
      this.displayTasks(filteredTasks, options);
      
    } catch (error) {
      console.error(chalk.red(`Error managing tasks: ${error.message}`));
    }
  }

  async readTaskLog() {
    const taskLogPath = await this.getTaskLogPath();
    
    try {
      const content = await fs.readFile(taskLogPath, 'utf-8');
      const lines = content.split('\n');
      const tasks = [];
      
      for (let i = 0; i < lines.length; i++) {
        const line = lines[i].trim();
        if (line.startsWith('- [ ]') || line.startsWith('- [x]')) {
          const completed = line.startsWith('- [x]');
          const taskMatch = line.match(/^- \[.\] (.+?)( \*\[\[(.+?)\]\]\*)?$/);
          
          if (taskMatch) {
            tasks.push({
              index: tasks.length,
              content: taskMatch[1],
              completed,
              sourceFile: taskMatch[3] || 'unknown',
              lineNumber: i,
              originalLine: line
            });
          }
        }
      }
      
      return tasks;
    } catch (error) {
      if (error.code === 'ENOENT') {
        console.log(chalk.yellow('No task log found. Create some tasks first!'));
        return [];
      }
      throw error;
    }
  }

  async filterRecentTasks(tasks, days) {
    const cutoffDate = new Date();
    cutoffDate.setDate(cutoffDate.getDate() - days);
    
    const recentTasks = [];
    
    for (const task of tasks) {
      try {
        const sourceFilePath = path.join(this.vaultPath, task.sourceFile + '.md');
        const stats = await fs.stat(sourceFilePath);
        
        if (stats.mtime >= cutoffDate) {
          recentTasks.push(task);
        }
      } catch (error) {
        recentTasks.push(task);
      }
    }
    
    return recentTasks;
  }

  displayTasks(tasks, options) {
    if (tasks.length === 0) {
      console.log(chalk.yellow('No tasks found matching your criteria.'));
      return;
    }
    
    const title = options.pending ? 'Pending Tasks' : 
      options.recent ? `Tasks from last ${options.recent} days` : 
        'All Tasks';
    
    console.log(chalk.blue.bold(`\n${title}:`));
    console.log(chalk.gray('─'.repeat(50)));
    
    tasks.forEach((task, displayIndex) => {
      const status = task.completed ? chalk.green('✓') : chalk.red('○');
      const indexStr = chalk.gray(`[${displayIndex + 1}]`);
      const content = task.completed ? chalk.strikethrough(task.content) : task.content;
      const source = chalk.gray(`(${task.sourceFile})`);
      
      console.log(`${indexStr} ${status} ${content} ${source}`);
    });
    
    if (!options.pending && !options.complete) {
      const pendingCount = tasks.filter(t => !t.completed).length;
      const completedCount = tasks.filter(t => t.completed).length;
      
      console.log(chalk.gray('─'.repeat(50)));
      console.log(chalk.blue(`Total: ${tasks.length} | Pending: ${pendingCount} | Completed: ${completedCount}`));
    }
    
    console.log(chalk.gray('\nTip: Use --complete <number> to mark a task as done'));
  }

  async completeTask(taskIndex, tasks = null, silent = false) {
    if (!tasks) {
      tasks = await this.readTaskLog();
    }
    
    if (taskIndex < 0 || taskIndex >= tasks.length) {
      if (!silent) {
        console.error(chalk.red(`Invalid task index. Use a number between 1 and ${tasks.length}`));
      }
      return;
    }
    
    const task = tasks[taskIndex];
    
    if (task.completed) {
      if (!silent) {
        console.log(chalk.yellow('Task is already completed!'));
      }
      return;
    }
    
    try {
      const taskLogPath = await this.getTaskLogPath();
      const content = await fs.readFile(taskLogPath, 'utf-8');
      const lines = content.split('\n');
      
      lines[task.lineNumber] = lines[task.lineNumber].replace('- [ ]', '- [x]');
      
      await fs.writeFile(taskLogPath, lines.join('\n'));
      
      if (!silent) {
        console.log(chalk.green(`✓ Task completed: ${task.content}`));
      }
      
    } catch (error) {
      if (!silent) {
        console.error(chalk.red(`Error completing task: ${error.message}`));
      }
    }
  }

  renderTabBar(tabs, activeTab) {
    return tabs.map((tab, index) => {
      if (index === activeTab) {
        return `{inverse} ${tab} {/inverse}`;
      } else {
        return ` ${tab} `;
      }
    }).join('');
  }

  async switchTab(tabIndex, notesDisplay, tasksDisplay) {
    if (tabIndex === 0) {
      tasksDisplay.hide();
      notesDisplay.show();
    } else if (tabIndex === 1) {
      notesDisplay.hide();
      tasksDisplay.show();
      await this.updateTasksDisplay(tasksDisplay);
    }
  }

  async updateTasksDisplay(tasksDisplay) {
    try {
      const tasks = await this.readTaskLog();
      const pendingTasks = tasks.filter(task => !task.completed);
      
      if (pendingTasks.length === 0) {
        tasksDisplay.setContent('\nNo pending tasks!\n\nCreate some tasks in your daily note using [] prefix');
        return;
      }

      let content = '{cyan-fg}──────────────────────────────────────────────────{/cyan-fg}\n';
      
      pendingTasks.forEach((task, index) => {
        const taskNum = `{yellow-fg}[${index + 1}]{/yellow-fg}`;
        const taskIcon = '{red-fg}○{/red-fg}';
        const taskContent = `{white-fg}${task.content}{/white-fg}`;
        const taskSource = `{gray-fg}(${task.sourceFile}){/gray-fg}`;
        content += `${taskNum} ${taskIcon} ${taskContent} ${taskSource}\n`;
      });
      
      content += '\n{gray-fg}Tip: Type a number (1-' + pendingTasks.length + ') and press Enter to complete that task{/gray-fg}';
      
      tasksDisplay.setContent(content);
    } catch (error) {
      tasksDisplay.setContent(`Error loading tasks: ${error.message}`);
    }
  }
}

module.exports = ObsidianCLI;