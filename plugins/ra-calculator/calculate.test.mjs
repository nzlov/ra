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
        return {data: {}};
      },
    },
  },
});

vm.runInContext(script, context, {filename: 'calculator/index.html'});

assert.equal(context.calculate('2+3*4'), '14');
assert.equal(context.calculate('(2+3)*4'), '20');
assert.equal(context.calculate('2/0'), 'Invalid expression');
assert.equal(context.calculate('2+bad'), 'Invalid expression');
assert.deepEqual(Array.from(context.calculateLines('1+1\n=2*3\n\nbad')), ['2', '6', '', 'Invalid expression']);
assert.equal(elements.get('#paper').value, '2+1');
assert.deepEqual(
  elements.get('#results').items.map((item) => item.textContent),
  ['3']
);
assert.deepEqual(calls, []);
