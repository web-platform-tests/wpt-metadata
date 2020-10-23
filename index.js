'use strict';

const fs = require('fs').promises;
const path = require('path');

const fetch = require('node-fetch');
const { JSDOM } = require('jsdom');

const bugInfoCache = new Map();

async function getBugInfo(url) {
  const cachedInfo = bugInfoCache.get(url);
  if (cachedInfo) {
    return cachedInfo;
  }
  const body = await (await fetch(url)).text();
  const dom = new JSDOM(body);
  const title = dom.window.document.getElementById('short_desc_nonedit_display').textContent;
  const status = dom.window.document.getElementById('static_bug_status')
      .textContent.split(/\s+/).filter(s=>s).join(' ');
  const info = {title, status};
  bugInfoCache.set(url, info);
  return info;
}

async function main() {
  const metadataURL = 'https://wpt.fyi/api/metadata?product=safari&product=webkitgtk';
  const metadata = await (await fetch(metadataURL)).json();

  const bugs = new Map();
  for (const [pattern, links] of Object.entries(metadata)) {
    if (!pattern.startsWith('/css/css-flexbox/')) {
      continue;
    }
    for (const link of links) {
      const url = new URL(link.url).toString();
      let patterns = bugs.get(url);
      if (!patterns) {
        patterns = new Set();
        bugs.set(url, patterns);
      }
      patterns.add(pattern);
    }
  }
  const sortedBugs = Array.from(bugs.keys()).sort();
  for (const url of sortedBugs) {
    const {title, status} = await getBugInfo(url);
    const sortedPatterns = Array.from(bugs.get(url)).sort();
    for (const [i, pattern] of Object.entries(sortedPatterns)) {
      if (i === "0") {
        console.log(`${url}\t${title}\t${status}\t${pattern}`);
      } else {
        console.log(`\t\t\t${pattern}`);
      }
    }
  }
}

if (require.main === module) {
  main().catch((error) => {
    console.error(error);
    process.exit(1);
  });
}
