/**
 * @template T
 * @typedef {object} SearchSchedulerOptions
 * @property {number} delay
 * @property {(query: string) => Promise<T> | T} search
 * @property {(results: T, query: string) => void} onResults
 * @property {(error: unknown, query: string) => void} onError
 * @property {(callback: () => void, delay: number) => number} [setTimer]
 * @property {(timer: number) => void} [clearTimer]
 */

/**
 * @template T
 * @typedef {object} SearchScheduler
 * @property {(query: string) => void} schedule
 * @property {(query: string) => Promise<void>} searchNow
 * @property {() => void} cancel
 */

/**
 * @template T
 * @param {SearchSchedulerOptions<T>} options
 * @returns {SearchScheduler<T>}
 */
export function createSearchScheduler(options) {
  const setTimer = options.setTimer || globalThis.setTimeout.bind(globalThis);
  const clearTimer = options.clearTimer || globalThis.clearTimeout.bind(globalThis);
  /** @type {number | null} */
  let timer = null;
  let version = 0;

  /**
   * @param {string} query
   * @param {number} searchVersion
   */
  async function run(query, searchVersion) {
    try {
      const results = await options.search(query);
      if (searchVersion === version) {
        options.onResults(results, query);
      }
    } catch (error) {
      if (searchVersion === version) {
        options.onError(error, query);
      }
    }
  }

  function clearPendingTimer() {
    if (timer !== null) {
      clearTimer(timer);
      timer = null;
    }
  }

  /** @type {SearchScheduler<T>} */
  const scheduler = {
    schedule(query) {
      version += 1;
      const searchVersion = version;
      clearPendingTimer();
      timer = setTimer(() => {
        timer = null;
        void run(query, searchVersion);
      }, options.delay);
    },

    searchNow(query) {
      version += 1;
      const searchVersion = version;
      clearPendingTimer();
      return run(query, searchVersion);
    },

    cancel() {
      version += 1;
      clearPendingTimer();
    }
  };
  return scheduler;
}
