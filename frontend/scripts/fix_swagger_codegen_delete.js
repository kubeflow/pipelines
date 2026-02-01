const fs = require('fs');
const path = require('path');

const TARGET_DIRS = ['src/apis', 'src/apisv2beta1'];
const PATTERNS = [
  [/delete localVarUrlObj\.search;/g, 'localVarUrlObj.search = null;'],
  [/localVarUrlObj\.search = undefined;/g, 'localVarUrlObj.search = null;'],
];

function walk(dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      walk(fullPath);
    } else if (entry.isFile() && entry.name === 'api.ts') {
      const original = fs.readFileSync(fullPath, 'utf8');
      let updated = original;
      for (const [pattern, replacement] of PATTERNS) {
        updated = updated.replace(pattern, replacement);
      }
      if (updated !== original) {
        fs.writeFileSync(fullPath, updated);
      }
    }
  }
}

for (const dir of TARGET_DIRS) {
  const fullPath = path.join(__dirname, '..', dir);
  if (fs.existsSync(fullPath)) {
    walk(fullPath);
  }
}
