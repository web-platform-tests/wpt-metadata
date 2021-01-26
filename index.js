'use strict';

const fetch = require('node-fetch');
const puppeteer = require('puppeteer');

const bugInfoCache = new Map();

async function getBugInfo(page, url) {
  let cachedInfo = bugInfoCache.get(url);
  if (cachedInfo) {
    return cachedInfo;
  }
  await page.goto(url);
  const infoHandle = await page.waitForFunction(() => {
    const hostname = location.hostname;
    if (hostname === 'bugs.chromium.org') {
      if (document.title.includes('Loading issue')) {
        return; // wait some more
      }
      return {
        url: document.URL,
        title: document.title,
        status: 'TODO'
      };
    } else if (hostname === 'bugs.webkit.org') {
      return {
        url: document.URL,
        title: document.getElementById('short_desc_nonedit_display').textContent,
        status: document.getElementById('static_bug_status')
            .textContent.split(/\s+/).filter(s=>s).join(' ')
      };
    } else {
      throw new Error(`Don't know how to scrape ${hostname}`);
    }
  });
  const info = await infoHandle.jsonValue();

  // Cache the info for both requested and final URL.
  bugInfoCache.set(url, info);
  bugInfoCache.set(info.url, info);
  return info;
}

async function main() {
  const metadataURL = 'https://wpt.fyi/api/metadata?product=chrome';
  const metadata = await (await fetch(metadataURL)).json();

  const browser = await puppeteer.launch();
  const page = await browser.newPage();

  const bugs = new Map();
  for (const [pattern, links] of Object.entries(metadata)) {
    if (!pattern.startsWith('/service-workers/')) {
      continue;
    }
    for (const link of links) {
      const info = await getBugInfo(page, link.url);
      const url = info.url;
      if (!bugs.has(url)) {
        bugs.set(url, [info, new Set()]);
      }
      const [_, patterns] = bugs.get(url);
      patterns.add(pattern);
    }
  }
  const sortedBugs = Array.from(bugs.keys()).sort();
  for (const url of sortedBugs) {
    const [info, patterns] = bugs.get(url);
    const {title, status} = info;
    const sortedPatterns = Array.from(patterns).sort();
    for (const [i, pattern] of Object.entries(sortedPatterns)) {
      if (i === "0") {
        console.log(`${url}\t${title}\t${status}\t${pattern}`);
      } else {
        console.log(`\t\t\t${pattern}`);
      }
    }
  }

  await browser.close();
}

if (require.main === module) {
  main().catch((error) => {
    console.error(error);
    process.exit(1);
  });
}
