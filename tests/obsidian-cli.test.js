const fs = require('fs').promises;
const path = require('path');
const os = require('os');
const ObsidianCLI = require('../src/obsidian-cli');

describe('ObsidianCLI', () => {
  let tempDir;
  let cli;

  beforeEach(async () => {
    tempDir = await fs.mkdtemp(path.join(os.tmpdir(), 'obsidian-cli-test-'));
    cli = new ObsidianCLI(tempDir);
  });

  afterEach(async () => {
    await fs.rm(tempDir, { recursive: true, force: true });
  });

  test('should get today\'s date in YYYY-MM-DD format', () => {
    const date = cli.getTodayDate();
    expect(date).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  test('should generate daily note filename', () => {
    const filename = cli.getDailyNoteFilename();
    expect(filename).toMatch(/^\d{4}-\d{2}-\d{2}\.md$/);
  });

  test('should generate daily note path', () => {
    const notePath = cli.getDailyNotePath();
    expect(notePath).toBe(path.join(tempDir, cli.getMonthFolder(), cli.getDailyNoteFilename()));
  });

  test('should find markdown files in vault', async () => {
    await fs.writeFile(path.join(tempDir, 'test1.md'), '# Test 1');
    await fs.writeFile(path.join(tempDir, 'test2.md'), '# Test 2');
    await fs.writeFile(path.join(tempDir, 'not-markdown.txt'), 'Not markdown');

    const files = await cli.getMarkdownFiles(tempDir);
    expect(files).toHaveLength(2);
    expect(files).toContain('test1.md');
    expect(files).toContain('test2.md');
  });

  test('should find markdown files in subdirectories', async () => {
    const subDir = path.join(tempDir, 'subfolder');
    await fs.mkdir(subDir);
    await fs.writeFile(path.join(subDir, 'nested.md'), '# Nested');
    await fs.writeFile(path.join(tempDir, 'root.md'), '# Root');

    const files = await cli.getMarkdownFiles(tempDir);
    expect(files).toHaveLength(2);
    expect(files).toContain('root.md');
    expect(files).toContain('subfolder/nested.md');
  });

  test('should return month folder in YYYY-MM format', () => {
    const monthFolder = cli.getMonthFolder();
    expect(monthFolder).toMatch(/^\d{4}-\d{2}$/);
  });

  test('should process template replacing date placeholder', () => {
    const template = '# {{date:YYYY-MM-DD}}\nSome content {{date:YYYY-MM-DD}}';
    const result = cli.processTemplate(template);
    // processTemplate uses local time, so build expected date the same way
    const now = new Date();
    const expectedDate = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
    expect(result).toBe(`# ${expectedDate}\nSome content ${expectedDate}`);
    expect(result).not.toContain('{{date:YYYY-MM-DD}}');
  });

  test('should find section index for existing sections', () => {
    const lines = ['# Title', '', '## Tasks', '- item', '', '## Ideas', '- idea'];
    expect(cli.findSectionIndex(lines, 'Tasks')).toBe(2);
    expect(cli.findSectionIndex(lines, 'Ideas')).toBe(5);
  });

  test('should return -1 for missing section', () => {
    const lines = ['# Title', '', '## Tasks'];
    expect(cli.findSectionIndex(lines, 'Questions')).toBe(-1);
  });

  test('should add content to the correct section', async () => {
    const notePath = cli.getDailyNotePath();
    const monthFolderPath = path.join(tempDir, cli.getMonthFolder());
    await fs.mkdir(monthFolderPath, { recursive: true });
    await fs.writeFile(notePath, '# Title\n\n## Tasks\n\n## Ideas\n');
    cli.currentFile = notePath;
    cli.currentContent = '# Title\n\n## Tasks\n\n## Ideas\n';

    await cli.addToSection('Tasks', '- [ ] New task');
    expect(cli.currentContent).toContain('- [ ] New task');

    const lines = cli.currentContent.split('\n');
    const tasksIdx = cli.findSectionIndex(lines, 'Tasks');
    const ideasIdx = cli.findSectionIndex(lines, 'Ideas');
    const taskLine = lines.findIndex(l => l.includes('New task'));
    expect(taskLine).toBeGreaterThan(tasksIdx);
    expect(taskLine).toBeLessThan(ideasIdx);
  });

  test('should append content in append mode', async () => {
    const notePath = path.join(tempDir, 'test-append.md');
    await fs.writeFile(notePath, '# Title\nLine 1');
    cli.currentFile = notePath;
    cli.currentContent = '# Title\nLine 1';

    await cli.addContent('New line', 'append');
    expect(cli.currentContent).toContain('Line 1\nNew line');
  });

  test('should prepend content in prepend mode', async () => {
    const notePath = path.join(tempDir, 'test-prepend.md');
    await fs.writeFile(notePath, '# Title\nLine 1');
    cli.currentFile = notePath;
    cli.currentContent = '# Title\nLine 1';

    await cli.addContent('Prepended', 'prepend');
    expect(cli.currentContent.startsWith('Prepended')).toBe(true);
  });

  test('should replace content in replace mode', async () => {
    const notePath = path.join(tempDir, 'test-replace.md');
    await fs.writeFile(notePath, '# Old content');
    cli.currentFile = notePath;
    cli.currentContent = '# Old content';

    await cli.addContent('# New content', 'replace');
    expect(cli.currentContent).toContain('# New content');
    expect(cli.currentContent).not.toContain('# Old content');
  });

  test('should return false when addContent has no current file', async () => {
    cli.currentFile = null;
    const result = await cli.addContent('content');
    expect(result).toBe(false);
  });

  test('should insert content at specific line', async () => {
    const notePath = path.join(tempDir, 'test-insert.md');
    await fs.writeFile(notePath, 'Line 0\nLine 1\nLine 2');
    cli.currentFile = notePath;
    cli.currentContent = 'Line 0\nLine 1\nLine 2';

    const result = await cli.insertContentAtLine('Inserted', 1);
    expect(result).toBe(true);
    const lines = cli.currentContent.split('\n');
    expect(lines[1]).toBe('Inserted');
    expect(lines[2]).toBe('Line 1');
  });

  test('should return false when insertContentAtLine has no current file', async () => {
    cli.currentFile = null;
    const result = await cli.insertContentAtLine('content', 0);
    expect(result).toBe(false);
  });

  test('should replace content at specific line', async () => {
    const notePath = path.join(tempDir, 'test-replace-line.md');
    await fs.writeFile(notePath, 'Line 1\nLine 2\nLine 3');
    cli.currentFile = notePath;
    cli.currentContent = 'Line 1\nLine 2\nLine 3';

    const result = await cli.replaceContentAtLine('Replaced', 2);
    expect(result).toBe(true);
    const lines = cli.currentContent.split('\n');
    expect(lines[1]).toBe('Replaced');
  });

  test('should return false when replaceContentAtLine has invalid line', async () => {
    const notePath = path.join(tempDir, 'test-replace-invalid.md');
    await fs.writeFile(notePath, 'Line 1');
    cli.currentFile = notePath;
    cli.currentContent = 'Line 1';

    const result = await cli.replaceContentAtLine('X', 99);
    expect(result).toBe(false);
  });

  test('should route [] prefix to Tasks section and log centrally', async () => {
    const notePath = cli.getDailyNotePath();
    const monthFolderPath = path.join(tempDir, cli.getMonthFolder());
    await fs.mkdir(monthFolderPath, { recursive: true });
    const template = '# Title\n\n## Tasks\n\n## Ideas\n';
    await fs.writeFile(notePath, template);
    cli.currentFile = notePath;
    cli.currentContent = template;

    const result = await cli.processContentInput('[] Buy groceries');
    expect(result).toBe(true);
    expect(cli.currentContent).toContain('- [ ] Buy groceries');

    const taskLogPath = await cli.getTaskLogPath();
    const taskLog = await fs.readFile(taskLogPath, 'utf-8');
    expect(taskLog).toContain('Buy groceries');
  });

  test('should route - prefix to Ideas section', async () => {
    const notePath = cli.getDailyNotePath();
    const monthFolderPath = path.join(tempDir, cli.getMonthFolder());
    await fs.mkdir(monthFolderPath, { recursive: true });
    const template = '# Title\n\n## Tasks\n\n## Ideas\n';
    await fs.writeFile(notePath, template);
    cli.currentFile = notePath;
    cli.currentContent = template;

    const result = await cli.processContentInput('- New idea');
    expect(result).toBe(true);
    expect(cli.currentContent).toContain('- New idea');
  });

  test('should route ? prefix to Questions section', async () => {
    const notePath = cli.getDailyNotePath();
    const monthFolderPath = path.join(tempDir, cli.getMonthFolder());
    await fs.mkdir(monthFolderPath, { recursive: true });
    const template = '# Title\n\n## Questions\n\n## Ideas\n';
    await fs.writeFile(notePath, template);
    cli.currentFile = notePath;
    cli.currentContent = template;

    const result = await cli.processContentInput('? Why does this work');
    expect(result).toBe(true);
    expect(cli.currentContent).toContain('- Why does this work');
  });

  test('should route ! prefix to Insights section', async () => {
    const notePath = cli.getDailyNotePath();
    const monthFolderPath = path.join(tempDir, cli.getMonthFolder());
    await fs.mkdir(monthFolderPath, { recursive: true });
    const template = '# Title\n\n##  Insights\n\n## Ideas\n';
    await fs.writeFile(notePath, template);
    cli.currentFile = notePath;
    cli.currentContent = template;

    const result = await cli.processContentInput('! Key learning');
    expect(result).toBe(true);
    expect(cli.currentContent).toContain('- Key learning');
  });

  test('should render tab bar with active tab highlighted', () => {
    const tabs = ['Daily Note', 'Tasks'];
    const result0 = cli.renderTabBar(tabs, 0);
    expect(result0).toContain('{inverse} Daily Note {/inverse}');
    expect(result0).toContain(' Tasks ');
    expect(result0).not.toContain('{inverse} Tasks {/inverse}');

    const result1 = cli.renderTabBar(tabs, 1);
    expect(result1).toContain(' Daily Note ');
    expect(result1).toContain('{inverse} Tasks {/inverse}');
  });

  test('should create month folder and daily note via openDailyNote', async () => {
    // Mock startInteractiveMode to prevent blessed UI from launching
    cli.startInteractiveMode = jest.fn().mockResolvedValue(undefined);

    await cli.openDailyNote();

    const monthFolderPath = path.join(tempDir, cli.getMonthFolder());
    const monthFolderExists = await fs.access(monthFolderPath).then(() => true).catch(() => false);
    expect(monthFolderExists).toBe(true);

    const notePath = cli.getDailyNotePath();
    const noteExists = await fs.access(notePath).then(() => true).catch(() => false);
    expect(noteExists).toBe(true);

    const content = await fs.readFile(notePath, 'utf-8');
    // Template uses local time for date, so build expected the same way
    const now = new Date();
    const localDate = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
    expect(content).toContain(`# ${localDate}`);
    expect(content).toContain('## Tasks');
    expect(content).toContain('## Ideas');
    expect(content).toContain('#daily #inbox');
  });
});