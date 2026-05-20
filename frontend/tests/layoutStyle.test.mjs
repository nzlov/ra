import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import test from 'node:test';

const css = readFileSync(new URL('../public/style.css', import.meta.url), 'utf8');

function rule(selector) {
  const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const match = css.match(new RegExp(`${escaped}\\s*\\{([\\s\\S]*?)\\n\\}`, 'm'));
  assert.ok(match, `missing ${selector} rule`);
  return match[1];
}

test('launcher surface fills the transparent window', () => {
  const launcher = rule('.launcher');
  assert.match(launcher, /height:\s*100vh;/);
  assert.match(launcher, /padding:\s*0;/);
  assert.match(launcher, /place-items:\s*stretch;/);

  const surface = rule('.surface');
  assert.match(surface, /width:\s*100%;/);
  assert.match(surface, /height:\s*100vh;/);
  assert.match(surface, /grid-template-rows:\s*auto minmax\(0,\s*1fr\) auto;/);

  const results = rule('.results');
  assert.match(results, /max-height:\s*none;/);
  assert.match(results, /min-height:\s*0;/);
});
