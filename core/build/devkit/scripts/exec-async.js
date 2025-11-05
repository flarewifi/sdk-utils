#!/usr/bin/env node

const { exec } = require('child_process');

module.exports = async (cmd, opts) => {
  console.log(`Executing: ${cmd}`);
  if (opts) console.log(`Exec options:`, opts);

  return await new Promise((resolve, reject) => {
    const proc = exec(cmd, opts, (err, stdout, _) => {
      if (err) {
        reject(err);
        return;
      }
      resolve(stdout);
    });

    proc.stdout.pipe(process.stdout);
    proc.stderr.pipe(process.stderr);
  });
};
