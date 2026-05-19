const status = document.querySelector('#status');
const runButton = document.querySelector('#run');

runButton?.addEventListener('click', async () => {
  status.textContent = 'checking wasm module...';

  try {
    const response = await fetch('./plugin.wasm');
    if (!response.ok) {
      status.textContent = 'plugin.wasm is not built yet';
      return;
    }
    const module = await WebAssembly.instantiateStreaming(response, {});
    const exports = module.instance.exports;
    if (typeof exports.answer === 'function') {
      status.textContent = `wasm answer: ${exports.answer()}`;
      return;
    }
    status.textContent = 'wasm loaded without answer() export';
  } catch (error) {
    status.textContent = error instanceof Error ? error.message : String(error);
  }
});
