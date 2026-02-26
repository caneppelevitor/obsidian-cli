const fs = require('fs').promises;
const path = require('path');
const chalk = require('chalk');

async function readFileContent(filePath) {
  return await fs.readFile(filePath, 'utf-8');
}

async function writeFileContent(filePath, content) {
  await fs.writeFile(filePath, content);
}

async function getMarkdownFiles(dir, baseDir, allFiles = []) {
  const items = await fs.readdir(dir);

  for (const item of items) {
    const fullPath = path.join(dir, item);
    const stat = await fs.stat(fullPath);

    if (stat.isDirectory() && !item.startsWith('.')) {
      await getMarkdownFiles(fullPath, baseDir, allFiles);
    } else if (item.endsWith('.md')) {
      const relativePath = path.relative(baseDir, fullPath);
      allFiles.push(relativePath);
    }
  }

  return allFiles;
}

function displayFileContent(filePath, content) {
  if (!content) {
    console.log(chalk.yellow('File is empty'));
    return;
  }

  console.log('\n' + chalk.gray('─'.repeat(80)));
  console.log(chalk.cyan(`File: ${path.basename(filePath)}`));
  console.log(chalk.gray('─'.repeat(80)));

  const lines = content.split('\n');
  lines.forEach((line, index) => {
    const lineNum = chalk.gray((index + 1).toString().padStart(3, ' '));
    console.log(`${lineNum} │ ${line}`);
  });

  console.log(chalk.gray('─'.repeat(80)) + '\n');
}

async function listMarkdownFiles(vaultPath) {
  try {
    const files = await getMarkdownFiles(vaultPath, vaultPath);

    if (files.length === 0) {
      console.log(chalk.yellow('No markdown files found in vault'));
      return [];
    }

    console.log('\n' + chalk.cyan('Markdown files in vault:'));
    console.log(chalk.gray('─'.repeat(50)));

    for (let i = 0; i < files.length; i++) {
      const filePath = path.join(vaultPath, files[i]);
      const stats = await fs.stat(filePath);
      const modifiedDate = stats.mtime.toLocaleDateString();

      console.log(`${chalk.gray((i + 1).toString().padStart(2, ' '))}. ${files[i]} ${chalk.gray(`(${modifiedDate})`)}`);
    }
    console.log(chalk.gray('─'.repeat(50)) + '\n');

    return files;
  } catch (error) {
    console.error(chalk.red('Error listing files:'), error.message);
  }
}

async function viewFile(vaultPath, filename) {
  const filePath = path.join(vaultPath, filename);

  try {
    const content = await fs.readFile(filePath, 'utf-8');

    console.log('\n' + chalk.gray('═'.repeat(80)));
    console.log(chalk.cyan(`Viewing: ${filename}`));
    console.log(chalk.gray('═'.repeat(80)));

    const lines = content.split('\n');
    lines.forEach((line, index) => {
      const lineNum = chalk.gray((index + 1).toString().padStart(3, ' '));
      console.log(`${lineNum} │ ${line}`);
    });

    console.log(chalk.gray('═'.repeat(80)) + '\n');

  } catch (error) {
    console.error(chalk.red(`Error reading file ${filename}:`), error.message);
  }
}

module.exports = {
  readFileContent,
  writeFileContent,
  getMarkdownFiles,
  displayFileContent,
  listMarkdownFiles,
  viewFile
};
