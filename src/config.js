const fs = require('fs').promises;
const path = require('path');
const os = require('os');
const yaml = require('js-yaml');

const CONFIG_DIR = path.join(os.homedir(), '.obsidian-cli');
const CONFIG_FILE = path.join(CONFIG_DIR, 'config.yaml');

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
      const yamlData = await fs.readFile(CONFIG_FILE, 'utf-8');
      return yaml.load(yamlData) || {};
    } catch (error) {
      return this.getDefaultConfig();
    }
  }

  getDefaultConfig() {
    return {
      vault: {
        defaultPath: ''
      },
      tasks: {
        logFile: 'tasks-log.md',
        autoLog: true,
        timestampFormat: 'simple'
      },
      dailyNotes: {
        sections: [
          'Daily Log',
          'Tasks', 
          'Ideas',
          'Questions',
          'Insights',
          'Links to Expand'
        ],
        tags: ['#daily', '#inbox'],
        titleFormat: 'YYYY-MM-DD'
      },
      interface: {
        theme: {
          border: 'cyan',
          title: 'white', 
          content: 'white',
          input: 'yellow',
          highlight: 'green'
        },
        autoScroll: true,
        showLineNumbers: true
      },
      organization: {
        sectionPrefixes: {
          '[]': 'Tasks',
          '-': 'Ideas',
          '?': 'Questions', 
          '!': 'Insights'
        }
      },
      advanced: {
        backup: {
          enabled: false,
          directory: '.obsidian-cli-backups',
          maxBackups: 5
        },
        performance: {
          maxFileSize: 10,
          watchFiles: false
        }
      }
    };
  }

  async saveConfig(config) {
    await this.ensureConfigDir();
    const yamlContent = yaml.dump(config, {
      indent: 2,
      lineWidth: 120,
      noRefs: true
    });
    await fs.writeFile(CONFIG_FILE, yamlContent);
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

    if (!config.vault) {
      config.vault = {};
    }
    
    config.vault.defaultPath = vaultPath;
    
    if (!config.tasks) {
      config.tasks = {
        logFile: 'tasks-log.md',
        autoLog: true,
        timestampFormat: 'simple'
      };
    } else if (!config.tasks.logFile) {
      config.tasks.logFile = 'tasks-log.md';
    }
    
    await this.saveConfig(config);
  }

  async setTaskLogFile(taskLogFile) {
    const config = await this.loadConfig();
    
    if (!config.tasks) {
      config.tasks = {};
    }
    
    config.tasks.logFile = taskLogFile;
    await this.saveConfig(config);
  }

  async getTaskLogFile() {
    try {
      const config = await this.loadConfig();
      return config.tasks?.logFile || 'tasks-log.md';
    } catch (error) {
      return 'tasks-log.md';
    }
  }

  async getVaultPath() {
    try {
      const config = await this.loadConfig();
      return config.vault?.defaultPath;
    } catch (error) {
      return null;
    }
  }
  
  async getFullConfig() {
    const config = await this.loadConfig();
    const defaultConfig = this.getDefaultConfig();
    
    return this.deepMerge(defaultConfig, config);
  }
  
  deepMerge(target, source) {
    const result = { ...target };
    
    for (const key in source) {
      if (source[key] && typeof source[key] === 'object' && !Array.isArray(source[key])) {
        result[key] = this.deepMerge(result[key] || {}, source[key]);
      } else {
        result[key] = source[key];
      }
    }
    
    return result;
  }
}

const config = new Config();

module.exports = {
  getVaultPath: () => config.getVaultPath(),
  getTaskLogFile: () => config.getTaskLogFile(),
  getFullConfig: () => config.getFullConfig(),
  saveConfig: (configData) => config.saveConfig(configData)
};