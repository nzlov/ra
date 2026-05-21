<script lang="ts">
  import {onMount} from 'svelte';
  import {Events, Window} from '@wailsio/runtime';
  import {LauncherService} from '../bindings/github.com/nzlov/ra/internal/app';
  import {launcherStateForOpen, shouldHideForEscape, shouldReturnToLauncherForEscape} from './launcherWindowBehavior.js';
  import {createSearchScheduler} from './searchScheduler.js';

  type Action = {
    type: string;
    appId?: string;
    text?: string;
    pluginId?: string;
    capabilityId?: string;
    ui?: string;
    query?: string;
  };

  type Result = {
    id: string;
    title: string;
    subtitle: string;
    kind: string;
    action: Action;
  };

  type InvokeResult = {
    type: string;
    message: string;
    data?: unknown;
  };

  type PluginMessage = {
    ra?: string;
    id?: string;
    action?: Action;
  };

  type ServiceStatus = {
    appCount: number;
    pluginCount: number;
    pluginErrorCount: number;
    pluginRoots: string[];
  };

  const fallbackResults: Result[] = [
    {
      id: 'demo:calculator',
      title: 'Type =6*7',
      subtitle: 'Calculator queries start with =',
      kind: 'hint',
      action: {type: 'noop'}
    },
    {
      id: 'demo:manager',
      title: 'Open Plugin Manager',
      subtitle: 'Install a local .wasm plugin package',
      kind: 'plugin',
      action: {type: 'plugin.manage', pluginId: 'ra-plugin-manager', capabilityId: 'manage'}
    }
  ];

  let query = '';
  let results: Result[] = fallbackResults;
  let status = 'Local preview';
  let serviceStatus: ServiceStatus | null = null;
  let view: 'launcher' | 'capability' = 'launcher';
  let activeCapability: Result | null = null;
  let activeIndex = 0;
  let searchInput: HTMLInputElement;
  let capabilityFrame: HTMLIFrameElement | null = null;

  const searchScheduler = createSearchScheduler<Result[]>({
    delay: 100,
    search: (searchQuery) => LauncherService.Search(searchQuery),
    onResults: (searchResults) => {
      results = searchResults;
      status = `${results.length} result${results.length === 1 ? '' : 's'}`;
      activeIndex = 0;
    },
    onError: () => {
      useFallbackResults();
      status = 'Local preview';
    }
  });

  function search() {
    searchScheduler.schedule(query);
  }

  async function searchNow() {
    await searchScheduler.searchNow(query);
  }

  async function invoke(result: Result) {
    if (result.action.type === 'plugin.manage') {
      openCapability({...result, action: {...result.action, type: 'capability.open'}});
      return;
    }
    if (result.action.type === 'capability.open') {
      openCapability(result);
      return;
    }
    try {
      const response = result.action.pluginId && result.action.capabilityId
        ? await LauncherService.InvokePluginAction({
            pluginId: result.action.pluginId,
            capabilityId: result.action.capabilityId,
            action: result.action
          })
        : await LauncherService.Invoke(result.action);
      status = response.message || response.type;
    } catch (error) {
      status = `${result.action.type}: ${result.title}`;
    }
  }

  function openCapability(result: Result) {
    activeCapability = result;
    view = 'capability';
    status = `${result.action.pluginId}.${result.action.capabilityId}`;
  }

  function backToLauncher() {
    activeCapability = null;
    view = 'launcher';
    setTimeout(() => searchInput?.focus(), 0);
  }

  function hideWindow() {
    Window.Hide();
  }

  function resetForOpen() {
    const state = launcherStateForOpen();
    activeCapability = state.activeCapability;
    activeIndex = state.activeIndex;
    query = state.query;
    view = state.view;
    setTimeout(() => searchInput?.focus(), 0);
  }

  function errorMessage(error: unknown) {
    if (error instanceof Error) {
      return error.message;
    }
    return String(error);
  }

  function useFallbackResults() {
    const needle = query.trim().toLowerCase();
    results = fallbackResults.filter((result) => {
      const haystack = `${result.title} ${result.subtitle}`.toLowerCase();
      return needle === '' || haystack.includes(needle);
    });
    if (results.length === 0) {
      results = fallbackResults;
    }
  }

  function keydown(event: KeyboardEvent) {
    if (event.key === 'ArrowDown') {
      activeIndex = Math.min(activeIndex + 1, results.length - 1);
      event.preventDefault();
    }
    if (event.key === 'ArrowUp') {
      activeIndex = Math.max(activeIndex - 1, 0);
      event.preventDefault();
    }
    if (event.key === 'Enter' && results[activeIndex]) {
      invoke(results[activeIndex]);
      event.preventDefault();
    }
  }

  function windowKeydown(event: KeyboardEvent) {
    if (shouldReturnToLauncherForEscape(event.key, view)) {
      backToLauncher();
      event.preventDefault();
      return;
    }
    if (shouldHideForEscape(event.key, query, view)) {
      hideWindow();
      event.preventDefault();
    }
  }

  async function pluginMessage(event: MessageEvent<PluginMessage>) {
    if (!activeCapability || !capabilityFrame || event.source !== capabilityFrame.contentWindow) {
      return;
    }
    const message = event.data || {};
    if (message.ra !== 'invoke' || !message.id || !message.action) {
      return;
    }
    const responseTarget = capabilityFrame.contentWindow;
    try {
      const result = await LauncherService.InvokePluginAction({
        pluginId: activeCapability.action.pluginId || '',
        capabilityId: activeCapability.action.capabilityId || '',
        action: message.action
      });
      if (result.type.startsWith('plugins.')) {
        await refreshStatus();
        await searchNow();
      }
      responseTarget?.postMessage({ra: 'response', id: message.id, result}, '*');
    } catch (error) {
      responseTarget?.postMessage({ra: 'response', id: message.id, error: errorMessage(error)}, '*');
    }
  }

  onMount(() => {
    resetForOpen();
    const removeLostFocusHandler = Events.On(Events.Types.Common.WindowLostFocus, hideWindow);
    const removeFocusHandler = Events.On(Events.Types.Common.WindowFocus, resetForOpen);
    refreshStatus();
    return () => {
      removeLostFocusHandler();
      removeFocusHandler();
    };
  });

  async function refreshStatus() {
    try {
      serviceStatus = await LauncherService.Status();
    } catch (error) {
      serviceStatus = null;
    }
  }

  $: query, search();
</script>

<svelte:window on:keydown={windowKeydown} on:message={pluginMessage} />

<main class="launcher">
  {#if view === 'launcher'}
    <section class="surface" aria-label="RA launcher">
    <div class="search-row">
      <div class="mark">RA</div>
      <input
        bind:value={query}
        bind:this={searchInput}
        on:keydown={keydown}
        autocomplete="off"
        spellcheck="false"
        placeholder="Search apps, run =1+2, open plugins"
        aria-label="Search"
      />
    </div>

    <div class="results" role="listbox" aria-label="Search results">
      {#each results as result, index}
        <button
          class:active={index === activeIndex}
          type="button"
          role="option"
          aria-selected={index === activeIndex}
          on:mouseenter={() => (activeIndex = index)}
          on:click={() => invoke(result)}
        >
          <span class="kind">{result.kind}</span>
          <span class="text">
            <strong>{result.title}</strong>
            <small>{result.subtitle || result.action.type}</small>
          </span>
        </button>
      {/each}
    </div>

    <footer>
      <span>{status}</span>
      {#if serviceStatus}
        <span>{serviceStatus.appCount} apps</span>
        <span>{serviceStatus.pluginCount} plugins</span>
        {#if serviceStatus.pluginErrorCount > 0}
          <span class="warning">{serviceStatus.pluginErrorCount} plugin errors</span>
        {/if}
      {/if}
    </footer>
  </section>
  {:else}
    <section class="surface capability-surface" aria-label="RA plugin capability">
      <header class="manager-header">
        <button class="icon-button" type="button" title="Back" aria-label="Back" on:click={backToLauncher}>
          &larr;
        </button>
        <div class="manager-title">
          <strong>{activeCapability?.title || 'Plugin Capability'}</strong>
          <small>{activeCapability?.subtitle || activeCapability?.action.pluginId || 'RA plugin'}</small>
        </div>
      </header>

      {#if activeCapability?.action.pluginId && activeCapability.action.capabilityId && activeCapability.action.ui}
        <iframe
          bind:this={capabilityFrame}
          title={activeCapability.title}
          sandbox="allow-scripts"
          src={`/plugins/${activeCapability.action.pluginId}/${activeCapability.action.capabilityId}${activeCapability.action.ui}?q=${encodeURIComponent(activeCapability.action.query || '')}`}
        ></iframe>
      {/if}

      <footer>
        <span>{status}</span>
      </footer>
    </section>
  {/if}
</main>
