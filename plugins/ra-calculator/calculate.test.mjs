import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const html = fs.readFileSync(new URL('./assets/calculator/index.html', import.meta.url), 'utf8');
const script = html.match(/<script>([\s\S]*)<\/script>/)?.[1];
assert.ok(script, 'calculator script should be embedded');

const elements = new Map([
  ['#query', {value: '2+1', addEventListener() {}, focus() {}}],
  ['#result', {value: ''}],
  ['#copy', {addEventListener() {}}],
]);
const context = vm.createContext({
  Function: undefined,
  URLSearchParams,
  location: {search: '?q=2%2B1'},
  document: {
    querySelector(selector) {
      const element = elements.get(selector);
      assert.ok(element, `missing element ${selector}`);
      return element;
    },
  },
  window: {},
});

vm.runInContext(script, context, {filename: 'calculator/index.html'});

assert.equal(elements.get('#result').value, '3');
assert.equal(context.calculate('2+3*4'), '14');
assert.equal(context.calculate('(2+3)*4'), '20');
assert.equal(context.calculate('2/0'), 'Invalid expression');
assert.equal(context.calculate('2+bad'), 'Invalid expression');
