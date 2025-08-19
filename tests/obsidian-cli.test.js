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
    await fs.rmdir(tempDir, { recursive: true });
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
    expect(notePath).toBe(path.join(tempDir, cli.getDailyNoteFilename()));
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
});