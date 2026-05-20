import assert from 'node:assert/strict';
import test from 'node:test';

import {
  shouldCloseForEscape,
  shouldReturnToLauncherForEscape
} from '../src/launcherWindowBehavior.js';

test('closes launcher on Escape only when the query is empty', () => {
  assert.equal(shouldCloseForEscape('Escape', '', 'launcher'), true);
  assert.equal(shouldCloseForEscape('Escape', '   ', 'launcher'), true);
  assert.equal(shouldCloseForEscape('Escape', 'brave', 'launcher'), false);
  assert.equal(shouldCloseForEscape('Enter', '', 'launcher'), false);
  assert.equal(shouldCloseForEscape('Escape', '', 'capability'), false);
});

test('keeps Escape as back navigation outside launcher view', () => {
  assert.equal(shouldReturnToLauncherForEscape('Escape', 'capability'), true);
  assert.equal(shouldReturnToLauncherForEscape('Escape', 'launcher'), false);
  assert.equal(shouldReturnToLauncherForEscape('Enter', 'capability'), false);
});
