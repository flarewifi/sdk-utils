#!/usr/bin/env node

const fs = require('fs-extra');
const path = require('path');

// Search directories containing package.json
/**
 * @param {string} searchPath path to search;
 * @param {function} filterFn async (dirPath, entry, stat)
 * @param {function} returnFn async (dirPath, entry, stat)
 * @param {object} opts {stopRecurse: bool}
 * @returns Array of returnFn results
 */
async function searchFiles(searchPath, filterFn, returnFn, opts) {
  opts = { stopRecurse: false, ...opts };

  filterFn = filterFn || (async () => true);

  returnFn =
    returnFn || (async (dirPath, entry, stat) => path.join(dirPath, entry));

  let results = [];

  for (const entry of await fs.readdir(searchPath)) {
    const stat = await fs.stat(path.join(searchPath, entry));
    const ok = await filterFn(searchPath, entry, stat);
    if (ok) {
      results.push(await returnFn(searchPath, entry, stat));
      if (stat.isDirectory()) {
        if (!opts.stopRecurse) {
          const dirPath = path.join(searchPath, entry);
          results = results.concat(
            await searchFiles(dirPath, filterFn, returnFn, opts)
          );
        }
      }
    } else {
      if (stat.isDirectory()) {
        const dirPath = path.join(searchPath, entry);
        results = results.concat(
          await searchFiles(dirPath, filterFn, returnFn, opts)
        );
      }
    }
  }

  return results;
}

module.exports = searchFiles;
