<script lang="ts">
  import {onMount} from 'svelte';
  import {LauncherService} from '../bindings/github.com/nzlov/ra/internal/app';

  type Action = {
    type: string;
    appId?: string;
    command?: string;
    text?: string;
    pluginId?: string;
    commandId?: string;
    entryPath?: string;
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
    entryPath?: string;
    url?: string;
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
      id: 'demo:plugin',
      title: 'Open Example Plugin',
      subtitle: 'HTML page with a WASM slot',
      kind: 'plugin',
      action: {type: 'plugin.open', pluginId: 'example-webview'}
    }
  ];

  let query = '';
  let results: Result[] = fallbackResults;
  let status = 'Local preview';
  let serviceStatus: ServiceStatus | null = null;
  let activeIndex = 0;
  let searchInput: HTMLInputElement;

  async function search() {
    try {
      results = await LauncherService.Search(query);
      status = `${results.length} result${results.length === 1 ? '' : 's'}`;
      activeIndex = 0;
    } catch (error) {
      useFallbackResults();
      status = 'Local preview';
    }
  }

  async function invoke(result: Result) {
    try {
      const response = await LauncherService.Invoke(result.action);
      handleInvokeResult(response);
      status = response.message || response.type;
    } catch (error) {
      status = `${result.action.type}: ${result.title}`;
    }
  }

  function handleInvokeResult(response: InvokeResult) {
    if (response.type === 'plugin.open' && response.url) {
      window.open(response.url, '_blank', 'noopener,noreferrer');
    }
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

  onMount(() => {
    searchInput?.focus();
    refreshStatus();
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

<main class="launcher">
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
</main>
