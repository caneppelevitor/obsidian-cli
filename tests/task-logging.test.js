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
    expect(taskLogContent).toContain('# Task Log');
    expect(taskLogContent).toContain('- [ ] Test task for logging *[[2025-08-29]]*');
    expect(taskLogContent).toContain('[[2025-08-29]]');
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
});