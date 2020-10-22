'use strict';

const fs = require('fs').promises;
const path = require('path');

const fetch = require('node-fetch');
const { JSDOM } = require('jsdom');
const klaw = require('klaw');
const yaml = require('js-yaml');

const bugInfoCache = new Map();

async function getBugInfo(url) {
  const cachedInfo = bugInfoCache.get(url);
  if (cachedInfo) {
    return cachedInfo;
  }
  const body = await (await fetch(url)).text();
  const dom = new JSDOM(body);
  const title = dom.window.document.title;
  const status = dom.window.document.getElementById('static_bug_status')
      .textContent.split(/\s+/).filter(s=>s).join(' ');
  const info = {title, status};
  bugInfoCache.set(url, info);
  return info;
}

const dotFilter = (item) => {
  const basename = path.basename(item);
  return basename === '.' || basename[0] !== '.';
};

async function main() {
  const files = await new Promise((resolve, reject) => {
    const files = [];
    klaw(__dirname, {filter: dotFilter})
        .on('data', (item) => {
          if (path.basename(item.path) === 'META.yml') {
            files.push(path.relative(__dirname, item.path));
          }
        })
        .on('error', reject)
        .on('end', () => {
          files.sort();
          resolve(files);
        });
  });

  for (const filename of files) {
    if (!filename.startsWith('css/css-grid/')) {
      continue;
    }
    const dirname = path.dirname(filename);
    const doc = yaml.safeLoad(await fs.readFile(filename, 'utf8'));
    for (const link of doc.links) {
      if (link.product !== 'safari' && link.product !== 'webkitgtk') {
        continue;
      }
      for (const result of link.results) {
        const {title, status} = await getBugInfo(link.url);
        if (result.test) {
          console.log(`${link.url}\t${title}\t${status}\t${path.join(dirname, result.test)}`);
        }
      }
    }
  }
};

if (require.main === module) {
  main().catch((error) => {
    console.error(error);
    process.exit(1);
  });
}
