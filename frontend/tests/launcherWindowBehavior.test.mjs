import assert from 'node:assert/strict';
import test from 'node:test';

import * as launcherBehavior from '../src/launcherWindowBehavior.js';

const {
  shouldHideForEscape,
  shouldReturnToLauncherForEscape
} = launcherBehavior;

test('hides launcher on Escape without checking the query', () => {
  assert.equal(shouldHideForEscape('Escape', '', 'launcher'), true);
  assert.equal(shouldHideForEscape('Escape', '   ', 'launcher'), true);
  assert.equal(shouldHideForEscape('Escape', 'brave', 'launcher'), true);
  assert.equal(shouldHideForEscape('Enter', '', 'launcher'), false);
  assert.equal(shouldHideForEscape('Escape', '', 'capability'), false);
});

test('keeps Escape as back navigation outside launcher view', () => {
  assert.equal(shouldReturnToLauncherForEscape('Escape', 'capability'), true);
  assert.equal(shouldReturnToLauncherForEscape('Escape', 'launcher'), false);
  assert.equal(shouldReturnToLauncherForEscape('Enter', 'capability'), false);
});

test('first window open starts on the launcher search state', () => {
  assert.equal(typeof launcherBehavior.launcherStateForOpen, 'function');
  assert.deepEqual(launcherBehavior.launcherStateForOpen(), {
    activeCapability: null,
    activeIndex: 0,
    query: '',
    view: 'launcher'
  });
});

test('reopening the window switches back to search and clears the query', () => {
  assert.equal(typeof launcherBehavior.launcherStateForOpen, 'function');
  const previousState = {
    activeCapability: {id: 'plugin:manager'},
    activeIndex: 2,
    query: 'brave',
    view: 'capability'
  };

  assert.deepEqual({...previousState, ...launcherBehavior.launcherStateForOpen()}, {
    activeCapability: null,
    activeIndex: 0,
    query: '',
    view: 'launcher'
  });
});
