/**
 * @param {string} key
 * @param {string} query
 * @param {'launcher' | 'capability'} view
 */
export function shouldCloseForEscape(key, query, view) {
  return key === 'Escape' && view === 'launcher' && query.trim() === '';
}

/**
 * @param {string} key
 * @param {'launcher' | 'capability'} view
 */
export function shouldReturnToLauncherForEscape(key, view) {
  return key === 'Escape' && view !== 'launcher';
}
