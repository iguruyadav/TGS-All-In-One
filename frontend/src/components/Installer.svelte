<script>
    import { onMount, onDestroy, tick } from "svelte";
    import {
        InstallSoftware,
        UninstallSoftware,
        DeepRemoveSoftware,
        GetInstallerCatalog,
        GetInstalledSoftwareStatus,
        StopInstall,
    } from "../../wailsjs/go/main/App.js";
    import { EventsOn, EventsOff } from "../../wailsjs/runtime/runtime.js";
    import { pushLog } from "../stores/log.js";

    // STATE
    let activeTab = "Basic"; // "All" | "Basic" | "Developer" | "Enterprise"
    let searchQuery = "";
    let scanning = false;
    let installing = false;
    let scanned = false;
    let showLog = false;
    let liveLog = [];
    let logEl = null;

    let catalog = [];
    let installedSet = {};
    let checked = {};
    let opState = {}; // { [id]: "installing" | "removing" | "done" | "error" }
    let progressMap = {}; // { [id]: number }
    let currentAppId = "";

    // Icons map
    const ICONS = {
        chrome: "🌐",
        vlc: "▶️",
        lightshot: "📸",
        "7zip": "📦",
        npp: "📝",
        vscode: "💻",
        winrar: "📚",
        zoom: "📹",
        git: "🌿",
        jdk23: "☕",
        nodejs: "🟩",
        nvm: "♻️",
        python: "🐍",
        postman: "📬",
        mysql: "🐬",
        iis10: "🌍",
        maven: "📐",
        filezilla: "📂",
        dbeaver: "🛢️",
        docker: "🐳",
        office2019: "🏢",
        activation: "🔑",
        vs2022: "🔷",
        sql2019: "🗄️",
        sql2022: "🗃️",
        ssms: "🖥️",
        mongodb: "🍃",
        sqlyog: "🔶",
        redis: "🔴",
        rabbitmq: "🐇",
        elasticsearch: "🔍",
        erlang: "⚙️",
        python2: "🐍",
    };

    // Filtered apps based on Tab and Search
    $: filteredApps = catalog.filter((app) => {
        const matchesTab =
            activeTab === "All" ||
            (app.SubCategory &&
                app.SubCategory.toLowerCase() === activeTab.toLowerCase());
        const q = searchQuery.trim().toLowerCase();
        const matchesSearch =
            !q ||
            app.Name.toLowerCase().includes(q) ||
            (app.Description && app.Description.toLowerCase().includes(q)) ||
            (app.Version && app.Version.toLowerCase().includes(q));
        return matchesTab && matchesSearch;
    });

    // Counts
    $: selectedApps = filteredApps.filter((a) => checked[a.ID]);
    $: selectedCount = selectedApps.length;
    $: totalInstalled = catalog.filter((a) => installedSet[a.ID]).length;
    $: tabInstalled = filteredApps.filter((a) => installedSet[a.ID]).length;

    // WAILS EVENT LISTENERS
    let scrollPending = false;
    function scheduleScroll() {
        if (scrollPending) return;
        scrollPending = true;
        requestAnimationFrame(() => {
            if (logEl) logEl.scrollTop = logEl.scrollHeight;
            scrollPending = false;
        });
    }

    function setupEvents() {
        EventsOn("installer:log", (ev) => {
            liveLog = [...liveLog, { text: ev.text, ok: ev.ok }];
            if (liveLog.length > 300) liveLog = liveLog.slice(-300);
            scheduleScroll();
        });
    }

    function teardownEvents() {
        EventsOff("installer:log");
    }

    // LOAD CATALOG & SCAN
    async function loadCatalog() {
        try {
            catalog = await GetInstallerCatalog();
        } catch (e) {
            pushLog(`Failed to load software catalog: ${e}`, false);
        }
    }

    async function scanSystem() {
        if (scanning) return;
        scanning = true;
        try {
            const res = await GetInstalledSoftwareStatus();
            installedSet = res || {};
            scanned = true;
        } catch (e) {
            pushLog(`Software scan error: ${e}`, false);
        } finally {
            scanning = false;
        }
    }

    // SELECTION HELPERS
    function selectAllInTab() {
        filteredApps.forEach((a) => (checked[a.ID] = true));
        checked = { ...checked };
    }

    function clearSelection() {
        checked = {};
    }

    function toggleSelection(id) {
        checked[id] = !checked[id];
        checked = { ...checked };
    }

    // STOP
    async function stopInstall() {
        try {
            await StopInstall();
            liveLog = [
                ...liveLog,
                { text: "Installation stopped by user.", ok: false },
            ];
        } catch (e) {}
        installing = false;
        currentAppId = "";
        opState = {};
        progressMap = {};
    }

    // SINGLE INSTALL
    async function installOne(id) {
        const app = catalog.find((a) => a.ID === id);
        if (!app) return false;
        currentAppId = id;
        opState[id] = "installing";
        progressMap[id] = 10;
        opState = { ...opState };
        progressMap = { ...progressMap };
        showLog = true;

        liveLog = [
            ...liveLog,
            { text: `\n-- Installing ${app.Name} --`, ok: true },
        ];

        // Smooth progress animation
        const progTimer = setInterval(() => {
            if (progressMap[id] < 85) {
                progressMap[id] += 15;
                progressMap = { ...progressMap };
            }
        }, 500);

        let success = false;
        try {
            await InstallSoftware(id);
            clearInterval(progTimer);
            progressMap[id] = 100;
            installedSet[id] = true;
            opState[id] = "done";
            installedSet = { ...installedSet };
            pushLog(`Installed: ${app.Name}`, true);
            liveLog = [
                ...liveLog,
                { text: `✅ Installed: ${app.Name}`, ok: true },
            ];
            success = true;
        } catch (e) {
            clearInterval(progTimer);
            opState[id] = "error";
            pushLog(`Failed: ${app.Name} — ${e}`, false);
            liveLog = [
                ...liveLog,
                { text: `❌ Failed: ${app.Name} — ${e}`, ok: false },
            ];
        } finally {
            clearInterval(progTimer);
            opState = { ...opState };
            progressMap = { ...progressMap };
            currentAppId = "";
            setTimeout(() => {
                delete progressMap[id];
                opState[id] = "";
                opState = { ...opState };
                progressMap = { ...progressMap };
            }, 3000);
            scheduleScroll();
        }
        return success;
    }

    // SINGLE REMOVE
    async function removeOne(id) {
        const app = catalog.find((a) => a.ID === id);
        if (!app) return;
        if (!confirm(`Are you sure you want to remove ${app.Name}?`)) return;

        currentAppId = id;
        opState[id] = "removing";
        opState = { ...opState };
        showLog = true;
        liveLog = [
            ...liveLog,
            { text: `\n-- Removing ${app.Name} --`, ok: true },
        ];

        try {
            await UninstallSoftware(id);
            installedSet[id] = false;
            installedSet = { ...installedSet };
            opState[id] = "done";
            pushLog(`Removed: ${app.Name}`, true);
            liveLog = [...liveLog, { text: `✅ Removed: ${app.Name}`, ok: true }];
        } catch (e) {
            opState[id] = "error";
            pushLog(`Remove failed: ${app.Name} — ${e}`, false);
            liveLog = [
                ...liveLog,
                { text: `❌ Remove failed: ${app.Name} — ${e}`, ok: false },
            ];
        } finally {
            opState = { ...opState };
            currentAppId = "";
            setTimeout(() => {
                opState[id] = "";
                opState = { ...opState };
            }, 3000);
            scheduleScroll();
        }
    }

    // BULK INSTALL (Sequential, One by One with Live Log)
    async function handleBulkAction(actionType = "install") {
        if (installing) return;
        const targetApps = filteredApps.filter((a) => checked[a.ID]);
        if (targetApps.length === 0) return;

        installing = true;
        showLog = true;
        liveLog = [
            {
                text: `Starting bulk ${actionType} of ${targetApps.length} task(s)...`,
                ok: true,
            },
        ];

        let successCount = 0;
        for (let i = 0; i < targetApps.length; i++) {
            if (!installing) break;
            const app = targetApps[i];
            liveLog = [
                ...liveLog,
                {
                    text: `\n[${i + 1}/${targetApps.length}] Processing: ${app.Name}...`,
                    ok: true,
                },
            ];

            if (actionType === "install") {
                const ok = await installOne(app.ID);
                if (ok) successCount++;
            } else {
                try {
                    await UninstallSoftware(app.ID);
                    installedSet[app.ID] = false;
                    installedSet = { ...installedSet };
                    successCount++;
                    liveLog = [
                        ...liveLog,
                        { text: `✅ Removed: ${app.Name}`, ok: true },
                    ];
                } catch (e) {
                    liveLog = [
                        ...liveLog,
                        { text: `❌ Remove failed: ${app.Name} — ${e}`, ok: false },
                    ];
                }
            }
        }

        installing = false;
        currentAppId = "";
        clearSelection();
        await scanSystem();

        liveLog = [
            ...liveLog,
            {
                text: `\n✨ Bulk ${actionType} finished: ${successCount}/${targetApps.length} successful.`,
                ok: successCount === targetApps.length,
            },
        ];
        scheduleScroll();
    }

    onMount(async () => {
        setupEvents();
        await loadCatalog();
        await scanSystem();
    });

    onDestroy(() => teardownEvents());
</script>

<div class="inst-container">
    <!-- TOP HEADER -->
    <header class="inst-header">
        <div class="title-group">
            <h1 class="page-title">Software Install</h1>
            <p class="page-subtitle">Automated workflows for Triveni Group.</p>
        </div>

        <div class="header-right">
            <!-- Search input -->
            <div class="search-box">
                <span class="search-icon">🔍</span>
                <input
                    type="text"
                    bind:value={searchQuery}
                    placeholder="Find task / software..."
                    class="search-input"
                />
                {#if searchQuery}
                    <button class="clear-search-btn" on:click={() => (searchQuery = "")}
                        >✕</button
                    >
                {/if}
            </div>

            <!-- Log Button -->
            <button
                class="icon-action-btn"
                on:click={() => (showLog = !showLog)}
                title="Toggle Console Log"
            >
                📋
            </button>

            <!-- Refresh Button -->
            <button
                class="icon-action-btn"
                class:spinning={scanning}
                on:click={scanSystem}
                disabled={scanning || installing}
                title="Refresh Status"
            >
                🔄
            </button>
        </div>
    </header>

    <!-- SUB-TABS NAVIGATION -->
    <nav class="subtabs-bar">
        {#each ["Basic", "Developer", "Enterprise", "All"] as tab}
            <button
                class="subtab-btn"
                class:active={activeTab === tab}
                on:click={() => {
                    activeTab = tab;
                    clearSelection();
                }}
            >
                {tab}
                <span class="subtab-count">
                    {#if tab === "All"}
                        {totalInstalled}/{catalog.length}
                    {:else}
                        {catalog.filter((a) => a.SubCategory === tab && installedSet[a.ID]).length}/{catalog.filter((a) => a.SubCategory === tab).length}
                    {/if}
                </span>
            </button>
        {/each}
    </nav>

    <!-- BULK ACTIONS BAR (Matches Screenshot) -->
    <div class="bulk-actions-bar">
        <button class="text-btn" on:click={selectAllInTab}>
            SELECT ALL IN {activeTab.toUpperCase()}
        </button>
        <button
            class="text-btn"
            on:click={clearSelection}
            disabled={selectedCount === 0}
        >
            CLEAR SELECTION
        </button>

        {#if selectedCount > 0}
            <div class="selection-count">
                {selectedCount} TASKS SELECTED
            </div>
            <button
                class="btn-bulk-install"
                on:click={() => handleBulkAction("install")}
                disabled={installing}
            >
                📥 INSTALL ALL ({selectedCount})
            </button>
            <button
                class="btn-bulk-remove"
                on:click={() => handleBulkAction("uninstall")}
                disabled={installing}
            >
                🗑️ REMOVE ALL ({selectedCount})
            </button>
        {/if}
    </div>

    <!-- MAIN CARDS VIEW + LOG PANEL -->
    <div class="main-body-layout">
        <!-- Software Cards List -->
        <div class="cards-list-wrap">
            <div class="section-title-label">
                {activeTab.toUpperCase()}
            </div>

            {#if scanning && !scanned}
                <div class="loading-state">
                    <span class="spin-huge">⟳</span>
                    <p>Scanning installed software...</p>
                </div>
            {:else if filteredApps.length === 0}
                <div class="empty-state">
                    <p>No software found matching "{searchQuery}"</p>
                </div>
            {:else}
                <div class="software-cards-column">
                    {#each filteredApps as app (app.ID)}
                        {@const isInst = !!installedSet[app.ID]}
                        {@const isBusy =
                            opState[app.ID] === "installing" ||
                            opState[app.ID] === "removing"}
                        {@const isSelected = !!checked[app.ID]}
                        {@const prog = progressMap[app.ID] || 0}

                        <!-- Horizontal Row Card (Matches Screenshot) -->
                        <div
                            class="software-card"
                            class:selected={isSelected}
                            class:installed={isInst}
                            class:busy={isBusy}
                            on:click={() => toggleSelection(app.ID)}
                        >
                            <!-- Left Pill Badge -->
                            <div class="badge-tag">
                                {app.Category || "Software install"}
                            </div>

                            <!-- Status Circle Checkmark -->
                            <div class="status-indicator">
                                {#if isInst}
                                    <div class="check-circle-green">✓</div>
                                {:else if isSelected}
                                    <div class="check-circle-selected">✓</div>
                                {:else}
                                    <div class="checkbox-circle"></div>
                                {/if}
                            </div>

                            <!-- App Icon & Title -->
                            <div class="card-icon">{ICONS[app.ID] || "📦"}</div>
                            <div class="card-title-group">
                                <h3 class="app-name">{app.Name}</h3>
                                {#if app.Description}
                                    <p class="app-desc">{app.Description}</p>
                                {/if}
                            </div>

                            <!-- Version Tag -->
                            <div class="version-tag">
                                {app.Version || "LATEST"}
                            </div>

                            <!-- Action Buttons on Right -->
                            <div
                                class="card-actions-group"
                                on:click|stopPropagation
                            >
                                {#if isBusy}
                                    <button class="btn-installing" disabled>
                                        <div
                                            class="progress-fill"
                                            style="width: {prog}%;"
                                        ></div>
                                        <span class="btn-label-text">
                                            Running {prog}%
                                        </span>
                                    </button>
                                {:else if isInst}
                                    <!-- Already Installed Button -->
                                    <button
                                        class="btn-already-installed"
                                        on:click={() => installOne(app.ID)}
                                        disabled={installing}
                                        title="Click to Reinstall"
                                    >
                                        Already Installed
                                    </button>

                                    <!-- Remove Button -->
                                    <button
                                        class="btn-remove"
                                        on:click={() => removeOne(app.ID)}
                                        disabled={installing}
                                        title="Uninstall Software"
                                    >
                                        REMOVE
                                    </button>
                                {:else}
                                    <!-- Install Button -->
                                    <button
                                        class="btn-install"
                                        on:click={() => installOne(app.ID)}
                                        disabled={installing}
                                    >
                                        ⚡ Install
                                    </button>
                                {/if}
                            </div>
                        </div>
                    {/each}
                </div>
            {/if}
        </div>

        <!-- RIGHT SIDE: LIVE OUTPUT CONSOLE -->
        {#if showLog}
            <aside class="log-sidebar">
                <div class="log-header">
                    <span class="log-title">📋 Live Execution Output</span>
                    <div class="log-actions">
                        <button
                            class="log-btn"
                            on:click={() => (liveLog = [])}
                            title="Clear">🗑️</button
                        >
                        <button
                            class="log-btn"
                            on:click={() => (showLog = false)}
                            title="Close">✕</button
                        >
                    </div>
                </div>

                <div class="log-content" bind:this={logEl}>
                    {#if liveLog.length === 0}
                        <div class="log-placeholder">
                            Waiting for software installation tasks to run...
                        </div>
                    {:else}
                        {#each liveLog as line}
                            <div class="log-line" class:log-err={!line.ok}>
                                {line.text}
                            </div>
                        {/each}
                    {/if}
                </div>

                <div class="log-footer">
                    <span>{liveLog.length} lines</span>
                    {#if installing}
                        <button class="stop-install-btn" on:click={stopInstall}>
                            ⛔ STOP TASK
                        </button>
                    {/if}
                </div>
            </aside>
        {/if}
    </div>
</div>

<style>
    /* ── BASE CONTAINER ────────────────────────────────────────────── */
    .inst-container {
        display: flex;
        flex-direction: column;
        height: 100%;
        overflow: hidden;
        color: #f8fafc;
        font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
            Oxygen, Ubuntu, Cantarell, "Open Sans", "Helvetica Neue", sans-serif;
        padding: 0 4px;
    }

    /* ── TOP HEADER ────────────────────────────────────────────────── */
    .inst-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 12px;
        flex-shrink: 0;
    }

    .page-title {
        font-size: 1.85rem;
        font-weight: 800;
        letter-spacing: -0.02em;
        margin: 0;
        background: linear-gradient(135deg, #ffffff 0%, #cbd5e1 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .page-subtitle {
        font-size: 0.85rem;
        color: #94a3b8;
        margin: 3px 0 0 0;
    }

    .header-right {
        display: flex;
        align-items: center;
        gap: 10px;
    }

    /* Search input box */
    .search-box {
        position: relative;
        display: flex;
        align-items: center;
        background: rgba(15, 23, 42, 0.7);
        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 10px;
        padding: 0 10px;
        width: 240px;
        transition: all 0.2s ease;
    }

    .search-box:focus-within {
        border-color: #6366f1;
        box-shadow: 0 0 15px rgba(99, 102, 241, 0.25);
        width: 280px;
    }

    .search-icon {
        font-size: 0.8rem;
        opacity: 0.6;
        margin-right: 6px;
    }

    .search-input {
        background: transparent;
        border: none;
        outline: none;
        color: #fff;
        font-size: 0.82rem;
        padding: 7px 0;
        width: 100%;
    }

    .clear-search-btn {
        background: transparent;
        border: none;
        color: #64748b;
        cursor: pointer;
        font-size: 0.75rem;
        padding: 2px 4px;
    }

    .clear-search-btn:hover {
        color: #fff;
    }

    .icon-action-btn {
        background: rgba(15, 23, 42, 0.7);
        border: 1px solid rgba(255, 255, 255, 0.08);
        color: #cbd5e1;
        width: 36px;
        height: 36px;
        border-radius: 10px;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        transition: all 0.2s;
        font-size: 0.95rem;
    }

    .icon-action-btn:hover:not(:disabled) {
        background: rgba(99, 102, 241, 0.2);
        border-color: #6366f1;
        color: #fff;
        transform: translateY(-2px);
    }

    .icon-action-btn.spinning {
        animation: spin 1s linear infinite;
    }

    @keyframes spin {
        from {
            transform: rotate(0deg);
        }
        to {
            transform: rotate(360deg);
        }
    }

    /* ── SUBTABS NAVIGATION ────────────────────────────────────────── */
    .subtabs-bar {
        display: flex;
        gap: 6px;
        margin-bottom: 12px;
        flex-shrink: 0;
    }

    .subtab-btn {
        background: rgba(255, 255, 255, 0.03);
        border: 1px solid rgba(255, 255, 255, 0.06);
        color: #94a3b8;
        font-size: 0.8rem;
        font-weight: 600;
        padding: 6px 16px;
        border-radius: 8px;
        cursor: pointer;
        transition: all 0.2s ease;
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .subtab-btn:hover {
        background: rgba(255, 255, 255, 0.07);
        color: #fff;
    }

    .subtab-btn.active {
        background: rgba(99, 102, 241, 0.15);
        border-color: rgba(99, 102, 241, 0.4);
        color: #a5b4fc;
        font-weight: 700;
    }

    .subtab-count {
        font-size: 0.65rem;
        background: rgba(255, 255, 255, 0.08);
        padding: 1px 6px;
        border-radius: 10px;
    }

    .subtab-btn.active .subtab-count {
        background: rgba(99, 102, 241, 0.3);
        color: #fff;
    }

    /* ── BULK ACTIONS BAR (Matches Screenshot) ─────────────────────── */
    .bulk-actions-bar {
        background: rgba(15, 23, 42, 0.5);
        padding: 8px 16px;
        border-radius: 10px;
        border: 1px solid rgba(255, 255, 255, 0.05);
        display: flex;
        align-items: center;
        gap: 14px;
        margin-bottom: 12px;
        flex-shrink: 0;
    }

    .text-btn {
        background: transparent;
        border: none;
        color: #94a3b8;
        font-size: 0.72rem;
        font-weight: 700;
        letter-spacing: 0.8px;
        cursor: pointer;
        padding: 4px 8px;
        border-radius: 6px;
        transition: all 0.2s;
    }

    .text-btn:hover:not(:disabled) {
        color: #a5b4fc;
        background: rgba(99, 102, 241, 0.1);
    }

    .text-btn:disabled {
        opacity: 0.3;
        cursor: not-allowed;
    }

    .selection-count {
        margin-left: auto;
        font-size: 0.7rem;
        font-weight: 800;
        color: #a5b4fc;
        background: rgba(99, 102, 241, 0.15);
        border: 1px solid rgba(99, 102, 241, 0.3);
        padding: 4px 12px;
        border-radius: 20px;
        letter-spacing: 0.5px;
    }

    .btn-bulk-install {
        background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
        color: #fff;
        border: none;
        padding: 6px 14px;
        border-radius: 8px;
        font-size: 0.75rem;
        font-weight: 700;
        cursor: pointer;
        transition: all 0.2s;
        box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
    }

    .btn-bulk-install:hover:not(:disabled) {
        transform: translateY(-1px);
        filter: brightness(1.1);
    }

    .btn-bulk-remove {
        background: rgba(239, 68, 68, 0.15);
        border: 1px solid rgba(239, 68, 68, 0.4);
        color: #f87171;
        padding: 6px 14px;
        border-radius: 8px;
        font-size: 0.75rem;
        font-weight: 700;
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn-bulk-remove:hover:not(:disabled) {
        background: rgba(239, 68, 68, 0.25);
        color: #fff;
    }

    /* ── MAIN BODY & CARDS ─────────────────────────────────────────── */
    .main-body-layout {
        display: flex;
        gap: 12px;
        flex: 1;
        overflow: hidden;
        min-height: 0;
    }

    .cards-list-wrap {
        flex: 1;
        overflow-y: auto;
        padding-right: 4px;
    }

    .section-title-label {
        font-size: 0.72rem;
        font-weight: 800;
        letter-spacing: 1.2px;
        color: #64748b;
        margin: 4px 0 10px 4px;
    }

    .software-cards-column {
        display: flex;
        flex-direction: column;
        gap: 10px;
        padding-bottom: 20px;
    }

    /* ── HORIZONTAL SOFTWARE CARD (Matches Screenshot) ─────────────── */
    .software-card {
        background: rgba(15, 23, 42, 0.6);
        border: 1px solid rgba(255, 255, 255, 0.07);
        border-radius: 12px;
        padding: 14px 20px;
        display: flex;
        flex-direction: row;
        align-items: center;
        gap: 16px;
        cursor: pointer;
        transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
        backdrop-filter: blur(12px);
        position: relative;
    }

    .software-card:hover {
        background: rgba(20, 30, 55, 0.75);
        border-color: rgba(255, 255, 255, 0.15);
        transform: translateX(4px);
    }

    .software-card.selected {
        border-color: rgba(139, 92, 246, 0.6);
        background: rgba(99, 102, 241, 0.08);
        box-shadow: 0 0 15px rgba(99, 102, 241, 0.15);
    }

    .software-card.installed {
        border-left: 3px solid #10b981;
    }

    .software-card.busy {
        border-color: #eab308;
    }

    /* Left pill tag */
    .badge-tag {
        font-size: 0.68rem;
        font-weight: 600;
        padding: 4px 10px;
        background: rgba(99, 102, 241, 0.1);
        color: #818cf8;
        border-radius: 20px;
        border: 1px solid rgba(99, 102, 241, 0.2);
        white-space: nowrap;
        flex-shrink: 0;
    }

    /* Status Circle Checkmark */
    .status-indicator {
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
    }

    .check-circle-green {
        width: 22px;
        height: 22px;
        border-radius: 50%;
        border: 1.5px solid #10b981;
        color: #10b981;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 0.75rem;
        font-weight: 900;
        background: rgba(16, 185, 129, 0.1);
    }

    .check-circle-selected {
        width: 22px;
        height: 22px;
        border-radius: 50%;
        border: 1.5px solid #8b5cf6;
        color: #fff;
        background: #8b5cf6;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 0.75rem;
        font-weight: 900;
    }

    .checkbox-circle {
        width: 20px;
        height: 20px;
        border-radius: 50%;
        border: 1.5px solid rgba(255, 255, 255, 0.2);
        transition: all 0.2s;
    }

    .card-icon {
        font-size: 1.2rem;
        flex-shrink: 0;
    }

    .card-title-group {
        flex: 1;
        min-width: 140px;
    }

    .app-name {
        font-size: 1.05rem;
        font-weight: 700;
        margin: 0;
        color: #fff;
        letter-spacing: -0.01em;
    }

    .app-desc {
        font-size: 0.72rem;
        color: #64748b;
        margin: 2px 0 0 0;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        max-width: 350px;
    }

    /* Version Tag */
    .version-tag {
        font-size: 0.65rem;
        font-weight: 700;
        padding: 3px 8px;
        background: rgba(255, 255, 255, 0.04);
        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 5px;
        color: #94a3b8;
        letter-spacing: 0.5px;
        text-transform: uppercase;
        flex-shrink: 0;
    }

    /* ── CARD BUTTONS (Matches Screenshot) ─────────────────────────── */
    .card-actions-group {
        display: flex;
        align-items: center;
        gap: 10px;
        margin-left: auto;
        flex-shrink: 0;
    }

    /* Purple "Already Installed" Button */
    .btn-already-installed {
        background: linear-gradient(135deg, #6366f1 0%, #a855f7 100%);
        color: #fff;
        border: none;
        padding: 8px 24px;
        border-radius: 8px;
        font-size: 0.85rem;
        font-weight: 700;
        cursor: pointer;
        transition: all 0.25s;
        box-shadow: 0 4px 15px rgba(99, 102, 241, 0.25);
        white-space: nowrap;
        min-width: 160px;
        text-align: center;
    }

    .btn-already-installed:hover:not(:disabled) {
        transform: scale(1.02);
        filter: brightness(1.1);
        box-shadow: 0 6px 20px rgba(168, 85, 247, 0.4);
    }

    /* Standard Install Button */
    .btn-install {
        background: linear-gradient(135deg, #3b82f6 0%, #6366f1 100%);
        color: #fff;
        border: none;
        padding: 8px 24px;
        border-radius: 8px;
        font-size: 0.85rem;
        font-weight: 700;
        cursor: pointer;
        transition: all 0.25s;
        box-shadow: 0 4px 15px rgba(59, 130, 246, 0.25);
        white-space: nowrap;
        min-width: 140px;
        text-align: center;
    }

    .btn-install:hover:not(:disabled) {
        transform: scale(1.02);
        filter: brightness(1.1);
        box-shadow: 0 6px 20px rgba(59, 130, 246, 0.4);
    }

    /* Red Remove Button */
    .btn-remove {
        background: rgba(239, 68, 68, 0.12);
        border: 1px solid rgba(239, 68, 68, 0.35);
        color: #f87171;
        padding: 8px 18px;
        border-radius: 8px;
        font-size: 0.8rem;
        font-weight: 700;
        letter-spacing: 0.5px;
        cursor: pointer;
        transition: all 0.2s;
        white-space: nowrap;
    }

    .btn-remove:hover:not(:disabled) {
        background: rgba(239, 68, 68, 0.25);
        border-color: #ef4444;
        color: #fff;
    }

    /* Installing progress button */
    .btn-installing {
        position: relative;
        background: #1e1b4b;
        border: 1px solid #6366f1;
        color: #fff;
        padding: 8px 24px;
        border-radius: 8px;
        font-size: 0.85rem;
        font-weight: 700;
        overflow: hidden;
        min-width: 160px;
        cursor: wait;
    }

    .progress-fill {
        position: absolute;
        left: 0;
        top: 0;
        bottom: 0;
        background: linear-gradient(135deg, #6366f1, #a855f7);
        transition: width 0.3s ease;
        z-index: 1;
    }

    .btn-label-text {
        position: relative;
        z-index: 2;
    }

    /* ── LIVE OUTPUT LOG SIDEBAR ───────────────────────────────────── */
    .log-sidebar {
        width: 380px;
        background: #090d16;
        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 12px;
        display: flex;
        flex-direction: column;
        overflow: hidden;
        flex-shrink: 0;
    }

    .log-header {
        padding: 10px 14px;
        background: rgba(255, 255, 255, 0.03);
        border-bottom: 1px solid rgba(255, 255, 255, 0.06);
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .log-title {
        font-size: 0.78rem;
        font-weight: 700;
        color: #cbd5e1;
    }

    .log-actions {
        display: flex;
        gap: 6px;
    }

    .log-btn {
        background: transparent;
        border: none;
        color: #64748b;
        cursor: pointer;
        font-size: 0.85rem;
        padding: 2px 6px;
        border-radius: 4px;
    }

    .log-btn:hover {
        color: #fff;
        background: rgba(255, 255, 255, 0.1);
    }

    .log-content {
        flex: 1;
        padding: 10px 12px;
        overflow-y: auto;
        font-family: "Consolas", "Courier New", monospace;
        font-size: 0.75rem;
        line-height: 1.45;
        background: #06090e;
    }

    .log-placeholder {
        color: #475569;
        font-style: italic;
        padding: 10px 0;
    }

    .log-line {
        color: #94a3b8;
        white-space: pre-wrap;
        word-break: break-all;
    }

    .log-line.log-err {
        color: #f87171;
    }

    .log-footer {
        padding: 8px 12px;
        background: rgba(255, 255, 255, 0.02);
        border-top: 1px solid rgba(255, 255, 255, 0.06);
        display: flex;
        justify-content: space-between;
        align-items: center;
        font-size: 0.7rem;
        color: #64748b;
    }

    .stop-install-btn {
        background: rgba(239, 68, 68, 0.2);
        border: 1px solid #ef4444;
        color: #f87171;
        font-weight: 700;
        font-size: 0.7rem;
        padding: 3px 8px;
        border-radius: 4px;
        cursor: pointer;
    }

    .stop-install-btn:hover {
        background: #ef4444;
        color: #fff;
    }

    /* ── LOADING & EMPTY STATES ────────────────────────────────────── */
    .loading-state,
    .empty-state {
        padding: 60px 0;
        text-align: center;
        color: #64748b;
    }

    .spin-huge {
        font-size: 2rem;
        display: inline-block;
        animation: spin 1s linear infinite;
        color: #6366f1;
        margin-bottom: 10px;
    }
</style>
