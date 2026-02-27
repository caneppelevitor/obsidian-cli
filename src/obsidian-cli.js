const fs = require('fs').promises;
const path = require('path');
const chalk = require('chalk');
const config = require('./config');
const { logToCentralFile } = require('./modules/logger');
const content = require('./modules/content');
const fileManager = require('./modules/file-manager');
const taskManager = require('./modules/task-manager');
const { renderTabBar: renderTabBarFn } = require('./modules/ui/styling');
const { createInterface, createCustomInput: createCustomInputFn, viewMode: viewModeFn } = require('./modules/ui/interface');

class ObsidianCLI {
  constructor(vaultPath) {
    this.vaultPath = vaultPath;
    this.currentFile = null;
    this.currentContent = '';
    this.lastInsertedLine = null;
    this.eisenhowerTags = null;
  }

  // ── Date & path helpers ───────────────────────────────────────────

  getTodayDate() {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const day = String(now.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  getMonthFolder() {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    return `${year}-${month}`;
  }

  getDailyNoteFilename() {
    return `${this.getTodayDate()}.md`;
  }

  getDailyNotePath() {
    const monthFolder = this.getMonthFolder();
    return path.join(this.vaultPath, monthFolder, this.getDailyNoteFilename());
  }

  // ── Config-dependent path helpers ─────────────────────────────────

  async loadEisenhowerTags() {
    if (!this.eisenhowerTags) {
      this.eisenhowerTags = await config.getEisenhowerTags();
    }
    return this.eisenhowerTags;
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

  // ── Logger wrappers ───────────────────────────────────────────────

  async logTaskToCentralFile(taskContent) {
    const logPath = await this.getTaskLogPath();
    const sourceFile = this.currentFile ? path.basename(this.currentFile, '.md') : 'unknown';
    const entry = `- [ ] ${taskContent} *[[${sourceFile}]]*`;
    const header = '# Task Log\n\nCentralized log of all tasks created across daily notes.\n\n';
    await logToCentralFile(logPath, entry, header, (line) =>
      line.trim().startsWith('- [ ]') || line.trim().startsWith('- [x]')
    );
  }

  async logIdeasToCentralFile(ideaContent) {
    const logPath = await this.getIdeasLogPath();
    const sourceFile = this.currentFile ? path.basename(this.currentFile, '.md') : 'unknown';
    const entry = `- ${ideaContent} *[[${sourceFile}]]*`;
    const header = '# Ideas Log\n\nCentralized log of all ideas captured across daily notes.\n\n';
    await logToCentralFile(logPath, entry, header, (line) =>
      line.trim().startsWith('- ') && !line.includes('[ ]') && !line.includes('[x]')
    );
  }

  async logQuestionsToCentralFile(questionContent) {
    const logPath = await this.getQuestionsLogPath();
    const sourceFile = this.currentFile ? path.basename(this.currentFile, '.md') : 'unknown';
    const entry = `- ${questionContent} *[[${sourceFile}]]*`;
    const header = '# Questions Log\n\nCentralized log of all questions captured across daily notes.\n\n';
    await logToCentralFile(logPath, entry, header, (line) =>
      line.trim().startsWith('- ') && !line.includes('[ ]') && !line.includes('[x]')
    );
  }

  async logInsightsToCentralFile(insightContent) {
    const logPath = await this.getInsightsLogPath();
    const sourceFile = this.currentFile ? path.basename(this.currentFile, '.md') : 'unknown';
    const entry = `- ${insightContent} *[[${sourceFile}]]*`;
    const header = '# Insights Log\n\nCentralized log of all insights captured across daily notes.\n\n';
    await logToCentralFile(logPath, entry, header, (line) =>
      line.trim().startsWith('- ') && !line.includes('[ ]') && !line.includes('[x]')
    );
  }

  // ── Content wrappers ──────────────────────────────────────────────

  processTemplate(template) {
    return content.processTemplate(template);
  }

  findSectionIndex(lines, sectionName) {
    return content.findSectionIndex(lines, sectionName);
  }

  async addToSection(sectionName, contentStr) {
    if (!this.currentFile) {
      return false;
    }

    const result = content.addToSection(this.currentContent, sectionName, contentStr);

    if (!result) {
      return await this.addContent(contentStr);
    }

    this.currentContent = result.newContent;
    await this.saveCurrentFileContent();
    this.lastInsertedLine = result.insertedLine;
    return true;
  }

  async addContent(contentStr, insertionMode = 'append') {
    if (!this.currentFile) {
      return false;
    }

    const result = content.addContent(this.currentContent, contentStr, insertionMode);
    this.currentContent = result.newContent;
    this.lastInsertedLine = result.insertedLine;
    await this.saveCurrentFileContent();
    return true;
  }

  async insertContentAtLine(contentStr, lineNumber) {
    if (!this.currentFile) {
      return false;
    }

    const result = content.insertContentAtLine(this.currentContent, contentStr, lineNumber);

    if (!result) {
      return false;
    }

    this.currentContent = result.newContent;
    this.lastInsertedLine = result.insertedLine;
    await this.saveCurrentFileContent();
    return true;
  }

  async replaceContentAtLine(contentStr, lineNumber) {
    if (!this.currentFile) {
      return false;
    }

    const result = content.replaceContentAtLine(this.currentContent, contentStr, lineNumber);

    if (!result) {
      return false;
    }

    this.currentContent = result.newContent;
    await this.saveCurrentFileContent();
    return true;
  }

  // ── Orchestration ─────────────────────────────────────────────────

  async processContentInput(input) {
    const parsed = content.parseContentInput(input);

    if (!parsed) {
      return await this.addContent(input.trim());
    }

    await this.addToSection(parsed.section, parsed.formattedContent);

    switch (parsed.logType) {
    case 'task': await this.logTaskToCentralFile(parsed.rawContent); break;
    case 'idea': await this.logIdeasToCentralFile(parsed.rawContent); break;
    case 'question': await this.logQuestionsToCentralFile(parsed.rawContent); break;
    case 'insight': await this.logInsightsToCentralFile(parsed.rawContent); break;
    }

    return true;
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

  async openDailyNote() {
    const dailyNotePath = this.getDailyNotePath();
    const dailyNoteFilename = this.getDailyNoteFilename();
    const monthFolder = this.getMonthFolder();
    const monthFolderPath = path.join(this.vaultPath, monthFolder);

    try {
      await fs.access(this.vaultPath);
    } catch {
      await fs.mkdir(this.vaultPath, { recursive: true });
    }

    try {
      await fs.access(monthFolderPath);
    } catch {
      await fs.mkdir(monthFolderPath, { recursive: true });
      console.log(chalk.gray(`Created month folder: ${monthFolder}`));
    }

    try {
      await fs.access(dailyNotePath);
      console.log(chalk.blue(`Opening existing daily note: ${dailyNoteFilename}`));
    } catch {
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
    await this.loadEisenhowerTags();
    console.log(chalk.gray(`Loaded Eisenhower tags: ${Object.keys(this.eisenhowerTags || {}).join(', ')}`));
    await this.startInteractiveMode();
  }

  // ── File I/O wrappers ─────────────────────────────────────────────

  async loadCurrentFileContent() {
    if (this.currentFile) {
      this.currentContent = await fileManager.readFileContent(this.currentFile);
    }
  }

  async saveCurrentFileContent() {
    if (this.currentFile) {
      this.currentContent = content.injectMetadata(this.currentContent);
      await fileManager.writeFileContent(this.currentFile, this.currentContent);
    }
  }

  displayFileContent() {
    fileManager.displayFileContent(this.currentFile, this.currentContent);
  }

  async listFiles() {
    return await fileManager.listMarkdownFiles(this.vaultPath);
  }

  async getMarkdownFiles(dir, allFiles = []) {
    return fileManager.getMarkdownFiles(dir, this.vaultPath, allFiles);
  }

  async viewFile(filename) {
    return fileManager.viewFile(this.vaultPath, filename);
  }

  // ── Task wrappers ─────────────────────────────────────────────────

  async readTaskLog() {
    const taskLogPath = await this.getTaskLogPath();
    return taskManager.readTaskLog(taskLogPath);
  }

  async completeTask(taskIndex, tasks = null, silent = false) {
    if (!tasks) {
      tasks = await this.readTaskLog();
    }
    const taskLogPath = await this.getTaskLogPath();
    return taskManager.completeTask(taskLogPath, taskIndex, tasks, silent);
  }

  async filterRecentTasks(tasks, days) {
    return taskManager.filterRecentTasks(tasks, days, this.vaultPath);
  }

  displayTasks(tasks, options) {
    return taskManager.displayTasks(tasks, options);
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

  // ── UI wrappers ───────────────────────────────────────────────────

  renderTabBar(tabs, activeTab) {
    return renderTabBarFn(tabs, activeTab);
  }

  createCustomInput() {
    return createCustomInputFn(this);
  }

  async startInteractiveMode() {
    await this.loadEisenhowerTags();
    return this.createClaudeStyleInterface();
  }

  createClaudeStyleInterface() {
    return createInterface(this);
  }

  async viewMode() {
    return viewModeFn(this);
  }

  async updateTasksDisplay(tasksDisplay) {
    try {
      const tasks = await this.readTaskLog();
      const pendingTasks = tasks.filter(task => !task.completed);
      const completedTasks = tasks.filter(task => task.completed);

      if (pendingTasks.length === 0) {
        tasksDisplay.setContent('\nNo pending tasks!\n\nCreate some tasks in your daily note using [] prefix');
        return;
      }

      let displayContent = ` {bold}{white-fg}${pendingTasks.length} pending{/white-fg}{/bold} | {green-fg}${completedTasks.length} completed{/green-fg} | {gray-fg}${tasks.length} total{/gray-fg}\n`;
      displayContent += '{cyan-fg}──────────────────────────────────────────────────{/cyan-fg}\n';

      const eisenhowerTagNames = ['#do', '#delegate', '#schedule', '#eliminate'];
      const groups = {};
      const untagged = [];

      pendingTasks.forEach((task, index) => {
        task._displayIndex = index;
        const matchedTag = eisenhowerTagNames.find(tag => task.content.includes(tag));
        if (matchedTag) {
          if (!groups[matchedTag]) groups[matchedTag] = [];
          groups[matchedTag].push(task);
        } else {
          untagged.push(task);
        }
      });

      const renderTask = (task) => {
        const taskNum = `{yellow-fg}[${task._displayIndex + 1}]{/yellow-fg}`;
        const taskIcon = '{red-fg}○{/red-fg}';

        let styledTaskContent = task.content;
        if (this.eisenhowerTags) {
          for (const [tag, color] of Object.entries(this.eisenhowerTags)) {
            if (task.content.includes(tag)) {
              const colorTag = `{/}{${color}-fg}{bold}${tag}{/bold}{/}{white-fg}`;
              styledTaskContent = styledTaskContent.replace(
                new RegExp(tag.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g'),
                colorTag
              );
            }
          }
        }

        const taskContent = `{white-fg}${styledTaskContent}{/white-fg}`;
        const taskSource = `{gray-fg}(${task.sourceFile}){/gray-fg}`;
        return `  ${taskNum} ${taskIcon} ${taskContent} ${taskSource}\n`;
      };

      for (const tag of eisenhowerTagNames) {
        if (groups[tag] && groups[tag].length > 0) {
          const color = (this.eisenhowerTags && this.eisenhowerTags[tag]) || 'white';
          displayContent += `\n{${color}-fg}{bold}${tag} (${groups[tag].length}){/bold}{/${color}-fg}\n`;
          for (const task of groups[tag]) {
            displayContent += renderTask(task);
          }
        }
      }

      if (untagged.length > 0) {
        displayContent += `\n{gray-fg}{bold}Untagged (${untagged.length}){/bold}{/gray-fg}\n`;
        for (const task of untagged) {
          displayContent += renderTask(task);
        }
      }

      displayContent += '\n{gray-fg}Tip: Type a number (1-' + pendingTasks.length + ') and press Enter to complete that task{/gray-fg}';

      tasksDisplay.setContent(displayContent);
    } catch (error) {
      tasksDisplay.setContent(`Error loading tasks: ${error.message}`);
    }
  }
}

module.exports = ObsidianCLI;
