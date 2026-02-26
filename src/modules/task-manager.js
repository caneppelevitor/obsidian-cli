const fs = require('fs').promises;
const path = require('path');
const chalk = require('chalk');

async function readTaskLog(taskLogPath) {
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

async function completeTask(taskLogPath, taskIndex, tasks, silent = false) {
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

async function filterRecentTasks(tasks, days, vaultPath) {
  const cutoffDate = new Date();
  cutoffDate.setDate(cutoffDate.getDate() - days);

  const recentTasks = [];

  for (const task of tasks) {
    try {
      const sourceFilePath = path.join(vaultPath, task.sourceFile + '.md');
      const stats = await fs.stat(sourceFilePath);

      if (stats.mtime >= cutoffDate) {
        recentTasks.push(task);
      }
    } catch {
      recentTasks.push(task);
    }
  }

  return recentTasks;
}

function displayTasks(tasks, options) {
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

module.exports = { readTaskLog, completeTask, filterRecentTasks, displayTasks };
