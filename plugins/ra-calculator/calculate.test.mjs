import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const html = fs.readFileSync(new URL('./assets/calculator/index.html', import.meta.url), 'utf8');
const script = html.match(/<script>([\s\S]*)<\/script>/)?.[1];
assert.ok(script, 'calculator script should be embedded');

const elements = new Map([
  ['#paper', {value: '', addEventListener() {}, focus() {}}],
  ['#results', {replaceChildren(...items) { this.items = items; }, items: []}],
  ['#status', {textContent: ''}],
  ['#copy', {addEventListener() {}}],
]);
const calls = [];
const context = vm.createContext({
  Function: undefined,
  URLSearchParams,
  location: {search: '?q=%3D2%2B1'},
  setTimeout(callback) {
    callback();
    return 1;
  },
  clearTimeout() {},
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
          return {data: {found: true, value: {text: '1+1\n2*3'}}};
        }
        if (action.type === 'store.set') {
          return {data: {ok: true}};
        }
        return {data: {}};
      },
    },
  },
});

vm.runInContext(script, context, {filename: 'calculator/index.html'});
for (let index = 0; index < 5; index += 1) {
  await Promise.resolve();
}

assert.equal(context.calculate('2+3*4'), '14');
assert.equal(context.calculate('(2+3)*4'), '20');
assert.equal(context.calculate('2/0'), 'Invalid expression');
assert.equal(context.calculate('2+bad'), 'Invalid expression');
assert.deepEqual(Array.from(context.calculateLines('1+1\n=2*3\n\nbad')), ['2', '6', '', 'Invalid expression']);
assert.equal(elements.get('#paper').value, '1+1\n2*3\n2+1');
assert.deepEqual(
  elements.get('#results').items.map((item) => item.textContent),
  ['2', '6', '3']
);
assert.deepEqual({...calls[0]}, {type: 'store.get', text: JSON.stringify({key: 'papers/current'})});
assert.deepEqual({...calls.at(-1)}, {
  type: 'store.set',
  text: JSON.stringify({key: 'papers/current', value: {text: '1+1\n2*3\n2+1'}}),
});
