/**
 * Pure content manipulation functions.
 * All functions return values instead of mutating state.
 */

function processTemplate(template) {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');

  return template.replace(/\{\{date:YYYY-MM-DD\}\}/g, `${year}-${month}-${day}`);
}

function findSectionIndex(lines, sectionName) {
  for (let i = 0; i < lines.length; i++) {
    if (lines[i].startsWith('## ') && lines[i].includes(sectionName)) {
      return i;
    }
  }
  return -1;
}

/**
 * @returns {{ newContent: string, insertedLine: number }} | null (null means section not found)
 */
function addToSection(currentContent, sectionName, content) {
  const lines = currentContent.split('\n');
  const sectionIndex = findSectionIndex(lines, sectionName);

  if (sectionIndex === -1) {
    return null;
  }

  let insertIndex = sectionIndex + 1;
  let lastContentLine = sectionIndex;
  let hasContent = false;

  while (insertIndex < lines.length && !lines[insertIndex].startsWith('## ')) {
    if (lines[insertIndex].trim() !== '') {
      lastContentLine = insertIndex;
      hasContent = true;
    }
    insertIndex++;
  }

  let actualInsertLine;
  if (!hasContent) {
    lines.splice(sectionIndex + 1, 0, content);
    actualInsertLine = sectionIndex + 1;
  } else {
    lines.splice(lastContentLine + 1, 0, content);
    actualInsertLine = lastContentLine + 1;
  }

  return {
    newContent: lines.join('\n'),
    insertedLine: actualInsertLine + 1
  };
}

/**
 * @returns {{ newContent: string, insertedLine: number }}
 */
function addContent(currentContent, newContent, mode = 'append') {
  const lines = currentContent.split('\n');
  let resultContent;
  let insertLine;

  switch (mode) {
  case 'append':
    resultContent = currentContent + '\n' + newContent;
    insertLine = lines.length + 1;
    break;
  case 'prepend':
    resultContent = newContent + '\n' + currentContent;
    insertLine = 1;
    break;
  case 'replace':
    resultContent = newContent;
    insertLine = 1;
    break;
  }

  return { newContent: resultContent, insertedLine: insertLine };
}

/**
 * @returns {{ newContent: string, insertedLine: number }} | null
 */
function insertContentAtLine(currentContent, content, lineNumber) {
  const lines = currentContent.split('\n');

  if (lineNumber >= 0 && lineNumber <= lines.length) {
    lines.splice(lineNumber, 0, content);
    return { newContent: lines.join('\n'), insertedLine: lineNumber + 1 };
  }

  return null;
}

/**
 * @returns {{ newContent: string }} | null
 */
function replaceContentAtLine(currentContent, content, lineNumber) {
  const lines = currentContent.split('\n');

  if (lineNumber > 0 && lineNumber <= lines.length) {
    lines[lineNumber - 1] = content;
    return { newContent: lines.join('\n') };
  }

  return null;
}

/**
 * Parse input prefix to determine section routing.
 * @returns {{ section: string, formattedContent: string, logType: string, rawContent: string }} | null
 */
function parseContentInput(input) {
  const trimmed = input.trim();

  if (trimmed.startsWith('[]')) {
    const rawContent = trimmed.slice(2).trim();
    return { section: 'Tasks', formattedContent: `- [ ] ${rawContent}`, logType: 'task', rawContent };
  } else if (trimmed.startsWith('-')) {
    const rawContent = trimmed.slice(1).trim();
    return { section: 'Ideas', formattedContent: `- ${rawContent}`, logType: 'idea', rawContent };
  } else if (trimmed.startsWith('?')) {
    const rawContent = trimmed.slice(1).trim();
    return { section: 'Questions', formattedContent: `- ${rawContent}`, logType: 'question', rawContent };
  } else if (trimmed.startsWith('!')) {
    const rawContent = trimmed.slice(1).trim();
    return { section: 'Insights', formattedContent: `- ${rawContent}`, logType: 'insight', rawContent };
  }

  return null;
}

/**
 * Inject or update metadata (updated_at) in content.
 */
function injectMetadata(content) {
  const lines = content.split('\n');
  const hasMetadata = lines.some(line => line.includes('updated_at:'));

  if (!hasMetadata && lines.length > 0) {
    const now = new Date();
    const metadata = [
      '---',
      `updated_at: ${now.toISOString()}`,
      '---'
    ];

    if (lines[0].startsWith('#')) {
      lines.splice(1, 0, ...metadata);
      return lines.join('\n');
    }
  } else if (hasMetadata) {
    const updatedLines = lines.map(line => {
      if (line.includes('updated_at:')) {
        return `updated_at: ${new Date().toISOString()}`;
      }
      return line;
    });
    return updatedLines.join('\n');
  }

  return content;
}

module.exports = {
  processTemplate,
  findSectionIndex,
  addToSection,
  addContent,
  insertContentAtLine,
  replaceContentAtLine,
  parseContentInput,
  injectMetadata
};
