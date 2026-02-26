const fs = require('fs').promises;
const path = require('path');
const os = require('os');
const ObsidianCLI = require('../src/obsidian-cli');

describe('Task Logging', () => {
  let tempDir;
  let cli;

  beforeEach(async () => {
    tempDir = await fs.mkdtemp(path.join(os.tmpdir(), 'obsidian-cli-task-test-'));
    cli = new ObsidianCLI(tempDir);
    // Mock startInteractiveMode to prevent blessed UI from launching
    cli.startInteractiveMode = jest.fn().mockResolvedValue(undefined);
  });

  afterEach(async () => {
    await fs.rm(tempDir, { recursive: true, force: true });
  });

  test('should create task log file and log tasks', async () => {
    await cli.openDailyNote();

    await cli.processContentInput('[] Test task for logging');

    const taskLogPath = await cli.getTaskLogPath();
    const taskLogExists = await fs.access(taskLogPath).then(() => true).catch(() => false);
    expect(taskLogExists).toBe(true);

    const taskLogContent = await fs.readFile(taskLogPath, 'utf-8');
    const todayDate = cli.getTodayDate();
    expect(taskLogContent).toContain('# Task Log');
    expect(taskLogContent).toContain(`- [ ] Test task for logging *[[${todayDate}]]*`);
    expect(taskLogContent).toContain(`[[${todayDate}]]`);
  });

  test('should append new tasks to existing log', async () => {
    await cli.openDailyNote();

    await cli.processContentInput('[] First task');
    await cli.processContentInput('[] Second task');

    const taskLogPath = await cli.getTaskLogPath();
    const taskLogContent = await fs.readFile(taskLogPath, 'utf-8');

    expect(taskLogContent).toContain('- [ ] First task');
    expect(taskLogContent).toContain('- [ ] Second task');

    const taskLines = taskLogContent.split('\n').filter(line => line.trim().startsWith('- [ ]'));
    expect(taskLines.length).toBe(2);
  });

  test('should log ideas to central ideas file', async () => {
    await cli.openDailyNote();

    await cli.processContentInput('- Build a cool feature');

    const ideasLogPath = await cli.getIdeasLogPath();
    const ideasLogExists = await fs.access(ideasLogPath).then(() => true).catch(() => false);
    expect(ideasLogExists).toBe(true);

    const ideasLogContent = await fs.readFile(ideasLogPath, 'utf-8');
    expect(ideasLogContent).toContain('# Ideas Log');
    expect(ideasLogContent).toContain('Build a cool feature');
  });

  test('should log questions to central questions file', async () => {
    await cli.openDailyNote();

    await cli.processContentInput('? How does this work');

    const questionsLogPath = await cli.getQuestionsLogPath();
    const questionsLogExists = await fs.access(questionsLogPath).then(() => true).catch(() => false);
    expect(questionsLogExists).toBe(true);

    const questionsLogContent = await fs.readFile(questionsLogPath, 'utf-8');
    expect(questionsLogContent).toContain('# Questions Log');
    expect(questionsLogContent).toContain('How does this work');
  });

  test('should log insights to central insights file', async () => {
    await cli.openDailyNote();

    await cli.processContentInput('! Important realization');

    const insightsLogPath = await cli.getInsightsLogPath();
    const insightsLogExists = await fs.access(insightsLogPath).then(() => true).catch(() => false);
    expect(insightsLogExists).toBe(true);

    const insightsLogContent = await fs.readFile(insightsLogPath, 'utf-8');
    expect(insightsLogContent).toContain('# Insights Log');
    expect(insightsLogContent).toContain('Important realization');
  });

  test('should read task log entries', async () => {
    await cli.openDailyNote();

    await cli.processContentInput('[] Task one');
    await cli.processContentInput('[] Task two');

    const tasks = await cli.readTaskLog();
    expect(tasks).toHaveLength(2);
    expect(tasks[0].content).toContain('Task');
    expect(tasks[0].completed).toBe(false);
    expect(tasks[1].completed).toBe(false);
  });

  test('should return empty array when no task log exists', async () => {
    const tasks = await cli.readTaskLog();
    expect(tasks).toEqual([]);
  });

  test('should complete a task by marking it as done', async () => {
    await cli.openDailyNote();

    await cli.processContentInput('[] Task to complete');

    let tasks = await cli.readTaskLog();
    expect(tasks).toHaveLength(1);
    expect(tasks[0].completed).toBe(false);

    await cli.completeTask(0, tasks, true);

    tasks = await cli.readTaskLog();
    expect(tasks[0].completed).toBe(true);
  });
});
