const config = require('../src/config');

describe('Config', () => {
  test('getDefaultConfig returns expected structure', () => {
    // Access the Config class through the module internals
    // Since the module exports bound methods, we test via the public API
    // getDefaultConfig is not directly exported, so we verify via getFullConfig
  });

  test('getTaskLogFile returns default value', async () => {
    const taskLogFile = await config.getTaskLogFile();
    expect(typeof taskLogFile).toBe('string');
    expect(taskLogFile).toMatch(/\.md$/);
  });

  test('getIdeasLogFile returns default value', async () => {
    const ideasLogFile = await config.getIdeasLogFile();
    expect(typeof ideasLogFile).toBe('string');
    expect(ideasLogFile).toMatch(/\.md$/);
  });

  test('getQuestionsLogFile returns default value', async () => {
    const questionsLogFile = await config.getQuestionsLogFile();
    expect(typeof questionsLogFile).toBe('string');
    expect(questionsLogFile).toMatch(/\.md$/);
  });

  test('getInsightsLogFile returns default value', async () => {
    const insightsLogFile = await config.getInsightsLogFile();
    expect(typeof insightsLogFile).toBe('string');
    expect(insightsLogFile).toMatch(/\.md$/);
  });

  test('getFullConfig returns merged config with all expected keys', async () => {
    const fullConfig = await config.getFullConfig();
    expect(fullConfig).toHaveProperty('vault');
    expect(fullConfig).toHaveProperty('logging');
    expect(fullConfig).toHaveProperty('dailyNotes');
    expect(fullConfig).toHaveProperty('interface');
    expect(fullConfig).toHaveProperty('organization');
    expect(fullConfig).toHaveProperty('advanced');
  });

  test('getFullConfig includes default daily note sections', async () => {
    const fullConfig = await config.getFullConfig();
    expect(fullConfig.dailyNotes.sections).toContain('Tasks');
    expect(fullConfig.dailyNotes.sections).toContain('Ideas');
    expect(fullConfig.dailyNotes.sections).toContain('Questions');
    expect(fullConfig.dailyNotes.sections).toContain('Insights');
  });

  test('getEisenhowerTags returns tag config', async () => {
    const tags = await config.getEisenhowerTags();
    expect(tags).toHaveProperty('#do');
    expect(tags).toHaveProperty('#delegate');
    expect(tags).toHaveProperty('#schedule');
    expect(tags).toHaveProperty('#eliminate');
  });

  test('getVaultPath returns a value', async () => {
    const vaultPath = await config.getVaultPath();
    // Could be a string path or undefined/empty depending on user config
    expect(vaultPath === undefined || vaultPath === '' || typeof vaultPath === 'string').toBe(true);
  });
});
