<script>
  import Dashboard from "./components/Dashboard.svelte";
  import Setup from "./components/Setup.svelte";
  import Installer from "./components/Installer.svelte";
  import Audit from "./components/Audit.svelte";
  import Cleaner from "./components/Cleaner.svelte";
  import Tweaks from "./components/Tweaks.svelte";
  import Debloater from "./components/Debloater.svelte";
  import Logs from "./components/Logs.svelte";

  import logoImg from "./assets/images/logo.png";
  import bgImg from "./assets/images/bg.jpg";

  import { onMount, tick } from "svelte";
  import { logStore, exportLogs, clearLogs } from "./stores/log.js";
  import { GetDashboardStatus, GetAppVersion } from "../wailsjs/go/main/App";

  const tabs = [
    { id: "dashboard",       label: "Dashboard",       sub: "Overview",      icon: "📊", component: Dashboard },
    { id: "setup",           label: "Setup",            sub: "Configuration", icon: "⚙️", component: Setup },
    { id: "installer",       label: "Installer",        sub: "Packages",      icon: "📦", component: Installer },
    { id: "audit",           label: "Audit & Security", sub: "TGS Tool",      icon: "🛡️", component: Audit },
    { id: "cleaner",         label: "Cleaner",          sub: "Maintenance",   icon: "🧹", component: Cleaner },
    { id: "tweaks",          label: "Optimizer",        sub: "Performance",   icon: "⚡", component: Tweaks },
    { id: "debloat",         label: "Debloater",        sub: "Windows",       icon: "🗑️", component: Debloater },
    { id: "logs",            label: "System Logs",      sub: "Event Analyzer",icon: "🚨", component: Logs },
  ];


  const CREDIT = "Developed by Yadav Guru Prasad";
  let activeTab = tabs[0];
  let isOnline = false;
  let appVersion = "V19.4";
  const themes = [
    { id: "deep-space", icon: "🌌", name: "Deep Space (Blue/Purple)" },
    { id: "light", icon: "☀️", name: "Clean (Blue/White)" },
    { id: "cyberpunk", icon: "⚡", name: "Cyberpunk (Yellow/Pink)" },
    { id: "hacker", icon: "💻", name: "Terminal (Hacker Green)" },
    { id: "crimson", icon: "🩸", name: "Crimson Void (Red/Black)" },
  ];
  let activeThemeIndex = 0;
  $: activeTheme = themes[activeThemeIndex];
  let isRefreshing = false;
  let searchQuery = "";
  let searchFocused = false;

  $: searchResults =
    searchQuery.trim().length > 0
      ? tabs.filter(
          (t) =>
            t.label.toLowerCase().includes(searchQuery.toLowerCase()) ||
            t.sub.toLowerCase().includes(searchQuery.toLowerCase()),
        )
      : [];

  function selectSearchResult(tab) {
    activeTab = tab;
    searchQuery = "";
    searchFocused = false;
  }

  // Console state: 'min' | 'mid' | 'max'
  let consoleState = "min";
  let logs = [];
  let bodyEl;
  let unsubLogs;

  onMount(async () => {
    // Subscribe to log store — unsubscribed on component destroy
    unsubLogs = logStore.subscribe(async (v) => {
      logs = v;
      if (consoleState !== "min" && bodyEl) {
        await tick();
        bodyEl.scrollTop = bodyEl.scrollHeight;
      }
    });

    try {
      const [s, v] = await Promise.all([GetDashboardStatus(), GetAppVersion()]);
      isOnline = !!(s && s.PowerPlan);
      if (v) appVersion = v;
    } catch (_) {
      isOnline = false;
    }
    applyTheme(activeTheme);

    return () => {
      if (unsubLogs) unsubLogs();
    };
  });

  function applyTheme(theme) {
    document.documentElement.setAttribute("data-theme", theme.id);
  }

  function cycleTheme() {
    activeThemeIndex = (activeThemeIndex + 1) % themes.length;
    applyTheme(themes[activeThemeIndex]);
  }

  async function handleRefresh() {
    if (isRefreshing) return;
    isRefreshing = true;
    try {
      const [s, v] = await Promise.all([GetDashboardStatus(), GetAppVersion()]);
      isOnline = !!(s && s.PowerPlan);
      if (v) appVersion = v;
    } catch (_) {
      isOnline = false;
    } finally {
      setTimeout(() => {
        isRefreshing = false;
      }, 800);
    }
  }

  function handleSettings() {
    // Toggle console open/close
    if (consoleState === "min") consoleState = "mid";
    else consoleState = "min";
  }

  function cycleConsole() {
    if (consoleState === "min") consoleState = "mid";
    else if (consoleState === "mid") consoleState = "max";
    else consoleState = "min";
  }

  function minimize(e) {
    e.stopPropagation();
    consoleState = "min";
  }
  function maximize(e) {
    e.stopPropagation();
    consoleState = "max";
  }
  function restore(e) {
    e.stopPropagation();
    consoleState = "mid";
  }
  function doExport(e) {
    e.stopPropagation();
    exportLogs(logs);
  }
  function doClear(e) {
    e.stopPropagation();
    clearLogs();
  }
</script>

<!-- Global background image -->
<div class="global-bg" aria-hidden="true">
  <img src={bgImg} alt="" class="global-bg-img" />
  <div class="global-bg-dim"></div>
</div>

<main class="shell">
  <header class="top-header">
    <div class="header-brand">
      <div class="brand-logo-wrap">
        <img src={logoImg} alt="logo" class="brand-logo" />
        <div class="brand-logo-ring" />
      </div>
      <div class="brand-info">
        <span class="brand-name">TGS WORLD</span>
      </div>
    </div>

    <div class="header-search-wrap">
      <div class="header-search">
        <svg class="hs-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
        <input 
          type="text" 
          class="hs-input" 
          placeholder="Search components or tasks..." 
          bind:value={searchQuery}
          on:focus={() => searchFocused = true}
          on:blur={() => setTimeout(() => searchFocused = false, 200)}
        />
        <span class="hs-kbd">Ctrl+K</span>
      </div>

      {#if searchFocused && (searchResults.length > 0 || searchQuery.trim())}
        <div class="search-dropdown">
          {#each searchResults as tab}
            <button class="search-result" on:click={() => selectSearchResult(tab)}>
              <span class="sr-label">{tab.label}</span>
              <span class="sr-sub">{tab.sub}</span>
            </button>
          {:else}
            <div class="sr-empty">No results for "{searchQuery}"</div>
          {/each}
        </div>
      {/if}
    </div>

    <div class="header-right">
      <div class="hdr-pill">
        <div class="hdr-dot {isOnline ? 'hdr-dot-live' : ''}" />
        <span class="hdr-pill-text">{isOnline ? 'PROTECTED' : 'OFFLINE'}</span>
      </div>
      
      <button class="hdr-icon-btn {isRefreshing ? 'hdr-spinning' : ''}" on:click={handleRefresh} title="Sync/Refresh">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 2v6h-6M3 12a9 9 0 0 1 15-6.7L21 8M3 22v-6h6M21 12a9 9 0 0 1-15 6.7L3 16"/></svg>
      </button>

      <button class="hdr-icon-btn" on:click={cycleTheme} title="Change Theme">
        <span>{activeTheme.icon}</span>
      </button>

      <button class="hdr-icon-btn {consoleState !== 'min' ? 'hdr-icon-active' : ''}" on:click={handleSettings} title="Toggle Console">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 17l6-6-6-6M12 19h8"/></svg>
      </button>

      <div class="hdr-user" style="display: flex; align-items: center; gap: 10px; margin-left: 10px; cursor: pointer;">
        <div style="width: 32px; height: 32px; border-radius: 50%; background: var(--grad); border: 1.5px solid var(--accent); display: flex; align-items: center; justify-content: center; font-weight: bold; font-size: 12px; color: #fff;">YG</div>
      </div>
    </div>
  </header>

  <div class="shell-body">
    <aside class="sidebar">
      <div class="sidebar-top">
        <nav class="nav-links">
          {#each tabs as tab}
            <button 
              class="nav-item" 
              class:active={activeTab.id === tab.id}
              on:click={() => activeTab = tab}
            >
              <div style="display: flex; align-items: center; gap: 12px; position: relative; z-index: 1;">
                <span style="font-size: 1.2rem;">{tab.icon}</span>
                <div class="nav-info">
                  <span class="nav-label">{tab.label}</span>
                  <p class="nav-sub">{tab.sub}</p>
                </div>
              </div>
            </button>
          {/each}
        </nav>
      </div>

      <div class="sidebar-footer">
        <div class="status-dot {isOnline ? 'live' : ''}" />
        <span class="status-label">{isOnline ? 'System Online' : 'System Offline'}</span>
        <span class="status-label" style="margin-left: auto; opacity: 0.5;">{appVersion}</span>
      </div>
    </aside>

    <section class="viewport">
      <svelte:component this={activeTab.component} />
    </section>
  </div>
</main>

<!-- TGS Console Terminal -->
<div
  class="cmd-terminal"
  class:cmd-mid={consoleState === "mid"}
  class:cmd-max={consoleState === "max"}
>
  <div
    class="cmd-header"
    on:click={cycleConsole}
    on:keydown={() => {}}
    role="button"
    tabindex="0"
  >
    <span class="cmd-title">📟 TGS CONSOLE</span>
    <span class="cmd-controls">
      {#if logs.length}<span class="cmd-count">{logs.length} entries</span>{/if}
      {#if logs.length > 0}
        <button class="cmd-btn" on:click={doClear} title="Clear logs"
          >✖ Clear</button
        >
        <button class="cmd-btn" on:click={doExport} title="Export logs"
          >⬇ Export</button
        >
      {/if}
      {#if consoleState !== "max"}
        <button class="cmd-btn" on:click={maximize} title="Maximize"
          >⬆ Max</button
        >
      {:else}
        <button class="cmd-btn" on:click={restore} title="Restore"
          >⬇ Mid</button
        >
      {/if}
      {#if consoleState !== "min"}
        <button class="cmd-btn" on:click={minimize} title="Minimize"
          >▼ Min</button
        >
      {:else}
        <button class="cmd-btn" on:click={restore} title="Open">▲ Open</button>
      {/if}
    </span>
  </div>
  {#if consoleState !== "min"}
    <div class="cmd-body" bind:this={bodyEl}>
      {#if logs.length === 0}
        <div class="cmd-line cmd-idle">
          <span class="cmd-ts">[--:--:--]</span>
          <span class="cmd-prompt">TGS &gt;</span>
          <span class="cmd-msg"
            >Waiting for commands… Perform any action to see logs here.</span
          >
        </div>
      {:else}
        {#each logs as entry}
          <div
            class="cmd-line"
            class:cmd-ok={entry.ok}
            class:cmd-err={entry.ok === false}
          >
            <span class="cmd-ts">[{entry.time}]</span>
            <span class="cmd-prompt">{entry.ok ? "✓" : "✗"}</span>
            <span class="cmd-msg">{entry.msg}</span>
          </div>
        {/each}
      {/if}
    </div>
  {/if}
</div>

<style>
  /* ═══ IMPORT GOOGLE FONTS ═══════════════════════════ */
  @import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800;900&family=Poppins:wght@400;500;600;700;800&display=swap');

  /* ═══ EXACT DESIGN TOKENS ════════════════════════════ */
  :global(:root) {
    /* Exact color palette from user spec */
    --bg:           #0B0E1A;
    --bg-panel:     rgba(20, 24, 42, 0.9);
    --bg-card:      rgba(20, 24, 42, 0.6);
    --bg-card-solid:#14182A;
    --bg-hover:     rgba(124, 58, 237, 0.18);
    --bg-active:    rgba(124, 58, 237, 0.3);

    /* Primary / Secondary accents */
    --accent:       #7C3AED;
    --accent-dim:   rgba(124, 58, 237, 0.15);
    --accent-text:  #c4b5fd;
    --accent2:      #22D3EE;
    --accent2-dim:  rgba(34, 211, 238, 0.15);
    --accent3:      #FF2CDF;
    --accent3-dim:  rgba(255, 44, 223, 0.15);

    /* Gradient  */
    --grad:         linear-gradient(135deg, #7C3AED, #22D3EE);
    --grad-rev:     linear-gradient(135deg, #22D3EE, #7C3AED);
    --grad-neon:    linear-gradient(135deg, #FF2CDF, #6C63FF);

    --success:      #10b981;
    --success-dim:  rgba(16, 185, 129, 0.15);
    --danger:       #ef4444;
    --danger-dim:   rgba(239, 68, 68, 0.15);
    --warn:         #f59e0b;

    /* Text */
    --text:         #E5E7EB;
    --text-dim:     #9CA3AF;
    --text-lo:      #4B5563;

    /* Borders */
    --border:       rgba(255, 255, 255, 0.07);
    --border-md:    rgba(255, 255, 255, 0.12);
    --border-hi:    rgba(124, 58, 237, 0.5);

    /* Radius */
    --radius-sm:    8px;
    --radius:       12px;
    --radius-lg:    16px;
    --radius-xl:    20px;

    /* Fonts */
    --font:         "Inter", "Poppins", "Segoe UI", system-ui, sans-serif;
    --font-mono:    "Cascadia Code", "JetBrains Mono", monospace;

    /* Glass card style (exact spec) */
    --glass-bg:     rgba(20, 24, 42, 0.6);
    --glass-border: rgba(255, 255, 255, 0.08);
    --glass-blur:   blur(10px);
    --glass-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
  }

  :global(*) { box-sizing: border-box; margin: 0; padding: 0; }

  :global(body) {
    margin: 0;
    background: var(--bg);
    color: var(--text);
    font-family: var(--font);
    font-size: 14px;
    -webkit-font-smoothing: antialiased;
    overflow: hidden;
  }

  /* ═══ GLOBAL TRANSITIONS ════════════════════════════ */
  :global(button), :global(.btn), :global(.card) {
    transition: transform 0.18s cubic-bezier(0.34,1.56,0.64,1),
      box-shadow 0.18s ease, background 0.18s ease, border-color 0.18s ease !important;
  }
  :global(button:hover:not(:disabled)), :global(.btn:hover:not(:disabled)) {
    transform: translateY(-2px) scale(1.03);
    box-shadow: 0 8px 24px rgba(124, 58, 237, 0.3), 0 2px 8px rgba(0,0,0,0.4);
  }
  :global(button:active:not(:disabled)), :global(.btn:active:not(:disabled)) {
    transform: translateY(0) scale(0.98);
  }
  :global(.card:hover) {
    transform: translateY(-3px);
    box-shadow: 0 16px 40px rgba(0,0,0,0.5), 0 0 20px rgba(124,58,237,0.12);
    border-color: rgba(124,58,237,0.3) !important;
  }
  :global(input:focus), :global(select:focus), :global(textarea:focus) {
    outline: none;
    border-color: var(--accent) !important;
    box-shadow: 0 0 0 3px rgba(124,58,237,0.2) !important;
  }

  /* ═══ GLOBAL BACKGROUND ═════════════════════════════ */
  .global-bg {
    position: fixed;
    inset: 0;
    z-index: 0;
    pointer-events: none;
  }
  .global-bg-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    opacity: 0.04;
    filter: blur(2px);
  }
  .global-bg-dim {
    position: absolute;
    inset: 0;
    background: radial-gradient(ellipse at 20% 30%, rgba(124,58,237,0.06) 0%, transparent 60%),
                radial-gradient(ellipse at 80% 70%, rgba(34,211,238,0.04) 0%, transparent 60%),
                var(--bg);
  }

  /* ═══ SHELL LAYOUT ══════════════════════════════════ */
  .shell {
    display: flex;
    flex-direction: column;
    width: 100vw;
    height: 100vh;
    position: relative;
    z-index: 1;
    overflow: hidden;
  }
  .shell-body {
    display: flex;
    flex: 1;
    overflow: hidden;
  }

  /* ═══ TOP HEADER ════════════════════════════════════ */
  .top-header {
    display: flex;
    height: 62px;
    min-height: 62px;
    background: rgba(11, 14, 26, 0.95);
    border-bottom: 1px solid var(--glass-border);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    z-index: 200;
    align-items: center;
    padding: 0 20px;
    position: relative;
    overflow: visible;
    gap: 16px;
  }
  /* Gradient glow line at bottom */
  .top-header::after {
    content: "";
    position: absolute;
    bottom: 0; left: 0; right: 0;
    height: 1px;
    background: linear-gradient(90deg, transparent 0%, #7C3AED 35%, #22D3EE 65%, transparent 100%);
    opacity: 0.6;
  }

  /* BRAND */
  .header-brand {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 220px;
    flex-shrink: 0;
  }
  .brand-logo-wrap {
    position: relative;
    width: 40px; height: 40px;
    flex-shrink: 0;
  }
  .brand-logo {
    width: 40px; height: 40px;
    border-radius: 10px;
    object-fit: contain;
    position: relative; z-index: 1;
    filter: drop-shadow(0 0 10px rgba(124,58,237,0.7));
    transition: filter 0.3s ease, transform 0.3s ease;
  }
  .brand-logo:hover { filter: drop-shadow(0 0 20px rgba(124,58,237,1)); transform: scale(1.1); }
  .brand-logo-ring {
    position: absolute; inset: -4px;
    border-radius: 13px;
    border: 1.5px solid rgba(124,58,237,0.5);
    animation: ring-pulse 3s ease-in-out infinite;
    pointer-events: none;
  }
  @keyframes ring-pulse {
    0%, 100% { opacity: 0.4; transform: scale(1); border-color: rgba(124,58,237,0.5); }
    50% { opacity: 1; transform: scale(1.07); border-color: rgba(34,211,238,0.8); }
  }
  .brand-info { display: flex; flex-direction: column; }
  .brand-name {
    font-family: "Poppins", var(--font);
    font-size: 1.05rem;
    font-weight: 800;
    letter-spacing: 2px;
    text-transform: uppercase;
    background: linear-gradient(90deg, #7C3AED 0%, #22D3EE 50%, #FF2CDF 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    line-height: 1;
  }

  /* SEARCH (CENTER) */
  .header-search-wrap {
    flex: 1;
    display: flex;
    justify-content: center;
    padding: 0 20px;
  }
  .header-search {
    display: flex;
    align-items: center;
    width: 100%;
    max-width: 440px;
    background: #14182A;
    border: 1px solid var(--glass-border);
    border-radius: 10px;
    padding: 9px 14px;
    transition: all 0.25s ease;
  }
  .header-search:focus-within {
    border-color: rgba(124,58,237,0.5);
    box-shadow: 0 0 0 3px rgba(124,58,237,0.12), 0 0 20px rgba(124,58,237,0.08);
  }
  .hs-icon { width: 15px; height: 15px; color: var(--text-dim); flex-shrink: 0; }
  .hs-input {
    flex: 1;
    background: transparent;
    border: none; outline: none;
    color: var(--text);
    font-family: var(--font);
    font-size: 0.82rem;
    margin: 0 10px;
  }
  .hs-input::placeholder { color: var(--text-lo); }
  .hs-kbd {
    font-family: var(--font-mono); font-size: 0.65rem;
    color: var(--text-lo);
    background: rgba(255,255,255,0.05);
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 4px; padding: 2px 6px;
  }
  .hs-clear { background: transparent; border: none; color: var(--text-lo); cursor: pointer; padding: 4px; }
  .hs-clear:hover { color: #fff; }
  .search-dropdown {
    position: absolute; top: 58px; left: 50%;
    transform: translateX(-50%);
    width: 440px;
    background: rgba(20,24,42,0.98);
    border: 1px solid var(--glass-border);
    backdrop-filter: blur(20px);
    border-radius: 12px;
    box-shadow: 0 10px 40px rgba(0,0,0,0.6);
    z-index: 1000; overflow: hidden; padding: 8px;
  }
  .search-result {
    display: flex; flex-direction: column;
    padding: 10px 14px; width: 100%;
    text-align: left; background: transparent;
    border: none; border-radius: 8px;
    cursor: pointer; transition: background 0.15s ease;
  }
  .search-result:hover { background: rgba(124,58,237,0.15); }
  .sr-label { font-size: 0.9rem; font-weight: 600; color: var(--text); }
  .sr-sub { font-size: 0.75rem; color: var(--text-dim); margin-top: 2px; }
  .sr-empty { padding: 16px; text-align: center; color: var(--text-dim); font-size: 0.85rem; }

  /* RIGHT HEADER */
  .header-right {
    display: flex; align-items: center; gap: 10px;
    flex-shrink: 0;
  }
  /* ONLINE status pill */
  .hdr-pill {
    display: flex; align-items: center; gap: 6px;
    background: rgba(34,211,238,0.06);
    border: 1px solid rgba(34,211,238,0.2);
    border-radius: 99px; padding: 5px 14px 5px 10px;
  }
  .hdr-dot {
    width: 7px; height: 7px;
    border-radius: 50%;
    background: var(--text-lo);
    flex-shrink: 0;
    transition: background 0.3s;
  }
  .hdr-dot-live {
    background: #22D3EE;
    box-shadow: 0 0 8px #22D3EE, 0 0 16px rgba(34,211,238,0.4);
    animation: dot-pulse 2s ease-in-out infinite;
  }
  @keyframes dot-pulse {
    0%, 100% { box-shadow: 0 0 6px #22D3EE; }
    50% { box-shadow: 0 0 14px #22D3EE, 0 0 28px rgba(34,211,238,0.5); }
  }
  .hdr-pill-text { font-size: 0.72rem; font-weight: 700; letter-spacing: 0.5px; color: #22D3EE; }
  .hdr-icon-btn {
    width: 36px; height: 36px;
    border-radius: 10px;
    background: rgba(255,255,255,0.04);
    border: 1px solid var(--glass-border);
    color: var(--text-dim);
    cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    transition: all 0.2s ease;
  }
  .hdr-icon-btn svg { width: 16px; height: 16px; }
  .hdr-icon-btn:hover { background: rgba(124,58,237,0.15); border-color: rgba(124,58,237,0.4); color: #c4b5fd; }
  .hdr-icon-accent { border-color: rgba(124,58,237,0.3); }
  .hdr-icon-active {
    background: rgba(124,58,237,0.2);
    border-color: var(--accent);
    color: var(--accent-text);
    box-shadow: 0 0 12px rgba(124,58,237,0.3);
  }
  @keyframes hdr-spin { to { transform: rotate(360deg); } }
  .hdr-spinning svg { animation: hdr-spin 0.8s linear infinite; }

  /* ═══ SIDEBAR ════════════════════════════════════════ */
  .sidebar {
    width: 220px;
    min-width: 220px;
    background: rgba(11, 14, 26, 0.92);
    border-right: 1px solid var(--glass-border);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    position: relative;
    z-index: 100;
    backdrop-filter: blur(20px);
  }
  /* subtle purple glow on right edge */
  .sidebar::after {
    content: "";
    position: absolute;
    top: 0; right: 0; bottom: 0;
    width: 1px;
    background: linear-gradient(180deg, transparent, rgba(124,58,237,0.4), transparent);
  }
  .sidebar-top { flex: 1; padding: 16px 12px; overflow-y: auto; }
  .nav-links { display: flex; flex-direction: column; gap: 4px; }
  .nav-item {
    width: 100%;
    display: flex; flex-direction: column;
    padding: 10px 14px;
    background: transparent;
    border: none;
    border-radius: var(--radius);
    color: var(--text-dim);
    cursor: pointer;
    text-align: left;
    font-family: var(--font);
    transition: all 0.2s ease;
    position: relative;
    overflow: hidden;
  }
  .nav-item::before {
    content: "";
    position: absolute; inset: 0;
    background: var(--grad);
    opacity: 0;
    transition: opacity 0.2s ease;
    border-radius: var(--radius);
  }
  .nav-item:hover { color: var(--text); }
  .nav-item:hover::before { opacity: 0.12; }
  .nav-item.active {
    background: linear-gradient(135deg, #7C3AED, #22D3EE);
    color: #fff;
    box-shadow: 0 4px 20px rgba(124,58,237,0.35), 0 0 20px rgba(34,211,238,0.15);
  }
  .nav-item.active::before { opacity: 0; }
  .nav-item.active .nav-sub { color: rgba(255,255,255,0.7); }
  .nav-label { font-size: 0.88rem; font-weight: 600; position: relative; z-index: 1; }
  .nav-sub { font-size: 0.7rem; color: var(--text-lo); position: relative; z-index: 1; margin-top: 2px; }

  .sidebar-footer {
    display: flex; align-items: center; gap: 8px;
    padding: 14px 16px;
    border-top: 1px solid var(--glass-border);
  }
  .status-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--text-lo); flex-shrink: 0; }
  .status-dot.live { background: #22D3EE; box-shadow: 0 0 8px #22D3EE; }
  .status-label { font-size: 0.72rem; color: var(--text-dim); }

  /* ═══ VIEWPORT ══════════════════════════════════════ */
  .viewport {
    flex: 1;
    overflow-y: auto;
    background: transparent;
    padding-bottom: 48px;
    position: relative; z-index: 1;
  }

  /* ═══ SCROLLBAR ═════════════════════════════════════ */
  :global(::-webkit-scrollbar) { width: 4px; height: 4px; }
  :global(::-webkit-scrollbar-track) { background: transparent; }
  :global(::-webkit-scrollbar-thumb) { background: rgba(124,58,237,0.3); border-radius: 4px; }
  :global(::-webkit-scrollbar-thumb:hover) { background: rgba(124,58,237,0.6); }

  /* ═══ VIEWPORT HEIGHT FIX ══════════════════════════════ */
  /* Force all direct children of viewport to scroll via viewport, not clip */
  :global(.viewport > *) {
    min-height: 100%;
  }
  /* Fix any component that self-clips on height */
  :global(.inst-page),
  :global(.dash),
  :global(.cleaner-page),
  :global(.optimizer-page),
  :global(.debloater-page),
  :global(.setup-wrap),
  :global(.audit-wrap) {
    min-height: 100%;
  }

  /* ═══ THEMES ═════════════════════════════════════════ */
  /* theme 1: Deep Space - default (same as root) */
  :global([data-theme="deep-space"]) { --accent: #7C3AED; --accent2: #22D3EE; }

  /* theme 2: Blue/White clean */
  :global([data-theme="light"]) {
    --bg: #f8fafc; --bg-panel: rgba(255,255,255,0.95); --bg-card: rgba(255,255,255,0.98); --bg-card-solid: #ffffff;
    --accent: #6d28d9; --accent2: #06b6d4; --text: #1e293b; --text-dim: #475569; --text-lo: #94a3b8;
    --border: rgba(0,0,0,0.08); --border-md: rgba(0,0,0,0.15); --border-hi: rgba(109,40,217,0.5);
    --glass-bg: rgba(255,255,255,0.7); --glass-border: rgba(0,0,0,0.08); --glass-shadow: 0 10px 30px rgba(0,0,0,0.12);
    --grad: linear-gradient(135deg,#6d28d9,#06b6d4);
  }

  /* theme 3: Cyberpunk */
  :global([data-theme="cyberpunk"]) {
    --bg: #0d0221; --bg-card-solid: #1a0526;
    --accent: #ffdd00; --accent-dim: rgba(255,221,0,0.15); --accent-text: #ffe94d;
    --accent2: #ff0080; --text: #ffffff; --text-dim: #e2cce8; --text-lo: #a684b0;
    --border: rgba(255,0,128,0.2); --glass-border: rgba(255,0,128,0.15);
    --grad: linear-gradient(135deg,#ffdd00,#ff0080);
  }

  /* theme 4: Hacker Green */
  :global([data-theme="hacker"]) {
    --bg: #000000; --bg-card-solid: #001500;
    --accent: #00ff41; --accent-dim: rgba(0,255,65,0.15); --accent-text: #4dff79;
    --accent2: #008f11; --text: #00ff41; --text-dim: #00ce35; --text-lo: #006019;
    --border: rgba(0,255,65,0.15); --glass-border: rgba(0,255,65,0.1);
    --grad: linear-gradient(135deg,#00ff41,#008f11);
  }

  /* theme 5: Crimson */
  :global([data-theme="crimson"]) {
    --bg: #0a0202; --bg-card-solid: #190505;
    --accent: #ff2a40; --accent-dim: rgba(255,42,64,0.15); --accent-text: #ff6b7b;
    --accent2: #ff9e00; --text: #ffffff; --text-dim: #d1a5a8;
    --border: rgba(255,42,64,0.15); --glass-border: rgba(255,42,64,0.1);
    --grad: linear-gradient(135deg,#ff2a40,#ff9e00);
  }


  /* ═══ CMD TERMINAL ═══════════════════════════════════ */
  .cmd-terminal {
    position: fixed;
    bottom: 0;
    left: 220px;
    right: 0;
    height: 40px;
    background: rgba(5, 5, 14, 0.97);
    border-top: 1px solid rgba(124, 58, 237, 0.25);
    z-index: 1000;
    transition: height 0.28s cubic-bezier(0.4, 0, 0.2, 1);
    backdrop-filter: blur(18px);
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }
  .cmd-mid { height: 300px; }
  .cmd-max { height: 600px; }

  .cmd-header {
    background: rgba(0, 0, 0, 0.55);
    padding: 0 12px;
    height: 40px; min-height: 40px;
    display: flex; align-items: center; justify-content: space-between;
    cursor: pointer;
    border-bottom: 1px solid rgba(124, 58, 237, 0.15);
    flex-shrink: 0; user-select: none;
  }
  .cmd-header:hover { background: rgba(124, 58, 237, 0.08); }

  .cmd-title {
    font-family: "Cascadia Code", "Consolas", monospace;
    font-size: 0.88rem; font-weight: 700;
    color: var(--accent); letter-spacing: 1.2px;
    text-shadow: 0 0 10px rgba(124, 58, 237, 0.5);
  }
  .cmd-controls { display: flex; align-items: center; gap: 6px; }
  .cmd-count {
    font-family: monospace; font-size: 0.72rem;
    color: var(--text-dim);
    background: rgba(255, 255, 255, 0.05);
    padding: 2px 8px; border-radius: 10px; margin-right: 4px;
  }
  .cmd-btn {
    font-family: monospace; font-size: 0.72rem;
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid rgba(124, 58, 237, 0.25);
    color: var(--accent-text);
    padding: 3px 10px; border-radius: 6px;
    cursor: pointer; transition: background 0.15s; white-space: nowrap;
  }
  .cmd-btn:hover { background: rgba(124, 58, 237, 0.2); }

  .cmd-body {
    flex: 1;
    padding: 10px 16px;
    overflow-y: scroll; overflow-x: hidden;
    background: transparent;
    font-family: "Cascadia Code", "Consolas", monospace;
    min-height: 0; scroll-behavior: smooth;
  }
  .cmd-body::-webkit-scrollbar { width: 4px; }
  .cmd-body::-webkit-scrollbar-track { background: rgba(0,0,0,0.3); }
  .cmd-body::-webkit-scrollbar-thumb { background: rgba(124,58,237,0.4); border-radius: 3px; }
  .cmd-body::-webkit-scrollbar-thumb:hover { background: rgba(124,58,237,0.7); }

  .cmd-line { display: flex; gap: 10px; margin-bottom: 4px; font-size: 0.82rem; line-height: 1.5; animation: cmdIn 0.15s ease-out; }
  @keyframes cmdIn { from { opacity: 0; transform: translateX(-3px); } to { opacity: 1; transform: translateX(0); } }
  .cmd-ts { color: var(--text-lo); font-size: 0.75rem; min-width: 80px; flex-shrink: 0; user-select: none; }
  .cmd-prompt { font-weight: 700; min-width: 16px; flex-shrink: 0; }
  .cmd-msg { color: #e0e8f0; white-space: pre-wrap; word-break: break-word; flex: 1; }
  .cmd-ok .cmd-prompt { color: #4ade80; }
  .cmd-ok .cmd-msg { color: #d9f99d; }
  .cmd-err .cmd-prompt { color: #f87171; }
  .cmd-err .cmd-msg { color: #fecaca; }
  .cmd-idle .cmd-prompt { color: var(--accent); }
  .cmd-idle .cmd-msg { color: var(--text-dim); font-style: italic; }

  /* ── SCROLLBAR ───────────────────────────────────── */
  :global(::-webkit-scrollbar) {
    width: 4px;
    height: 4px;
  }
  :global(::-webkit-scrollbar-track) {
    background: transparent;
  }
  :global(::-webkit-scrollbar-thumb) {
    background: var(--border-md);
    border-radius: 4px;
  }
  :global(::-webkit-scrollbar-thumb:hover) {
    background: var(--border-hi);
  }

  /* ── GLOBAL SHARED LAYOUT ────────────────────────── */
  :global(.page) {
    padding: 32px 36px;
    display: flex;
    flex-direction: column;
    min-height: 100%;
    box-sizing: border-box;
  }
  :global(.page-header) {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 28px;
    padding-bottom: 20px;
    border-bottom: 1px solid var(--border);
  }
  :global(.page-title) {
    margin: 0;
    font-size: 1.1rem;
    font-weight: 700;
    color: var(--text);
  }
  :global(.page-desc) {
    margin: 4px 0 0;
    font-size: 0.8rem;
    color: var(--text-dim);
  }
  :global(.section-label) {
    font-size: 0.72rem;
    font-weight: 600;
    color: var(--text-dim);
    text-transform: uppercase;
    letter-spacing: 0.8px;
    margin-bottom: 12px;
  }
  /* Cards */
  :global(.card) {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    padding: 24px;
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    box-shadow: var(--glass-shadow);
  }
  /* Buttons */
  :global(.btn) {
    font-family: var(--font);
    font-size: 0.82rem;
    font-weight: 600;
    padding: 10px 20px;
    border-radius: var(--radius);
    cursor: pointer;
    border: none;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  }
  :global(.btn-primary) {
    background: var(--accent);
    color: #fff;
    border: none;
  }
  :global(.btn-primary:hover:not(:disabled)) {
    opacity: 0.88;
  }
  :global(.btn-primary:disabled) {
    opacity: 0.35;
    cursor: not-allowed;
  }
  :global(.btn-ghost) {
    background: transparent;
    border: 1px solid var(--border-md);
    color: var(--text-dim);
    font-size: 0.8rem;
    font-family: var(--font);
    font-weight: 500;
    padding: 8px 16px;
    border-radius: var(--radius);
    cursor: pointer;
    transition:
      background 0.15s,
      border-color 0.15s,
      color 0.15s;
  }
  :global(.btn-ghost:hover) {
    background: var(--bg-hover);
    border-color: var(--border-hi);
    color: var(--text);
  }
  :global(.btn-danger) {
    background: transparent;
    border: 1px solid var(--danger);
    color: var(--danger);
    font-size: 0.82rem;
    font-family: var(--font);
    font-weight: 600;
    padding: 9px 20px;
    border-radius: var(--radius);
    cursor: pointer;
    transition: background 0.15s;
  }
  :global(.btn-danger:hover:not(:disabled)) {
    background: var(--danger-dim);
  }
  :global(.btn-danger:disabled) {
    opacity: 0.35;
    cursor: not-allowed;
  }
  /* Status tags */
  :global(.tag-on) {
    display: inline-block;
    font-size: 0.68rem;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: var(--radius-sm);
    background: var(--success-dim);
    color: var(--success);
    border: 1px solid rgba(76, 175, 114, 0.25);
  }
  :global(.tag-off) {
    display: inline-block;
    font-size: 0.68rem;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: var(--radius-sm);
    background: var(--danger-dim);
    color: var(--danger);
    border: 1px solid rgba(224, 85, 85, 0.22);
  }
  :global(.tag-neutral) {
    display: inline-block;
    font-size: 0.68rem;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: var(--radius-sm);
    background: var(--bg-hover);
    color: var(--text-dim);
    border: 1px solid var(--border-md);
  }
  :global(.tag-accent) {
    display: inline-block;
    font-size: 0.68rem;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: var(--radius-sm);
    background: var(--accent-dim);
    color: var(--accent-text);
    border: 1px solid rgba(74, 143, 219, 0.25);
  }
  /* Input */
  :global(.input) {
    background: rgba(0, 0, 0, 0.4);
    border: 1px solid var(--border-md);
    border-radius: var(--radius);
    color: var(--text);
    font-family: var(--font);
    font-size: 0.85rem;
    padding: 10px 14px;
    outline: none;
    width: 100%;
    transition: all 0.2s;
    backdrop-filter: blur(5px);
  }
  :global(.input:focus) {
    border-color: var(--accent);
    background: rgba(0, 0, 0, 0.5);
    box-shadow: 0 0 0 2px var(--accent-dim);
  }
  /* Progress bar */
  :global(.bar-track) {
    height: 3px;
    background: var(--border);
    border-radius: 2px;
    overflow: hidden;
  }
  :global(.bar-fill) {
    height: 100%;
    border-radius: 2px;
    background: var(--accent);
    transition: width 0.5s ease;
  }
  :global(.bar-fill.warn) {
    background: var(--warn);
  }
  :global(.bar-fill.danger) {
    background: var(--danger);
  }
  /* KV row */
  :global(.kv) {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 0;
    border-bottom: 1px solid var(--border);
    font-size: 0.83rem;
  }
  :global(.kv:last-child) {
    border-bottom: none;
    padding-bottom: 0;
  }
  :global(.kv-k) {
    color: var(--text-dim);
  }
  :global(.kv-v) {
    color: var(--text);
    font-weight: 500;
    max-width: 58%;
    text-align: right;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  /* Divider */
  :global(.divider) {
    height: 1px;
    background: var(--border);
    margin: 20px 0;
  }
  /* Toast */
  :global(.toast) {
    position: fixed;
    bottom: 24px;
    right: 24px;
    background: var(--bg-card);
    border: 1px solid var(--border-hi);
    color: var(--text);
    padding: 12px 20px;
    border-radius: var(--radius);
    font-size: 0.8rem;
    font-family: var(--font);
    z-index: 9999;
    animation: fade-up 0.2s ease;
  }
  @keyframes fade-up {
    from {
      transform: translateY(6px);
      opacity: 0;
    }
    to {
      transform: translateY(0);
      opacity: 1;
    }
  }
  .status-dot.live {
    background: var(--success);
    box-shadow: 0 0 8px var(--success);
  }

  /* ═══ MODERN UTILITIES ══════════════════════════════ */
  :global(.page) {
    padding: 30px;
    height: 100%;
    overflow-y: auto;
    animation: fade-in 0.4s ease;
  }
  @keyframes fade-in { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: translateY(0); } }

  :global(.page-header) {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 30px;
  }
  :global(.page-title) {
    font-family: "Poppins", var(--font);
    font-size: 1.6rem;
    font-weight: 800;
    letter-spacing: -0.5px;
    margin-bottom: 4px;
    color: var(--text);
  }
  :global(.page-desc) {
    color: var(--text-lo);
    font-size: 0.9rem;
  }

  :global(.modern-table) {
    display: flex;
    flex-direction: column;
    width: 100%;
  }
  :global(.table-header) {
    display: grid;
    padding: 12px 16px;
    background: rgba(255,255,255,0.03);
    border-radius: 8px;
    color: var(--text-lo);
    font-size: 11px;
    font-weight: 800;
    letter-spacing: 1px;
    text-transform: uppercase;
    margin-bottom: 8px;
  }
  :global(.table-row) {
    display: grid;
    padding: 14px 16px;
    border-bottom: 1px solid var(--border);
    align-items: center;
    transition: background 0.2s;
    border-radius: 8px;
  }
  :global(.table-row:hover) {
    background: rgba(255,255,255,0.02);
  }
  :global(.text-dim) { color: var(--text-dim); }
  :global(.text-lo) { color: var(--text-lo); }

  /* ── CMD TERMINAL ───────────────────────────────── */
  .cmd-terminal {
    position: fixed;
    bottom: 0;
    left: 240px;
    right: 0;
    height: 40px;
    background: rgba(5, 5, 12, 0.95);
    border-top: 1px solid rgba(139, 92, 246, 0.25);
    z-index: 1000;
    transition: height 0.28s cubic-bezier(0.4, 0, 0.2, 1);
    backdrop-filter: blur(18px);
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }
  .cmd-mid {
    height: 300px;
  }
  .cmd-max {
    height: 600px;
  }

  .cmd-header {
    background: rgba(0, 0, 0, 0.6);
    padding: 0 12px;
    height: 40px;
    min-height: 40px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    cursor: pointer;
    border-bottom: 1px solid rgba(139, 92, 246, 0.15);
    flex-shrink: 0;
    user-select: none;
  }
  .cmd-header:hover {
    background: rgba(139, 92, 246, 0.08);
  }

  .cmd-title {
    font-family: "Cascadia Code", "Consolas", monospace;
    font-size: 0.88rem;
    font-weight: 700;
    color: var(--accent);
    letter-spacing: 1.2px;
    text-shadow: 0 0 8px rgba(74, 143, 219, 0.5);
  }
  .cmd-controls {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .cmd-count {
    font-family: monospace;
    font-size: 0.72rem;
    color: var(--text-dim);
    background: rgba(255, 255, 255, 0.05);
    padding: 2px 8px;
    border-radius: 10px;
    margin-right: 4px;
  }
  .cmd-btn {
    font-family: monospace;
    font-size: 0.72rem;
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid rgba(139, 92, 246, 0.25);
    color: var(--accent-text);
    padding: 3px 10px;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.15s;
    white-space: nowrap;
  }
  .cmd-btn:hover {
    background: rgba(139, 92, 246, 0.2);
  }

  .cmd-body {
    flex: 1;
    padding: 10px 16px 10px;
    overflow-y: scroll; /* Always show scrollbar */
    overflow-x: hidden;
    background: transparent;
    font-family: "Cascadia Code", "Consolas", monospace;
    min-height: 0;
    scroll-behavior: smooth;
  }
  /* Scrollbar styling for cmd body */
  .cmd-body::-webkit-scrollbar {
    width: 6px;
  }
  .cmd-body::-webkit-scrollbar-track {
    background: rgba(0, 0, 0, 0.3);
  }
  .cmd-body::-webkit-scrollbar-thumb {
    background: rgba(74, 143, 219, 0.4);
    border-radius: 3px;
  }
  .cmd-body::-webkit-scrollbar-thumb:hover {
    background: rgba(74, 143, 219, 0.7);
  }

  .cmd-line {
    display: flex;
    gap: 10px;
    margin-bottom: 4px;
    font-size: 0.82rem;
    line-height: 1.5;
    animation: cmdIn 0.15s ease-out;
  }
  @keyframes cmdIn {
    from {
      opacity: 0;
      transform: translateX(-3px);
    }
    to {
      opacity: 1;
      transform: translateX(0);
    }
  }
  .cmd-ts {
    color: var(--text-lo);
    font-size: 0.75rem;
    min-width: 80px;
    flex-shrink: 0;
    user-select: none;
  }
  .cmd-prompt {
    font-weight: 700;
    min-width: 16px;
    flex-shrink: 0;
  }
  .cmd-msg {
    color: #e0e8f0;
    white-space: pre-wrap;
    word-break: break-word;
    flex: 1;
  }
  /* Status Colors */
  .cmd-ok .cmd-prompt {
    color: #4ade80;
  }
  .cmd-ok .cmd-msg {
    color: #d9f99d;
  }
  .cmd-err .cmd-prompt {
    color: #f87171;
  }
  .cmd-err .cmd-msg {
    color: #fecaca;
  }
  .cmd-idle .cmd-prompt {
    color: var(--accent);
  }
  .cmd-idle .cmd-msg {
    color: var(--text-dim);
    font-style: italic;
  }
</style>
