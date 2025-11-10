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
      logging: {
        tasks: {
          logFile: 'tasks-log.md',
          autoLog: true
        },
        ideas: {
          logFile: 'ideas-log.md',
          autoLog: true
        },
        questions: {
          logFile: 'questions-log.md',
          autoLog: true
        },
        insights: {
          logFile: 'insights-log.md',
          autoLog: true
        },
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
        showLineNumbers: true,
        eisenhowerTags: {
          '#do': 'red',
          '#delegate': 'yellow',
          '#schedule': 'blue',
          '#eliminate': 'gray'
        }
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
    
    if (!config.logging) {
      config.logging = {
        tasks: {
          logFile: 'tasks-log.md',
          autoLog: true
        },
        ideas: {
          logFile: 'ideas-log.md',
          autoLog: true
        },
        questions: {
          logFile: 'questions-log.md',
          autoLog: true
        },
        insights: {
          logFile: 'insights-log.md',
          autoLog: true
        },
        timestampFormat: 'simple'
      };
    } else {
      if (!config.logging.tasks?.logFile) {
        config.logging.tasks = { ...config.logging.tasks, logFile: 'tasks-log.md' };
      }
      if (!config.logging.ideas?.logFile) {
        config.logging.ideas = { ...config.logging.ideas, logFile: 'ideas-log.md' };
      }
      if (!config.logging.questions?.logFile) {
        config.logging.questions = { ...config.logging.questions, logFile: 'questions-log.md' };
      }
      if (!config.logging.insights?.logFile) {
        config.logging.insights = { ...config.logging.insights, logFile: 'insights-log.md' };
      }
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
      return config.logging?.tasks?.logFile || 'tasks-log.md';
    } catch (error) {
      return 'tasks-log.md';
    }
  }

  async getIdeasLogFile() {
    try {
      const config = await this.loadConfig();
      return config.logging?.ideas?.logFile || 'ideas-log.md';
    } catch (error) {
      return 'ideas-log.md';
    }
  }

  async getQuestionsLogFile() {
    try {
      const config = await this.loadConfig();
      return config.logging?.questions?.logFile || 'questions-log.md';
    } catch (error) {
      return 'questions-log.md';
    }
  }

  async getInsightsLogFile() {
    try {
      const config = await this.loadConfig();
      return config.logging?.insights?.logFile || 'insights-log.md';
    } catch (error) {
      return 'insights-log.md';
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

  async getEisenhowerTags() {
    try {
      const config = await this.loadConfig();
      return config.interface?.eisenhowerTags || {
        '#do': '196',        // Bright red (256-color palette) - urgent & important
        '#delegate': '214',  // Orange (256-color palette) - urgent & not important
        '#schedule': '33',   // Bright blue (256-color palette) - not urgent & important
        '#eliminate': '244'  // Gray (256-color palette) - not urgent & not important
      };
    } catch (error) {
      return {
        '#do': '196',
        '#delegate': '214',
        '#schedule': '33',
        '#eliminate': '244'
      };
    }
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
  getIdeasLogFile: () => config.getIdeasLogFile(),
  getQuestionsLogFile: () => config.getQuestionsLogFile(),
  getInsightsLogFile: () => config.getInsightsLogFile(),
  getFullConfig: () => config.getFullConfig(),
  getEisenhowerTags: () => config.getEisenhowerTags(),
  saveConfig: (configData) => config.saveConfig(configData)
};