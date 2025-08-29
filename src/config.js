const fs = require('fs').promises;
const path = require('path');
const os = require('os');

const CONFIG_DIR = path.join(os.homedir(), '.obsidian-cli');
const CONFIG_FILE = path.join(CONFIG_DIR, 'config.json');

class Config {
  async ensureConfigDir() {
    try {
      await fs.access(CONFIG_DIR);
    } catch (error) {
      await fs.mkdir(CONFIG_DIR, { recursive: true });
    }
  }

  async loadConfig() {
    try {
      await this.ensureConfigDir();
      const configData = await fs.readFile(CONFIG_FILE, 'utf-8');
      return JSON.parse(configData);
    } catch (error) {
      return {};
    }
  }

  async saveConfig(config) {
    await this.ensureConfigDir();
    await fs.writeFile(CONFIG_FILE, JSON.stringify(config, null, 2));
  }

  async setVaultPath(vaultPath) {
    const config = await this.loadConfig();

    try {
      const stats = await fs.stat(vaultPath);
      if (!stats.isDirectory()) {
        throw new Error('Vault path must be a directory');
      }
    } catch (error) {
      throw new Error(`Vault path does not exist or is not accessible: ${vaultPath}`);
    }

    config.defaultVault = vaultPath;
    if (!config.taskLogFile) {
      config.taskLogFile = 'tasks-log.md';
    }
    await this.saveConfig(config);
  }

  async setTaskLogFile(taskLogFile) {
    const config = await this.loadConfig();
    config.taskLogFile = taskLogFile;
    await this.saveConfig(config);
  }

  async getTaskLogFile() {
    try {
      const config = await this.loadConfig();
      return config.taskLogFile || 'tasks-log.md';
    } catch (error) {
      return 'tasks-log.md';
    }
  }

  async getVaultPath() {
    try {
      const config = await this.loadConfig();
      return config.defaultVault;
    } catch (error) {
      return null;
    }
  }
}

const config = new Config();

module.exports = {
  setVaultPath: (vaultPath) => config.setVaultPath(vaultPath),
  getVaultPath: () => config.getVaultPath(),
  setTaskLogFile: (taskLogFile) => config.setTaskLogFile(taskLogFile),
  getTaskLogFile: () => config.getTaskLogFile(),
  loadConfig: () => config.loadConfig(),
  saveConfig: (configData) => config.saveConfig(configData)
};