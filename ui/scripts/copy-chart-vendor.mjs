import { copyFileSync, mkdirSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const destDir = join(root, 'public', 'vendor')
const src = join(root, 'node_modules', 'chart.js', 'dist', 'chart.umd.min.js')
const dest = join(destDir, 'chart.min.js')

mkdirSync(destDir, { recursive: true })
copyFileSync(src, dest)
