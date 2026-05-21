import assert from 'node:assert/strict';
import test from 'node:test';

import {createSearchScheduler} from '../src/searchScheduler.js';

class FakeTimers {
  #now = 0;
  #nextID = 1;
  #timers = new Map();

  setTimeout(callback, delay) {
    const id = this.#nextID;
    this.#nextID += 1;
    this.#timers.set(id, {callback, time: this.#now + delay});
    return id;
  }

  clearTimeout(id) {
    this.#timers.delete(id);
  }

  async tick(ms) {
    this.#now += ms;
    const due = Array.from(this.#timers.entries())
      .filter(([, timer]) => timer.time <= this.#now)
      .sort(([, a], [, b]) => a.time - b.time);
    for (const [id, timer] of due) {
      if (!this.#timers.delete(id)) {
        continue;
      }
      timer.callback();
      await Promise.resolve();
    }
  }
}

test('debounces rapid scheduled searches', async () => {
  const timers = new FakeTimers();
  const calls = [];
  const applied = [];
  const scheduler = createSearchScheduler({
    delay: 100,
    setTimer: timers.setTimeout.bind(timers),
    clearTimer: timers.clearTimeout.bind(timers),
    search: async (query) => {
      calls.push(query);
      return [`result:${query}`];
    },
    onResults: (results, query) => {
      applied.push({query, results});
    },
    onError: () => assert.fail('unexpected search error')
  });

  scheduler.schedule('f');
  scheduler.schedule('fi');
  scheduler.schedule('fir');

  await timers.tick(99);
  assert.deepEqual(calls, []);
  assert.deepEqual(applied, []);

  await timers.tick(1);
  assert.deepEqual(calls, ['fir']);
  assert.deepEqual(applied, [{query: 'fir', results: ['result:fir']}]);
});

test('ignores stale responses after a newer query is scheduled', async () => {
  const timers = new FakeTimers();
  const calls = [];
  const applied = [];
  const resolvers = new Map();
  const scheduler = createSearchScheduler({
    delay: 100,
    setTimer: timers.setTimeout.bind(timers),
    clearTimer: timers.clearTimeout.bind(timers),
    search: (query) => {
      calls.push(query);
      return new Promise((resolve) => {
        resolvers.set(query, resolve);
      });
    },
    onResults: (results, query) => {
      applied.push({query, results});
    },
    onError: () => assert.fail('unexpected search error')
  });

  const first = scheduler.searchNow('f');
  scheduler.schedule('fi');
  resolvers.get('f')(['old']);
  await first;
  assert.deepEqual(applied, []);

  await timers.tick(100);
  resolvers.get('fi')(['new']);
  await Promise.resolve();

  assert.deepEqual(calls, ['f', 'fi']);
  assert.deepEqual(applied, [{query: 'fi', results: ['new']}]);
});

test('cancels an in-flight search when searchNow supersedes it', async () => {
  const cancelled = [];
  const applied = [];
  const resolvers = new Map();
  const scheduler = createSearchScheduler({
    delay: 100,
    search: (query) => {
      const promise = new Promise((resolve) => {
        resolvers.set(query, resolve);
      });
      promise.cancel = () => {
        cancelled.push(query);
      };
      return promise;
    },
    onResults: (results, query) => {
      applied.push({query, results});
    },
    onError: () => assert.fail('unexpected search error')
  });

  const first = scheduler.searchNow('f');
  const second = scheduler.searchNow('fi');

  assert.deepEqual(cancelled, ['f']);

  resolvers.get('f')(['old']);
  await first;
  assert.deepEqual(applied, []);

  resolvers.get('fi')(['new']);
  await second;
  assert.deepEqual(applied, [{query: 'fi', results: ['new']}]);
});

test('cancels an in-flight search when a scheduled search starts', async () => {
  const timers = new FakeTimers();
  const cancelled = [];
  const applied = [];
  const resolvers = new Map();
  const scheduler = createSearchScheduler({
    delay: 100,
    setTimer: timers.setTimeout.bind(timers),
    clearTimer: timers.clearTimeout.bind(timers),
    search: (query) => {
      const promise = new Promise((resolve) => {
        resolvers.set(query, resolve);
      });
      promise.cancel = () => {
        cancelled.push(query);
      };
      return promise;
    },
    onResults: (results, query) => {
      applied.push({query, results});
    },
    onError: () => assert.fail('unexpected search error')
  });

  const first = scheduler.searchNow('f');
  scheduler.schedule('fi');
  assert.deepEqual(cancelled, ['f']);

  await timers.tick(100);

  resolvers.get('f')(['old']);
  await first;
  assert.deepEqual(applied, []);

  resolvers.get('fi')(['new']);
  await Promise.resolve();
  assert.deepEqual(applied, [{query: 'fi', results: ['new']}]);
});
