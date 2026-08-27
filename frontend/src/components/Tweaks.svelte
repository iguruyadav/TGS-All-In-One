<script>
    import { onMount } from "svelte";
    import {
        GetPerformanceStatus,
        ConfigurePageFile,
        GetStartupApps,
        ToggleStartupApp,
        GetSystemHealth,
        InstallGlobalRamOptimizer,
        TestRamOptimizer,
        InstallStartupCleanup,
        ExecuteDevAction,
    } from "../../wailsjs/go/main/App";
    import { pushLog } from "../stores/log.js";

    let activeSubTab = "sys";

    let status = {
        PageFileSizeMB: 0,
        PageFileManaged: true,
        GlobalRamOptimizerEnabled: false,
        StartupCleanupEnabled: false,
    };
    let startupApps = [];
    let pageSizeMB = 4096;
    let pageSizeMaxMB = 4096;
    let pageFileManaged = true;
    let feedback = "";
    let loading = false;
    let startupLoading = false;
    let health = { StabilityScore: 10, LastFailures: [], CriticalLogs: [] };
    let healthLoading = false;

    onMount(async () => {
        await loadAll();
    });

    // ── STARTUP IMPACT RATING ────────────────────────────────────────────────────
    const IMPACT_HIGH = ['OneDrive','Teams','MicrosoftTeams','Zoom','Slack','Discord',
        'GoogleDriveFS','Dropbox','iCloudServices','Adobe','Acrobat',
        'SteamBootstrapper','EpicGamesLauncher','Skype'];
    const IMPACT_LOW  = ['WindowsDefender','SecurityHealthSystray','ctfmon',
        'RealTek','RtkNGUI64','IntelligentDisplay','HDAudBus'];

    function getImpact(name) {
        const n = name || '';
        if (IMPACT_HIGH.some(h => n.toLowerCase().includes(h.toLowerCase()))) return 'High';
        if (IMPACT_LOW.some(l  => n.toLowerCase().includes(l.toLowerCase())))  return 'Low';
        return 'Medium';
    }
    function impactColor(impact) {
        if (impact === 'High')   return '#ef4444';
        if (impact === 'Medium') return '#f59e0b';
        return '#10b981';
    }

    async function loadAll() {
        try {
            status = await GetPerformanceStatus();
            pageFileManaged = status.PageFileManaged;
            pageSizeMB = status.PageFileSizeMB || 4096;
            pageSizeMaxMB = status.PageFileMaxMB || 4096;
            startupApps = (await GetStartupApps()) || [];
            await loadHealth();
        } catch (e) {
            console.error("Load error:", e);
        }
    }

    async function loadHealth() {
        healthLoading = true;
        try {
            health = await GetSystemHealth();
        } catch (e) {
            console.error("Health error:", e);
        } finally {
            healthLoading = false;
        }
    }

    async function act(fn, ...args) {
        loading = true;
        feedback = "";
        pushLog(`▶ Starting optimizer task…`);
        try {
            const msg = await fn(...args);
            feedback = msg;
            pushLog(`✓ ${msg}`, true);
            await loadAll();
        } catch (e) {
            feedback = "Error: " + e;
            pushLog(`✗ Error: ${e}`, false);
        } finally {
            loading = false;
            setTimeout(() => (feedback = ""), 5000);
        }
    }

    async function toggleStartup(app) {
        startupLoading = true;
        const newState = !app.Enabled;
        pushLog(
            `▶ ${newState ? "Enabling" : "Disabling"} startup app: ${app.Name}…`,
        );
        try {
            const msg = await ToggleStartupApp(app.Name, app.Source, newState);
            feedback = msg;
            pushLog(`✓ ${msg}`, true);
            startupApps = await GetStartupApps();
        } catch (e) {
            feedback = "Error: " + e;
            pushLog(`✗ Error: ${e}`, false);
        } finally {
            startupLoading = false;
            setTimeout(() => (feedback = ""), 4000);
        }
    }
</script>

<div class="page">
    <header class="page-header">
        <div>
            <h1 class="page-title">Optimizer</h1>
            <p class="page-desc">
                Performance tuning, virtual memory, and startup management
            </p>
        </div>
        {#if healthLoading}
            <div class="health-mini-load">Analyzing Health…</div>
        {:else}
            <div class="health-mini">
                <span class="h-score" class:bad={health.StabilityScore < 7}
                    >{health.StabilityScore.toFixed(1)}/10</span
                >
                <span class="h-lbl">Stability Index</span>
            </div>
        {/if}
    </header>

    <div class="sub-tabs">
        <button class="sub-tab {activeSubTab === 'sys' ? 'active' : ''}" on:click={() => activeSubTab = 'sys'}>System Tweaks</button>
        <button class="sub-tab {activeSubTab === 'dev' ? 'active' : ''}" on:click={() => activeSubTab = 'dev'}>Developer & VS Code</button>
    </div>

    {#if activeSubTab === 'sys'}
    <div class="content">
        <!-- SECTION: Virtual Memory -->
        <div class="divider"></div>
        <p class="section-label">Virtual Memory (PageFile)</p>
        <div class="tweaks-grid">
            <div class="card tweak-card full">
                <div class="tweak-row">
                    <div class="tweak-info">
                        <span class="tweak-name">Virtual Memory (PageFile)</span
                        >
                        <span class="tweak-desc"
                            >Fixed size for predictable performance</span
                        >
                    </div>
                    <span class={pageFileManaged ? "tag-neutral" : "tag-accent"}
                        >{pageFileManaged
                            ? "System Managed"
                            : `${pageSizeMB} MB / ${pageSizeMaxMB} MB`}</span
                    >
                </div>
                <div class="pf-controls">
                    <label class="check-label">
                        <input type="checkbox" bind:checked={pageFileManaged} />
                        <span>System Managed</span>
                    </label>
                    {#if !pageFileManaged}
                        <div class="pf-inputs-row">
                            <div class="pf-col">
                                <span class="pf-lbl">Initial Size:</span>
                                <div class="pf-row">
                                    <input
                                        type="number"
                                        class="input pf-input"
                                        bind:value={pageSizeMB}
                                        min="512"
                                        max="65536"
                                        step="512"
                                    />
                                    <span class="unit-label">MB</span>
                                </div>
                            </div>
                            <div class="pf-col">
                                <span class="pf-lbl">Maximum Size:</span>
                                <div class="pf-row">
                                    <input
                                        type="number"
                                        class="input pf-input"
                                        bind:value={pageSizeMaxMB}
                                        min="512"
                                        max="65536"
                                        step="512"
                                    />
                                    <span class="unit-label">MB</span>
                                </div>
                            </div>
                        </div>
                    {/if}
                    <div class="pf-actions">
                        <button
                            class="btn btn-primary"
                            on:click={() =>
                                act(
                                    ConfigurePageFile,
                                    pageFileManaged ? 0 : pageSizeMB,
                                    pageFileManaged ? 0 : pageSizeMaxMB,
                                )}
                            disabled={loading}>Apply</button
                        >
                        <button
                            class="btn-ghost"
                            on:click={() => {
                                pageFileManaged = true;
                                act(ConfigurePageFile, 0, 0);
                            }}>Restore Default</button
                        >
                    </div>
                    <p class="warn-note">
                        ⚠ A reboot is required for PageFile changes to take
                        effect.
                    </p>
                    <div class="pf-tip-box">
                        <p class="pf-tip-title">
                            💡 Virtual Memory Best Practices
                        </p>
                        <ul class="pf-tip-list">
                            <li>
                                <strong>Initial size (MB):</strong> Set this to 1.5x
                                your physical RAM.
                            </li>
                            <li>
                                <strong>Maximum size (MB):</strong> Set this to 3x
                                your physical RAM.
                            </li>
                        </ul>
                        <table class="pf-table">
                            <thead>
                                <tr
                                    ><th>Physical RAM</th><th
                                        >Initial Size (Min)</th
                                    ><th>Maximum Size (Max)</th></tr
                                >
                            </thead>
                            <tbody>
                                <tr
                                    ><td>8 GB</td><td>12288 MB</td><td
                                        >24576 MB</td
                                    ></tr
                                >
                                <tr
                                    ><td>16 GB</td><td>24576 MB</td><td
                                        >49152 MB</td
                                    ></tr
                                >
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>

        <!-- SMART OPTIMIZATION TASKS -->
        <div class="divider"></div>
        <p class="section-label">Smart Optimization</p>
        <div class="tweaks-grid">
            <!-- Global RAM Optimizer -->
            <div class="card tweak-card">
                <div class="tweak-row">
                    <div class="tweak-info">
                        <span class="tweak-name"
                            >Global RAM Optimizer Service</span
                        >
                        <span class="tweak-desc"
                            >Silently trims idle memory from apps in the
                            background</span
                        >
                    </div>
                    <button
                        class="toggle-btn"
                        class:on={status.GlobalRamOptimizerEnabled}
                        on:click={() =>
                            act(
                                InstallGlobalRamOptimizer,
                                !status.GlobalRamOptimizerEnabled,
                            )}
                        disabled={loading}
                    >
                        <span class="toggle-knob"></span>
                    </button>
                </div>
                <div class="tweak-foot">
                    <span
                        class={status.GlobalRamOptimizerEnabled
                            ? "tag-on"
                            : "tag-off"}
                        >{status.GlobalRamOptimizerEnabled
                            ? "Running"
                            : "Stopped"}</span
                    >
                    <button
                        class="btn-ghost"
                        on:click={() => act(TestRamOptimizer)}
                        disabled={loading}>Test Flow</button
                    >
                </div>
            </div>

            <!-- Startup Cleanup -->
            <div class="card tweak-card">
                <div class="tweak-row">
                    <div class="tweak-info">
                        <span class="tweak-name">Startup Cleanup Task</span>
                        <span class="tweak-desc"
                            >Automatically clears system temp files and
                            optimizes on boot</span
                        >
                    </div>
                    <button
                        class="toggle-btn"
                        class:on={status.StartupCleanupEnabled}
                        on:click={() =>
                            act(
                                InstallStartupCleanup,
                                !status.StartupCleanupEnabled,
                            )}
                        disabled={loading}
                    >
                        <span class="toggle-knob"></span>
                    </button>
                </div>
                <div class="tweak-foot">
                    <span
                        class={status.StartupCleanupEnabled
                            ? "tag-on"
                            : "tag-off"}
                        >{status.StartupCleanupEnabled
                            ? "Enabled"
                            : "Disabled"}</span
                    >
                </div>
            </div>
        </div>

        <!-- STARTUP -->
        <div class="divider"></div>
        <p class="section-label">Startup Manager</p>
        <div class="startup-table">
            {#if startupApps.length === 0}
                <div class="empty-msg">No startup apps found.</div>
            {:else}
                <div class="table-head">
                    <span>Application</span>
                    <span>Source</span>
                    <span>Impact</span>
                    <span>Status</span>
                    <span>Action</span>
                </div>
                {#each startupApps as app}
                    {@const impact = getImpact(app.Name)}
                    <div class="table-row" class:dimmed={!app.Enabled}>
                        <div class="app-col">
                            <span class="app-name">{app.Name}</span>
                            <span class="app-cmd">{app.Command.slice(0, 55)}{app.Command.length > 55 ? '…' : ''}</span>
                        </div>
                        <span class="src-tag tag-neutral">{app.Source}</span>
                        <span class="impact-badge" style="color:{impactColor(impact)};background:{impactColor(impact)}22">{impact}</span>
                        <span class={app.Enabled ? "tag-on" : "tag-off"}>{app.Enabled ? "Enabled" : "Disabled"}</span>
                        <button class="btn-ghost" on:click={() => toggleStartup(app)} disabled={startupLoading}>
                            {app.Enabled ? "Disable" : "Enable"}
                        </button>
                    </div>
                {/each}
            {/if}
        </div>
    </div>
    {/if}

    {#if activeSubTab === 'dev'}
    <div class="content dev-grid">
        <!-- SYSTEM -->
        <div class="card tweak-card">
            <p class="section-label">System Power</p>
            <button class="btn btn-primary dev-btn" on:click={() => act(ExecuteDevAction, 'power_ultimate')}>🔥 Enable Ultimate Performance</button>
            <button class="btn-ghost dev-btn" on:click={() => act(ExecuteDevAction, 'power_balanced')}>↩ Revert Power Mode</button>
        </div>

        <!-- DEFENDER -->
        <div class="card tweak-card">
            <p class="section-label">Windows Defender</p>
            <button class="btn btn-primary dev-btn" on:click={() => act(ExecuteDevAction, 'defender_exclusions')}>🛡 Add Dev Folder Exclusions</button>
            <button class="btn-ghost dev-btn" on:click={() => act(ExecuteDevAction, 'defender_remove')}>❌ Remove Exclusions</button>
        </div>

        <!-- VS CODE -->
        <div class="card tweak-card">
            <p class="section-label">VS Code</p>
            <button class="btn btn-primary dev-btn" on:click={() => act(ExecuteDevAction, 'vscode_optimize')}>⚡ Optimize settings.json</button>
            <button class="btn-ghost dev-btn" on:click={() => act(ExecuteDevAction, 'vscode_highram')}>🚀 Launch (High RAM)</button>
            <button class="btn-ghost dev-btn" on:click={() => act(ExecuteDevAction, 'vscode_noext')}>⚙ Launch (No Ext)</button>
        </div>

        <!-- VISUAL STUDIO & GIT -->
        <div class="card tweak-card">
            <p class="section-label">Visual Studio & Git</p>
            <button class="btn btn-primary dev-btn" on:click={() => act(ExecuteDevAction, 'vs_multicore')}>💻 Enable Multi-Core Build</button>
            <button class="btn-primary dev-btn" on:click={() => act(ExecuteDevAction, 'git_optimize')}>🌐 Optimize Git Speed</button>
        </div>

        <!-- UTILITIES -->
        <div class="card tweak-card full">
            <p class="section-label">Utilities</p>
            <div style="display: flex; gap: 10px; flex-wrap: wrap;">
                <button class="btn-ghost dev-btn" on:click={() => act(ExecuteDevAction, 'ping_test')}>📡 Test Ping</button>
                <button class="btn-ghost dev-btn" on:click={() => act(ExecuteDevAction, 'taskmgr')}>🧰 Open Task Manager</button>
                <button class="btn-ghost dev-btn" on:click={() => act(ExecuteDevAction, 'restart_explorer')}>🔄 Restart Explorer</button>
                <button class="btn-danger dev-btn" on:click={() => act(ExecuteDevAction, 'kill_chrome')}>💣 Kill Chrome Forcefully</button>
            </div>
        </div>
    </div>
    {/if}

    {#if feedback}
        <div class="toast">{feedback}</div>
    {/if}
</div>

<style>
    .sub-tabs {
        display: flex; gap: 8px; margin-bottom: 16px; border-bottom: 1px solid var(--border); padding-bottom: 8px;
    }
    .sub-tab {
        background: transparent; border: none; font-size: 0.9rem; font-weight: 600; font-family: var(--font);
        color: var(--text-dim); padding: 8px 16px; cursor: pointer; border-radius: 6px; transition: all 0.2s;
    }
    .sub-tab:hover { background: var(--bg-hover); color: var(--text); }
    .sub-tab.active { background: var(--accent-dim); color: var(--accent-text); box-shadow: 0 0 10px rgba(124,58,237,0.2); }
    
    .dev-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
    .dev-btn { width: 100%; margin-bottom: 8px; text-align: left; padding: 12px 18px; display: block; }
    .content {
        flex: 1;
    }
    .tweaks-grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 12px;
        margin-bottom: 4px;
    }
    .tweak-card {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }
    .tweak-card.full {
        grid-column: span 2;
    }
    .tweak-row {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 16px;
    }
    .tweak-info {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }
    .tweak-name {
        font-size: 0.88rem;
        font-weight: 600;
        color: var(--text);
    }
    .tweak-desc {
        font-size: 0.74rem;
        color: var(--text-dim);
    }
    .tweak-foot {
        display: flex;
        align-items: center;
        gap: 10px;
    }

    /* Toggle */
    .toggle-btn {
        position: relative;
        width: 40px;
        height: 22px;
        background: var(--border-md);
        border: none;
        border-radius: 11px;
        cursor: pointer;
        flex-shrink: 0;
        transition: background 0.2s;
    }
    .toggle-btn.on {
        background: var(--accent);
    }
    .toggle-btn:disabled {
        opacity: 0.4;
        cursor: not-allowed;
    }
    .toggle-knob {
        position: absolute;
        top: 3px;
        left: 3px;
        width: 16px;
        height: 16px;
        border-radius: 50%;
        background: #fff;
        transition: transform 0.2s;
    }
    .toggle-btn.on .toggle-knob {
        transform: translateX(18px);
    }

    /* Health UI */
    .health-mini {
        display: flex;
        flex-direction: column;
        align-items: flex-end;
    }
    .h-score {
        font-size: 1.4rem;
        font-weight: 800;
        color: #4ade80;
        text-shadow: 0 0 10px rgba(74, 222, 128, 0.3);
    }
    .h-score.bad {
        color: #f87171;
    }
    .h-lbl {
        font-size: 0.65rem;
        text-transform: uppercase;
        letter-spacing: 1px;
        color: var(--text-dim);
    }
    .health-mini-load {
        font-size: 0.75rem;
        font-style: italic;
        color: var(--text-dim);
    }

    /* PageFile */
    .pf-controls {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }
    .check-label {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 0.84rem;
        color: var(--text-dim);
        cursor: pointer;
    }
    .check-label input {
        accent-color: var(--accent);
        width: 14px;
        height: 14px;
    }
    .pf-inputs-row {
        display: flex;
        gap: 20px;
        margin-top: 4px;
    }
    .pf-col {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }
    .pf-lbl {
        font-size: 0.75rem;
        color: var(--text-dim);
        font-weight: 500;
    }
    .pf-row {
        display: flex;
        align-items: center;
        gap: 8px;
    }
    .pf-input {
        width: 90px;
    }
    .unit-label {
        font-size: 0.8rem;
        color: var(--text-dim);
    }
    .pf-actions {
        display: flex;
        gap: 10px;
        align-items: center;
    }
    .warn-note {
        margin: 0;
        font-size: 0.72rem;
        color: var(--warn);
    }

    .pf-tip-box {
        margin-top: 10px;
        background: rgba(74, 143, 219, 0.08);
        border: 1px solid rgba(74, 143, 219, 0.2);
        border-radius: var(--radius-sm);
        padding: 12px 16px;
        color: var(--text-dim);
    }
    .pf-tip-title {
        margin: 0 0 8px 0;
        font-size: 0.8rem;
        font-weight: 600;
        color: var(--accent-text);
    }
    .pf-tip-list {
        margin: 0 0 10px 0;
        padding-left: 18px;
        font-size: 0.75rem;
        line-height: 1.5;
    }
    .pf-tip-list strong {
        color: var(--text);
    }
    .pf-table {
        width: 100%;
        border-collapse: collapse;
        font-size: 0.7rem;
        background: rgba(0, 0, 0, 0.2);
        border-radius: 4px;
        overflow: hidden;
    }
    .pf-table th,
    .pf-table td {
        padding: 6px 10px;
        text-align: left;
        border-bottom: 1px solid var(--border);
    }
    .pf-table th {
        background: rgba(0, 0, 0, 0.4);
        color: var(--text);
        font-weight: 600;
    }
    .pf-table tbody tr:last-child td {
        border-bottom: none;
    }

    /* Startup Table */
    .startup-table {
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: var(--radius);
        overflow: hidden;
    }
    .table-head {
        display: grid;
        grid-template-columns: 1fr 80px 70px 70px 90px;
        padding: 9px 18px;
        font-size: 0.7rem;
        font-weight: 600;
        color: var(--text-dim);
        text-transform: uppercase;
        letter-spacing: 0.5px;
        background: var(--bg);
        border-bottom: 1px solid var(--border);
    }
    .table-row {
        display: grid;
        grid-template-columns: 1fr 80px 70px 70px 90px;
        align-items: center;
        padding: 10px 18px;
        border-bottom: 1px solid var(--border);
        transition: background 0.12s;
    }
    .table-row:last-child {
        border-bottom: none;
    }
    .table-row:hover {
        background: var(--bg-hover);
    }
    .table-row.dimmed {
        opacity: 0.45;
    }
    .app-col {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }
    .app-name {
        font-size: 0.82rem;
        font-weight: 600;
        color: var(--text);
    }
    .app-cmd {
        font-size: 0.67rem;
        color: var(--text-dim);
    }
    .src-tag {
        font-size: 0.65rem;
    }
    .empty-msg {
        padding: 40px;
        text-align: center;
        font-size: 0.85rem;
        color: var(--text-dim);
    }

    /* Impact badge */
    .impact-badge {
        display: inline-block; font-size: 0.65rem; font-weight: 700;
        padding: 2px 8px; border-radius: 99px;
    }
</style>
