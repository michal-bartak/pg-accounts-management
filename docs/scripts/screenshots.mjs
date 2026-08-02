// Screenshot placeholders for the docs.
//
// Every figure in the docs references a real .png path, so replacing a screenshot is just
// "overwrite the file and rebuild" — no markdown editing. This script stands in a generated
// placeholder for any referenced file that doesn't exist yet, so the build never fails on a
// shot you haven't taken.
//
//   node scripts/screenshots.mjs           generate placeholders for missing files
//   node scripts/screenshots.mjs status    list which files are real and which are still stand-ins
//
// A placeholder is recognised by regenerating it in memory and comparing bytes, so there is no
// state file to keep in sync: overwrite a placeholder with a real capture and it reports as real.
import { existsSync, readdirSync, readFileSync, mkdirSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import sharp from 'sharp';

const DOCS = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const CONTENT = join(DOCS, 'src/content/docs');
const ASSETS = join(DOCS, 'src/assets');

const WIDTH = 1200;
const HEIGHT = 750;
const THEME = {
  light: { bg: '#f4f6f8', frame: '#c3cad3', ink: '#5b6472', faint: '#8b95a3' },
  dark: { bg: '#191c21', frame: '#3b424c', ink: '#98a2b0', faint: '#6d7787' },
};

/** Every image path the docs reference, relative to src/assets. */
function referenced() {
  const found = new Set();
  const walk = (dir) => {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      const path = join(dir, entry.name);
      if (entry.isDirectory()) walk(path);
      else if (entry.name.endsWith('.md')) {
        const text = readFileSync(path, 'utf8');
        for (const m of text.matchAll(/\]\(\.\.[./]*assets\/([^)]+\.png)\)/g)) found.add(m[1]);
      }
    }
  };
  walk(CONTENT);
  return [...found].sort();
}

const escapeXml = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

function placeholderPng(assetPath) {
  const theme = assetPath.endsWith('-dark.png') ? THEME.dark : THEME.light;
  const label = assetPath.endsWith('-dark.png')
    ? 'dark theme'
    : assetPath.endsWith('-light.png')
      ? 'light theme'
      : 'either theme';
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${WIDTH}" height="${HEIGHT}">
  <rect width="${WIDTH}" height="${HEIGHT}" fill="${theme.bg}"/>
  <rect x="24" y="24" width="${WIDTH - 48}" height="${HEIGHT - 48}" fill="none"
        stroke="${theme.frame}" stroke-width="3" stroke-dasharray="12 9" rx="10"/>
  <text x="${WIDTH / 2}" y="${HEIGHT / 2 - 46}" text-anchor="middle" fill="${theme.faint}"
        font-family="Segoe UI, Helvetica, Arial, sans-serif" font-size="22" letter-spacing="3">SCREENSHOT PLACEHOLDER</text>
  <text x="${WIDTH / 2}" y="${HEIGHT / 2 + 12}" text-anchor="middle" fill="${theme.ink}"
        font-family="Consolas, DejaVu Sans Mono, monospace" font-size="30">${escapeXml(assetPath)}</text>
  <text x="${WIDTH / 2}" y="${HEIGHT / 2 + 62}" text-anchor="middle" fill="${theme.faint}"
        font-family="Segoe UI, Helvetica, Arial, sans-serif" font-size="21">overwrite this file with a ${label} capture</text>
</svg>`;
  return sharp(Buffer.from(svg)).png({ compressionLevel: 9 }).toBuffer();
}

const paths = referenced();
const mode = process.argv[2] ?? 'generate';

if (mode === 'status') {
  const rows = [];
  for (const p of paths) {
    const file = join(ASSETS, p);
    if (!existsSync(file)) rows.push(['MISSING', p]);
    else rows.push([Buffer.compare(readFileSync(file), await placeholderPng(p)) === 0 ? 'todo' : 'real', p]);
  }
  for (const [state, p] of rows) console.log(`${state.padEnd(8)}${p}`);
  const todo = rows.filter(([s]) => s !== 'real').length;
  console.log(`\n${rows.length - todo}/${rows.length} captured, ${todo} still placeholder`);
  process.exit(0);
}

let written = 0;
for (const p of paths) {
  const file = join(ASSETS, p);
  if (existsSync(file)) continue;
  mkdirSync(dirname(file), { recursive: true });
  writeFileSync(file, await placeholderPng(p));
  console.log(`placeholder  ${p}`);
  written++;
}
console.log(`${written} placeholder(s) written, ${paths.length - written} already present`);
