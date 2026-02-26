const blessed = require('blessed');
const path = require('path');
const readline = require('readline');
const chalk = require('chalk');
const { styleLineContent, renderTabBar } = require('./styling');

async function switchTab(cli, tabIndex, notesDisplay, tasksDisplay) {
  if (tabIndex === 0) {
    tasksDisplay.hide();
    notesDisplay.show();
  } else if (tabIndex === 1) {
    notesDisplay.hide();
    tasksDisplay.show();
    await cli.updateTasksDisplay(tasksDisplay);
  }
}

function createInterface(cli) {
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
    content: renderTabBar(tabs, currentTab),
    tags: true
  });

  const notesDisplay = blessed.text({
    parent: screen,
    top: 2,
    left: 1,
    width: '100%-2',
    height: '100%-6',
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
      text: ` ${path.basename(cli.currentFile || 'No file')} `,
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
    height: '100%-6',
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

  const statusBar = blessed.text({
    parent: screen,
    top: 0,
    left: 0,
    width: '100%',
    height: 1,
    content: '',
    tags: true,
    style: {
      fg: 'white',
      inverse: true
    }
  });

  blessed.line({
    parent: screen,
    bottom: 3,
    left: 0,
    width: '100%',
    orientation: 'horizontal',
    style: { fg: 'cyan' }
  });

  blessed.line({
    parent: screen,
    bottom: 1,
    left: 0,
    width: '100%',
    orientation: 'horizontal',
    style: { fg: 'cyan' }
  });

  const inputContainer = blessed.box({
    parent: screen,
    bottom: 2,
    left: 0,
    width: '100%',
    height: 1
  });

  blessed.text({
    parent: inputContainer,
    top: 0,
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
    top: 0,
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

  const updateNotesDisplay = () => {
    if (cli.currentContent) {
      const lines = cli.currentContent.split('\n');
      const numberedLines = lines.map((line, index) => {
        const lineNum = (index + 1).toString().padStart(3, ' ');
        const styledLine = styleLineContent(line, cli.eisenhowerTags);
        return `{gray-fg}${lineNum} │{/gray-fg} ${styledLine}`;
      });
      const displayHeight = notesDisplay.height - 2;
      const spareLines = displayHeight - numberedLines.length;
      if (spareLines >= 8) {
        const cheatSheet = [
          '',
          '{gray-fg}  Quick Reference:{/gray-fg}',
          '{gray-fg}    []  text   →  Tasks{/gray-fg}',
          '{gray-fg}    -   text   →  Ideas{/gray-fg}',
          '{gray-fg}    ?   text   →  Questions{/gray-fg}',
          '{gray-fg}    !   text   →  Insights{/gray-fg}',
          '{gray-fg}    /help      →  Show commands{/gray-fg}',
          '{gray-fg}    /save      →  Save file{/gray-fg}',
        ];
        const blankLines = spareLines - cheatSheet.length;
        for (let i = 0; i < blankLines; i++) {
          numberedLines.push('');
        }
        numberedLines.push(...cheatSheet);
      }

      notesDisplay.setContent(numberedLines.join('\n'));
      notesDisplay.setLabel(` ${path.basename(cli.currentFile || 'No file')} `);

      if (cli.lastInsertedLine) {
        const dh = notesDisplay.height - 2;

        const currentScrollTop = notesDisplay.getScroll();
        const currentScrollBottom = currentScrollTop + dh;

        const insertedLineIndex = cli.lastInsertedLine - 1;

        if (insertedLineIndex < currentScrollTop || insertedLineIndex >= currentScrollBottom) {
          const targetScrollTop = Math.max(0, insertedLineIndex - Math.floor(dh / 2));
          notesDisplay.scrollTo(targetScrollTop);
        }

        cli.lastInsertedLine = null;
      } else {
        notesDisplay.scrollTo(notesDisplay.getScrollHeight());
      }
    } else {
      notesDisplay.setContent('File is empty');
    }
    screen.render();
  };

  const updateStatusBar = async (tab) => {
    if (tab === 0) {
      const words = cli.currentContent ? cli.currentContent.split(/\s+/).filter(w => w.length > 0).length : 0;
      const sections = cli.currentContent ? (cli.currentContent.match(/^## /gm) || []).length : 0;
      const now = new Date();
      const timeStr = `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`;
      statusBar.setContent(` Daily Note | ${words} words | ${sections} sections | Last edit: ${timeStr}`);
    } else if (tab === 1) {
      try {
        const tasks = await cli.readTaskLog();
        const pending = tasks.filter(t => !t.completed).length;
        const completed = tasks.filter(t => t.completed).length;
        const total = tasks.length;
        statusBar.setContent(` Tasks | ${pending} pending | ${completed} completed | ${total} total`);
      } catch {
        statusBar.setContent(' Tasks');
      }
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
          const success = await cli.processInput(inputBuffer, currentTab, tasksDisplay);
          if (success === 'task_completed') {
            await updateStatusBar(currentTab);
            clearInput();
            return;
          } else if (success === 'invalid_task_number') {
            clearInput();
            return;
          } else if (success !== false) {
            updateNotesDisplay();
            await updateStatusBar(currentTab);
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
    tabBar.setContent(renderTabBar(tabs, currentTab));
    await switchTab(cli, currentTab, notesDisplay, tasksDisplay);
    await updateStatusBar(currentTab);
    screen.render();
  });

  screen.key('q', () => {
    if (!inputBox.focused) {
      process.exit(0);
    }
  });

  inputBox.on('focus', () => {
    notesDisplay.style.border.fg = 'white';
    renderInput();
  });

  notesDisplay.on('focus', () => {
    notesDisplay.style.border.fg = 'green';
    screen.render();
  });

  process.on('exit', () => {
    process.stdout.write('\x1b[0m');
    process.stdout.write('\x1b[?25h');
  });

  inputBox.focus();
  renderInput();

  updateNotesDisplay();
  updateStatusBar(currentTab);
  screen.render();

  return screen;
}

function createCustomInput(cli) {
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
        await cli.processInput(inputBuffer);
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

async function viewMode(cli) {
  const fileManager = require('../file-manager');
  const files = await fileManager.getMarkdownFiles(cli.vaultPath, cli.vaultPath);
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
        await fileManager.viewFile(cli.vaultPath, filename);
      } else {
        console.log(chalk.red('File not found'));
      }
    }
    rl.close();
  });
}

module.exports = { createInterface, createCustomInput, viewMode };
