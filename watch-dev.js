#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { exec } = require('child_process');

console.log('🔄 Watching for changes and auto-installing globally...');
console.log('Press Ctrl+C to stop\n');

let isInstalling = false;

function installGlobally() {
  if (isInstalling) return;
  
  isInstalling = true;
  console.log('📦 Installing CLI globally...');
  
  exec('npm link', (error, stdout, stderr) => {
    if (error) {
      console.error('❌ Installation failed:', error);
    } else {
      console.log('✅ CLI installed globally! Use "obsidian" command to test.');
    }
    isInstalling = false;
  });
}

// Watch src directory for changes
const srcDir = path.join(__dirname, 'src');
const packageJsonPath = path.join(__dirname, 'package.json');

// Initial installation
installGlobally();

// Watch src directory
if (fs.existsSync(srcDir)) {
  fs.watch(srcDir, { recursive: true }, (eventType, filename) => {
    if (filename && filename.endsWith('.js')) {
      console.log(`🔄 Detected change in ${filename}`);
      setTimeout(installGlobally, 100); // Debounce
    }
  });
}

// Watch package.json
fs.watch(packageJsonPath, (eventType) => {
  console.log('🔄 Detected change in package.json');
  setTimeout(installGlobally, 100); // Debounce
});

// Keep process alive
process.on('SIGINT', () => {
  console.log('\n👋 Stopping file watcher...');
  process.exit(0);
});