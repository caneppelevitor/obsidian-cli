const fs = require('fs').promises;

/**
 * Generic central log function that collapses the 4 identical log methods.
 * @param {string} logPath - Path to the log file
 * @param {string} logEntry - Formatted log entry to insert
 * @param {string} headerTemplate - Header to use if file doesn't exist
 * @param {function} insertPredicate - Function(line) → boolean, finds first matching line to insert before
 */
async function logToCentralFile(logPath, logEntry, headerTemplate, insertPredicate) {
  try {
    let existingContent = '';
    try {
      existingContent = await fs.readFile(logPath, 'utf-8');
    } catch {
      existingContent = headerTemplate;
    }

    const lines = existingContent.split('\n');

    let insertIndex = -1;
    for (let i = 0; i < lines.length; i++) {
      if (insertPredicate(lines[i])) {
        insertIndex = i;
        break;
      }
    }

    if (insertIndex === -1) {
      lines.push('', logEntry);
    } else {
      lines.splice(insertIndex, 0, logEntry);
    }

    const updatedContent = lines.join('\n');
    await fs.writeFile(logPath, updatedContent);
  } catch (error) {
    console.error('Error logging to central file:', error.message);
  }
}

module.exports = { logToCentralFile };
