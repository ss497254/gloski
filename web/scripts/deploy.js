import { createHash } from 'node:crypto'
import { createReadStream, existsSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { readdir } from 'fs/promises'
import Queue from 'queue'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = join(dirname(__filename), '..')
const API_KEY = process.env.NETLIFY_API_KEY
const SITE_ID = process.env.NETLIFY_SITE_ID

async function deploy() {
  const files = await listFilesWithHashes(join(__dirname, 'dist'))

  const deploy = await createDeploy(files)
  await uploadFiles('dist', files, deploy)
  // Wait for 5 seconds before publishing, netlify does some post processing
  await new Promise((res) => setTimeout(res, 5000))
  await publishDeploy(deploy)
}

deploy()

/* Utitlities functions */

/**
 * List all files in a directory and compute each SHA256
 */
async function listFilesWithHashes(dir) {
  if (!existsSync(dir)) {
    throw new Error("Missing build folder 'dist'", {
      cause: 'Please make sure to run build command before deployment',
    })
  }

  const files = (await readdir(dir, { recursive: true, withFileTypes: true }))
    .filter((entry) => entry.isFile())
    .map((entry) => join(entry.parentPath ?? entry.path, entry.name))

  console.log(`Creating hash for ${files.length} in ${dir}`)

  const queue = new Queue({
    concurrency: 5,
    results: [],
  })

  files.forEach((file) => {
    queue.push((cb) => {
      const input = createReadStream(file)

      const hash = createHash('sha1')
      hash.setEncoding('hex')

      input.on('end', () => {
        hash.end()
        cb(null, file.replace(dir, '').replace(/\\/g, '/'), hash.read())
      })

      input.on('error', (err) => {
        hash.end()
        cb(err)
      })

      input.pipe(hash)
    })
  })

  return new Promise((resolve, reject) => {
    queue.start((err, results) => {
      if (err) {
        console.error('❌  Hash Generation failed\n', err)
        reject(err)
      } else {
        const filesHashes = results.reduce((res, [file, hash]) => {
          res[file] = hash
          return res
        }, {})
        console.log(`✅  Hash Generated for ${files.length} in ${dir}`)
        resolve(filesHashes)
      }
    })
  })
}

/**
 * Creates a new deployment on Netlify
 */
async function createDeploy(files) {
  try {
    console.log(`Creating deploy`)
    const result = await fetch(`https://api.netlify.com/api/v1/sites/${SITE_ID}/deploys`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + API_KEY,
      },
      body: JSON.stringify({ files }),
    })

    const deploy = await result.json()
    if (!result.ok) {
      throw new Error('Deploy failed')
    } else if (deploy.required.length) {
      console.log(`✅  Created deploy #${deploy.id} (${deploy.deploy_ssl_url}). ${deploy.required.length} new files.`)
      return deploy
    } else {
      console.error('No required files to deploy')
      process.exit(0)
    }
  } catch (e) {
    console.error('❌  Cannot create deploy\n', e)
    process.exit(1)
  }
}

/**
 * Cancel the deploy
 */
async function cancelDeploy(deploy) {
  try {
    const res = await fetch(`https://api.netlify.com/api/v1/deploys/${deploy.id}/cancel`, {
      method: 'POST',
      headers: {
        Authorization: 'Bearer ' + API_KEY,
      },
    })

    console.log(`Cancelled deploy #${deploy.id} (${deploy.ssl_url}).`, await res.text())
  } catch {
    console.warn(`Cannot cancel deploy`)
  }
}

/**
 * Publish the deploy
 */
async function publishDeploy(deploy) {
  try {
    const res = await fetch(`https://api.netlify.com/api/v1/sites/${SITE_ID}/deploys/${deploy.id}/restore`, {
      method: 'POST',
      headers: {
        Authorization: 'Bearer ' + API_KEY,
      },
    })

    if (res.ok) {
      console.log(`✅  Published deploy #${deploy.id} (${deploy.ssl_url}).`)
    } else {
      console.error(await res.text())
    }
  } catch {
    console.warn('❌  Cannot publish deploy')
  }
}

/**
 * Upload new files to Netlify
 */
async function uploadFiles(dir, files, deploy) {
  if (!deploy.required.length) {
    return
  }

  console.log(`Uploading ${deploy.required.length} from ${dir}`)
  const fileByHash = {}
  Object.entries(files).forEach(([file, hash]) => (fileByHash[hash] = file))

  const queue = new Queue({
    concurrency: 5,
    results: [],
  })

  deploy.required.forEach((hash) => {
    queue.push((cb) => {
      const file = fileByHash[hash]

      fetch(`https://api.netlify.com/api/v1/deploys/${deploy.id}/files/${encodeURIComponent(file)}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/octet-stream',
          Authorization: 'Bearer ' + API_KEY,
        },
        body: createReadStream(join(dir, file)),
        duplex: 'half',
      })
        .then(() => cb(null))
        .catch((err) => cb(err))
    })
  })

  try {
    await new Promise((resolve, reject) => {
      queue.start((err) => {
        if (err) {
          console.error('❌  Upload failed\n', err)
          reject(err)
        } else {
          console.log('✅  Upload completed')
          resolve()
        }
      })
    })
  } catch {
    console.error('❌  Cannot upload files')
    await cancelDeploy(deploy)
    process.exit(1)
  }
}
