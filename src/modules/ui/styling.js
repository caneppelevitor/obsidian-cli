/**
 * Pure rendering functions for blessed-tagged strings.
 */

function styleLineContent(line, eisenhowerTags) {
  let parentColor = '';
  let needsParentColor = false;

  if (line.match(/^##\s+/)) {
    parentColor = 'cyan';
  } else if (line.match(/^#\s+/)) {
    parentColor = 'magenta';
  } else if (line.match(/^\s*-\s+\[[ x]\]\s+/)) {
    parentColor = 'green';
    needsParentColor = true;
  } else if (line.match(/^\s*-\s+/)) {
    parentColor = 'yellow';
    needsParentColor = true;
  } else if (line.match(/^#\w+/)) {
    parentColor = 'blue';
  }

  let processedLine = line;
  if (eisenhowerTags && needsParentColor) {
    for (const [tag, color] of Object.entries(eisenhowerTags)) {
      if (line.includes(tag)) {
        const colorTag = `{/}{${color}-fg}{bold}${tag}{/bold}{/}{${parentColor}-fg}`;
        processedLine = processedLine.replace(new RegExp(tag.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g'), colorTag);
      }
    }
  } else if (eisenhowerTags) {
    for (const [tag, color] of Object.entries(eisenhowerTags)) {
      if (line.includes(tag)) {
        const colorTag = `{${color}-fg}{bold}${tag}{/bold}{/}`;
        processedLine = processedLine.replace(new RegExp(tag.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g'), colorTag);
      }
    }
  }

  if (line.match(/^##\s+/)) {
    return `{cyan-fg}{bold}${processedLine}{/bold}{/cyan-fg}`;
  }

  if (line.match(/^#\s+/)) {
    return `{magenta-fg}{bold}${processedLine}{/bold}{/magenta-fg}`;
  }

  if (line.match(/^\s*-\s+\[[ x]\]\s+/)) {
    return `{green-fg}${processedLine}{/green-fg}`;
  }

  if (line.match(/^\s*-\s+/)) {
    return `{yellow-fg}${processedLine}{/yellow-fg}`;
  }

  if (line.match(/^#\w+/)) {
    return `{blue-fg}${processedLine}{/blue-fg}`;
  }

  return processedLine;
}

function renderTabBar(tabs, activeTab) {
  return tabs.map((tab, index) => {
    if (index === activeTab) {
      return `{bold}{white-fg} ● ${tab} {/white-fg}{/bold}`;
    } else {
      return `{gray-fg} ${tab} {/gray-fg}`;
    }
  }).join('{gray-fg}|{/gray-fg}');
}

module.exports = { styleLineContent, renderTabBar };
