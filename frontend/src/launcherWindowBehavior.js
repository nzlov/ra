/**
 * @param {string} key
 * @param {string} query
 * @param {'launcher' | 'capability'} view
 */
export function shouldHideForEscape(key, query, view) {
  return key === 'Escape' && view === 'launcher';
}

/**
 * @param {string} key
 * @param {'launcher' | 'capability'} view
 */
export function shouldReturnToLauncherForEscape(key, view) {
  return key === 'Escape' && view !== 'launcher';
}

/**
 * @returns {{activeCapability: null, activeIndex: number, query: string, view: 'launcher'}}
 */
export function launcherStateForOpen() {
  return {
    activeCapability: null,
    activeIndex: 0,
    query: '',
    view: 'launcher'
  };
}
