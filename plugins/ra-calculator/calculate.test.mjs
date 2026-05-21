import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const html = fs.readFileSync(new URL('./assets/calculator/index.html', import.meta.url), 'utf8');
const script = html.match(/<script>([\s\S]*)<\/script>/)?.[1];
assert.ok(script, 'calculator script should be embedded');

function makeElement(value = '') {
  const listeners = new Map();
  return {
    value,
    addEventListener(type, listener) {
      listeners.set(type, listener);
    },
    focus() {},
    dispatch(type) {
      const listener = listeners.get(type);
      assert.ok(listener, `missing listener ${type}`);
      return listener();
    },
  };
}

const paper = makeElement();
const elements = new Map([
  ['#paper', paper],
  ['#results', {replaceChildren(...items) { this.items = items; }, items: []}],
  ['#status', {textContent: ''}],
  ['#copy', {addEventListener() {}}],
]);
const calls = [];
const context = vm.createContext({
  Function: undefined,
  URLSearchParams,
  location: {search: '?q=%3D2%2B1'},
  document: {
    createElement(tagName) {
      return {tagName, textContent: '', className: ''};
    },
    querySelector(selector) {
      const element = elements.get(selector);
      assert.ok(element, `missing element ${selector}`);
      return element;
    },
  },
  window: {
    ra: {
      async invoke(action) {
        calls.push(action);
        if (action.type === 'store.get') {
          return {value: '1+1\n=2*3'};
        }
        return {ok: true};
      },
    },
  },
});

vm.runInContext(script, context, {filename: 'calculator/index.html'});
await new Promise(setImmediate);

assert.equal(context.calculate('2+3*4'), '14');
assert.equal(context.calculate('(2+3)*4'), '20');
assert.equal(context.calculate('2/0'), 'Invalid expression');
assert.equal(context.calculate('2+bad'), 'Invalid expression');
assert.deepEqual(Array.from(context.calculateLines('1+1\n=2*3\n\nbad')), ['2', '6', '', 'Invalid expression']);
assert.equal(paper.value, '1+1\n=2*3\n2+1');
assert.deepEqual(
  elements.get('#results').items.map((item) => item.textContent),
  ['2', '6', '3']
);
paper.value = `${paper.value}\n4+5`;
await paper.dispatch('input');
assert.deepEqual(JSON.parse(JSON.stringify(calls)), [
  {type: 'store.get', text: '{"key":"papers/current"}'},
  {type: 'store.set', text: '{"key":"papers/current","value":"1+1\\n=2*3\\n2+1"}'},
  {type: 'store.set', text: '{"key":"papers/current","value":"1+1\\n=2*3\\n2+1\\n4+5"}'},
]);
