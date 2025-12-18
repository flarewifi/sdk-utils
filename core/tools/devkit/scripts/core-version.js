#!/usr/bin/env node

const path = require('path');
const fs = require('fs-extra');
const CORE_PLUGIN_JSON = path.join(__dirname, '../../../plugin.json');

module.exports = async function () {
  const { version } = await fs.readJSON(CORE_PLUGIN_JSON);
  return version;
};
