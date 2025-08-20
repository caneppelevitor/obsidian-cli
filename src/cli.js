#!/usr/bin/env node

const { Command } = require('commander');
const ObsidianCLI = require('./obsidian-cli');
const config = require('./config');
const chalk = require('chalk');

const program = new Command();

program
  .name('obsidian-cli')
  .description('CLI tool for managing Obsidian vault with Claude Code-inspired interface')
  .version('1.0.0')
  .action(async (options) => {
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
  .description('Configure default vault path')
  .argument('[vault-path]', 'Path to set as default vault')
  .action(async (vaultPath) => {
    if (vaultPath) {
      await config.setVaultPath(vaultPath);
      console.log(chalk.green(`Default vault set to: ${vaultPath}`));
    } else {
      const currentVault = await config.getVaultPath();
      if (currentVault) {
        console.log(chalk.blue(`Current default vault: ${currentVault}`));
      } else {
        console.log(chalk.yellow('No default vault configured'));
      }
    }
  });

program
  .command('init')
  .description('Initialize configuration with your vault path')
  .action(async () => {
    const vaultPath = '/Users/vitorcaneppele/Documents/Notes do Papai/zettelkasten vault/raw notes';
    await config.setVaultPath(vaultPath);
    console.log(chalk.green(`Initialized with vault: ${vaultPath}`));
    console.log(chalk.blue('You can now use commands without specifying --vault option'));
  });

if (require.main === module) {
  program.parse();
}

module.exports = program;