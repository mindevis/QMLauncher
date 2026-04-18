#!/usr/bin/env node
/**
 * Apply icon to Windows exe. Requires: npm install rcedit (in frontend/), Wine (on Linux).
 * Usage: from QMLauncher: node scripts/apply-icon.mjs [exe] [ico]
 */
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
import { pathToFileURL } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = join(__dirname, '..');
const exe = process.argv[2] || join(root, 'build/QMLauncher-windows-amd64.exe');
const ico = process.argv[3] || join(root, 'assets/icon.ico');

const rceditPath = join(root, 'frontend/node_modules/rcedit/lib/index.js');
const { rcedit } = await import(pathToFileURL(rceditPath).href);

try {
  await rcedit(exe, { icon: ico });
  console.log('✓ Icon applied');
} catch (err) {
  console.error('Failed:', err.message);
  process.exit(1);
}
