#!/usr/bin/env node

const path = require('path')
const fs = require('fs-extra')
const { Octokit } = require('@octokit/core')
const coreVersion = require('./core-version.js')
const searchFiles = require('./search-files.js')
const GITHUB_TOKEN = process.env.GITHUB_TOKEN
const OWNER = 'flarehotspot'
const REPO = 'sdk'
const octokit = new Octokit({ auth: GITHUB_TOKEN })

const main = async () => {
  const CORE_VERSION = await coreVersion()
  const DEVKIT_DIR = path.join(__dirname, '../../../../output/devkit')

  async function isPreRelease () {
    const preKeywords = ['alpha', 'beta', 'rc', 'pre']
    const tag = CORE_VERSION.toLowerCase()
    for (const keyword of preKeywords) {
      if (tag.includes(keyword)) {
        return true
      }
    }
    return false
  }

  async function releaseNotes () {
    const notes = await fs.readFile(
      path.join(__dirname, '../../release-notes/', `${CORE_VERSION}.md`),
      'utf8'
    )
    return (
      notes +
          `
    ---
    **Download Instruction:**

    If you are using Windows, Mac or Linux on x86, select the \`amd64\` zip file.
    If you are using Mac on Apple silicon, select the \`arm64\` zip file.
    Otherwise, select the version that's compatible with your CPU.
          `
    )
  }

  const { data } = await octokit.request(
    'POST /repos/{owner}/{repo}/releases',
    {
      owner: OWNER,
      repo: REPO,
      tag_name: CORE_VERSION,
      name: CORE_VERSION,
      body: await releaseNotes(),
      draft: false,
      prerelease: await isPreRelease(),
      generate_release_notes: false,
      headers: {
        'X-GitHub-Api-Version': '2022-11-28'
      }
    }
  )

  async function deleteRelease () {
    await octokit.request(
      'DELETE /repos/{owner}/{repo}/releases/{release_id}',
      {
        owner: OWNER,
        repo: REPO,
        release_id: data.id,
        headers: {
          'X-GitHub-Api-Version': '2022-11-28'
        }
      }
    )
    console.log(`Deleted release: ${data.id}`)
  }

  async function uploadZipFile (filePath) {
    const fileData = await fs.readFile(filePath)
    await octokit.request(`POST ${data.upload_url}`, {
      owner: OWNER,
      repo: REPO,
      name: path.basename(filePath),
      release_id: data.id,
      data: fileData,
      headers: {
        'X-GitHub-Api-Version': '2022-11-28',
        'Content-Type': 'application/zip'
      }
    })
    console.log(`Success uploading file: ${filePath}`)
  }

  async function zipAndUploadDevkit () {
    const zipFiles = await searchFiles(
      DEVKIT_DIR,
      (_, entry) => path.extname(entry) === '.gz',
      (dir, entry) => path.join(dir, entry),
      { stopRecurse: true }
    )

    for (const zipPath of zipFiles) {
      await uploadZipFile(zipPath)
    }
  }

  try {
    await zipAndUploadDevkit()
  } catch (e) {
    console.log(e)
    await deleteRelease()
    process.exit(1)
  }
}

module.exports = main()
