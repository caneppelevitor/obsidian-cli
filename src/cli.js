#!/usr/bin/env node

const { Command } = require('commander');
const ObsidianCLI = require('./obsidian-cli');
const config = require('./config');
const chalk = require('chalk');
const path = require('path');
const fs = require('fs').promises;

const program = new Command();

program
  .name('obsidian-cli')
  .description('CLI tool for managing Obsidian vault with Claude Code-inspired interface')
  .version('1.0.0')
  .action(async () => {
    const config = require('./config');
    const vaultPath = await config.getVaultPath();
    if (!vaultPath) {
      console.error(chalk.red('No vault path configured. Run "obsidian init" first.'));
      process.exit(1);
    }

    const cli = new ObsidianCLI(vaultPath);
    await cli.openDailyNote();
  });

program
  .command('daily')
  .description('Open or create today\'s daily note (interactive mode)')
  .option('-v, --vault <path>', 'Path to Obsidian vault')
  .action(async (options) => {
    const vaultPath = options.vault || await config.getVaultPath();
    if (!vaultPath) {
      console.error(chalk.red('No vault path specified. Use --vault option or set default vault.'));
      process.exit(1);
    }

    const cli = new ObsidianCLI(vaultPath);
    await cli.openDailyNote();
  });

program
  .command('view')
  .description('View mode to browse files in vault')
  .option('-v, --vault <path>', 'Path to Obsidian vault')
  .action(async (options) => {
    const vaultPath = options.vault || await config.getVaultPath();
    if (!vaultPath) {
      console.error(chalk.red('No vault path specified. Use --vault option or set default vault.'));
      process.exit(1);
    }

    const cli = new ObsidianCLI(vaultPath);
    await cli.viewMode();
  });

program
  .command('files')
  .description('List all markdown files in vault')
  .option('-v, --vault <path>', 'Path to Obsidian vault')
  .action(async (options) => {
    const vaultPath = options.vault || await config.getVaultPath();
    if (!vaultPath) {
      console.error(chalk.red('No vault path specified. Use --vault option or set default vault.'));
      process.exit(1);
    }

    const cli = new ObsidianCLI(vaultPath);
    await cli.listFiles();
  });

program
  .command('config')
  .description('Show current configuration or create/edit config file')
  .option('--show', 'Show current configuration')
  .option('--edit', 'Open config file in default editor')
  .action(async (options) => {
    const os = require('os');
    const configPath = path.join(os.homedir(), '.obsidian-cli', 'config.yaml');
    
    if (options.edit) {
      const { spawn } = require('child_process');
      try {
        await fs.access(configPath);
      } catch (error) {
        const fullConfig = await config.getFullConfig();
        await config.saveConfig(fullConfig);
        console.log(chalk.green(`Created config file: ${configPath}`));
      }
      
      const editor = process.env.EDITOR || 'nano';
      spawn(editor, [configPath], { stdio: 'inherit' });
      return;
    }
    
    try {
      const fullConfig = await config.getFullConfig();
      const currentVault = await config.getVaultPath();
      
      if (options.show) {
        console.log(chalk.blue('Current Configuration:'));
        console.log(JSON.stringify(fullConfig, null, 2));
      } else {
        console.log(chalk.blue(`Config file: ${configPath}`));
        if (currentVault) {
          console.log(chalk.green(`Current vault: ${currentVault}`));
        } else {
          console.log(chalk.yellow('No vault configured'));
        }
        console.log(chalk.gray('Use --show to see full config, --edit to modify'));
      }
    } catch (error) {
      console.error(chalk.red(`Error reading config: ${error.message}`));
    }
  });

program
  .command('tasks')
  .description('View and manage tasks from the centralized task log')
  .option('-v, --vault <path>', 'Path to Obsidian vault')
  .option('--pending', 'Show only unchecked tasks')
  .option('--recent [days]', 'Show tasks from last N days (default: 7)', '7')
  .option('--complete <taskIndex>', 'Mark task as complete by index number')
  .action(async (options) => {
    const vaultPath = options.vault || await config.getVaultPath();
    if (!vaultPath) {
      console.error(chalk.red('No vault path specified. Use --vault option or set default vault.'));
      process.exit(1);
    }

    const cli = new ObsidianCLI(vaultPath);
    await cli.manageTasks(options);
  });

program
  .command('init')
  .description('Initialize Obsidian CLI with YAML configuration')
  .option('-v, --vault <path>', 'Path to your Obsidian vault')
  .option('--sample-config', 'Create a sample YAML config file in current directory')
  .action(async (options) => {
    const os = require('os');
    const homeConfigPath = path.join(os.homedir(), '.obsidian-cli', 'config.yaml');
    
    try {
      if (options.sampleConfig) {
        const sampleConfigPath = path.join(process.cwd(), 'obsidian-cli.config.yaml');
        const templatePath = path.join(__dirname, '..', 'obsidian-cli.config.yaml');
        
        await fs.copyFile(templatePath, sampleConfigPath);
        console.log(chalk.green(`✓ Sample config created: ${sampleConfigPath}`));
        console.log(chalk.blue('Edit this file and copy it to ~/.obsidian-cli/config.yaml'));
        return;
      }

      let vaultPath = options.vault;
      if (!vaultPath) {
        const inquirer = require('inquirer');
        const answers = await inquirer.prompt([
          {
            type: 'input',
            name: 'vaultPath',
            message: 'Enter the path to your Obsidian vault:',
            validate: async (input) => {
              try {
                const stats = await fs.stat(input);
                return stats.isDirectory() ? true : 'Path must be a directory';
              } catch (error) {
                return 'Path does not exist or is not accessible';
              }
            }
          }
        ]);
        vaultPath = answers.vaultPath;
      }

      const fullConfig = await config.getFullConfig();
      fullConfig.vault.defaultPath = vaultPath;
      
      await config.saveConfig(fullConfig);
      console.log(chalk.green(`✓ Configuration created: ${homeConfigPath}`));
      console.log(chalk.green(`✓ Vault path set to: ${vaultPath}`));
      console.log(chalk.blue('💡 Use "obsidian config --edit" to customize further'));
      console.log(chalk.green('🎉 Setup complete!'));
      
    } catch (error) {
      console.error(chalk.red(`Initialization failed: ${error.message}`));
      process.exit(1);
    }
  });

if (require.main === module) {
  program.parse();
}

module.exports = program;