<script lang="ts">
  import {onMount} from 'svelte';
  import {Dialogs} from '@wailsio/runtime';
  import {LauncherService} from '../bindings/github.com/nzlov/ra/internal/app';
  import type {ManagedPlugin, PluginManagerState} from '../bindings/github.com/nzlov/ra/internal/app/models';

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
  let managerState: PluginManagerState | null = null;
  let managerStatus = 'Ready';
  let view: 'launcher' | 'manager' = 'launcher';
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
    if (result.action.type === 'plugin.manage') {
      await openPluginManager();
      return;
    }
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

  async function openPluginManager() {
    view = 'manager';
    await refreshPluginManager();
  }

  async function refreshPluginManager() {
    try {
      managerState = await LauncherService.PluginManagerState();
      serviceStatus = await LauncherService.Status();
      managerStatus = `${managerState.plugins.length} plugin${managerState.plugins.length === 1 ? '' : 's'}`;
    } catch (error) {
      managerStatus = errorMessage(error);
    }
  }

  async function installPlugin() {
    try {
      const selected = await Dialogs.OpenFile({
        Title: 'Install RA plugin',
        Message: 'Select a local plugin directory',
        ButtonText: 'Install',
        CanChooseDirectories: true,
        CanChooseFiles: false
      });
      if (!selected || Array.isArray(selected)) {
        return;
      }
      const result = await LauncherService.InstallPlugin(selected);
      managerState = result.state;
      serviceStatus = await LauncherService.Status();
      managerStatus = `Installed ${result.pluginId}`;
    } catch (error) {
      managerStatus = errorMessage(error);
    }
  }

  async function setPluginEnabled(plugin: ManagedPlugin, enabled: boolean) {
    try {
      managerState = await LauncherService.SetPluginEnabled(plugin.id, enabled);
      serviceStatus = await LauncherService.Status();
      managerStatus = `${enabled ? 'Enabled' : 'Disabled'} ${plugin.name}`;
      await search();
    } catch (error) {
      managerStatus = errorMessage(error);
    }
  }

  function pluginToggleChanged(plugin: ManagedPlugin, event: Event) {
    const target = event.currentTarget;
    if (!(target instanceof HTMLInputElement)) {
      return;
    }
    setPluginEnabled(plugin, target.checked);
  }

  async function uninstallPlugin(plugin: ManagedPlugin) {
    try {
      const choice = await Dialogs.Question({
        Title: 'Uninstall plugin',
        Message: `Remove ${plugin.name}?`,
        Buttons: [
          {Label: 'Cancel', IsCancel: true},
          {Label: 'Uninstall', IsDefault: true}
        ]
      });
      if (choice !== 'Uninstall') {
        return;
      }
      managerState = await LauncherService.UninstallPlugin(plugin.id);
      serviceStatus = await LauncherService.Status();
      managerStatus = `Uninstalled ${plugin.name}`;
      await search();
    } catch (error) {
      managerStatus = errorMessage(error);
    }
  }

  async function reloadPlugins() {
    try {
      await LauncherService.RefreshPlugins();
      await refreshPluginManager();
      await search();
      managerStatus = 'Refreshed plugins';
    } catch (error) {
      managerStatus = errorMessage(error);
    }
  }

  function backToLauncher() {
    view = 'launcher';
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
    if (event.key === 'Escape' && view === 'manager') {
      backToLauncher();
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

<svelte:window on:keydown={windowKeydown} />

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
    <section class="surface manager-surface" aria-label="RA plugin manager">
      <header class="manager-header">
        <button class="icon-button" type="button" title="Back" aria-label="Back" on:click={backToLauncher}>
          &larr;
        </button>
        <div class="manager-title">
          <strong>RA Plugin Manager</strong>
          <small>{managerState?.userPluginRoot || 'Local plugins'}</small>
        </div>
        <button type="button" class="command-button" on:click={installPlugin}>Install</button>
        <button type="button" class="command-button" on:click={reloadPlugins}>Refresh</button>
      </header>

      <div class="manager-list">
        {#if managerState}
          {#each managerState.plugins as plugin}
            <article class:disabled={!plugin.enabled} class="plugin-row">
              <div class="plugin-main">
                <span class="kind">{plugin.source}</span>
                <span class="text">
                  <strong>{plugin.name}</strong>
                  <small>{plugin.id} &middot; {plugin.type}</small>
                </span>
              </div>
              <div class="plugin-actions">
                {#if plugin.protected}
                  <span class="locked">Protected</span>
                {:else}
                  <label class="switch">
                    <input
                      type="checkbox"
                      checked={plugin.enabled}
                      on:change={(event) => pluginToggleChanged(plugin, event)}
                    />
                    <span>{plugin.enabled ? 'Enabled' : 'Disabled'}</span>
                  </label>
                  <button
                    type="button"
                    class="danger-button"
                    disabled={!plugin.uninstallable}
                    title={plugin.uninstallable ? 'Uninstall plugin' : 'Only user plugins can be uninstalled'}
                    on:click={() => uninstallPlugin(plugin)}
                  >
                    Remove
                  </button>
                {/if}
              </div>
            </article>
          {/each}

          {#if managerState.loadErrors.length > 0}
            <section class="error-list" aria-label="Plugin load errors">
              <strong>Load Errors</strong>
              {#each managerState.loadErrors as loadError}
                <div class="error-row">
                  <small>{loadError.path}</small>
                  <span>{loadError.error}</span>
                </div>
              {/each}
            </section>
          {/if}
        {/if}
      </div>

      <footer>
        <span>{managerStatus}</span>
        {#if managerState}
          <span>{managerState.plugins.length} plugins</span>
          <span>{managerState.loadErrors.length} errors</span>
        {/if}
      </footer>
    </section>
  {/if}
</main>
