<script>
    import { ApplyAdvancedDebloat } from "../../wailsjs/go/main/App";
    import { pushLog } from "../stores/log.js";

    let step = 0;
    let mode = "default";
    let statusMessage = "";
    let statusOk = true;
    let applying = false;

    let apps = [
        {
            id: "3dviewer",
            name: "3D Viewer",
            desc: "Viewer for 3D models",
            package: "Microsoft.Microsoft3DViewer",
            selected: true,
        },
        {
            id: "alarms",
            name: "Alarms & Clock",
            desc: "Alarms & Clock app",
            package: "Microsoft.WindowsAlarms",
            selected: true,
        },
        {
            id: "bingsearch",
            name: "Bing Search",
            desc: "Web Search from Microsoft Bing",
            package: "Microsoft.BingSearch",
            selected: false,
        },
        {
            id: "calculator",
            name: "Calculator",
            desc: "Calculator app",
            package: "Microsoft.WindowsCalculator",
            selected: false,
        },
        {
            id: "camera",
            name: "Camera",
            desc: "Camera for built-in or connected cameras",
            package: "Microsoft.WindowsCamera",
            selected: false,
        },
        {
            id: "clipchamp",
            name: "Clipchamp",
            desc: "Video editor from Microsoft",
            package: "Clipchamp.Clipchamp",
            selected: true,
        },
        {
            id: "devhome",
            name: "Dev Home",
            desc: "Developer dashboard",
            package: "Microsoft.Windows.DevHome",
            selected: true,
        },
        {
            id: "teams",
            name: "Microsoft Teams",
            desc: "New Microsoft Teams app",
            package: "MSTeams",
            selected: true,
        },
        {
            id: "todo",
            name: "Microsoft To Do",
            desc: "Task management app",
            package: "Microsoft.Todos",
            selected: true,
        },
        {
            id: "onenote",
            name: "OneNote",
            desc: "Digital note-taking app",
            package: "Microsoft.Office.OneNote",
            selected: true,
        },
        {
            id: "powerautomate",
            name: "Power Automate",
            desc: "Desktop automation tool",
            package: "Microsoft.PowerAutomateDesktop",
            selected: true,
        },
        {
            id: "quickassist",
            name: "Quick Assist",
            desc: "Remote assistance tool",
            package: "Microsoft.CorporationII.QuickAssist",
            selected: true,
        },
        {
            id: "stickynotes",
            name: "Sticky Notes",
            desc: "Digital sticky notes",
            package: "Microsoft.MicrosoftStickyNotes",
            selected: true,
        },
        {
            id: "xbox",
            name: "Xbox / Game Bar",
            desc: "Xbox app, Game Bar, Gaming overlay",
            package: "Microsoft.GamingApp",
            selected: true,
        },
        {
            id: "solitaire",
            name: "Solitaire Collection",
            desc: "Microsoft Solitaire, FreeCell, Spider etc.",
            package: "Microsoft.MicrosoftSolitaireCollection",
            selected: true,
        },
        {
            id: "casualgames",
            name: "Casual Games (MSN)",
            desc: "MSN Weather, News, Sports, Money, Food",
            package: "Microsoft.MicrosoftCasualGames",
            selected: true,
        },
        {
            id: "whatsapp",
            name: "WhatsApp",
            desc: "WhatsApp messenger app from Meta",
            package: "WhatsApp.WhatsApp",
            selected: true,
        },
        {
            id: "linkedin",
            name: "LinkedIn",
            desc: "LinkedIn professional network app",
            package: "LinkedIn.LinkedIn",
            selected: true,
        },
    ];


    let tweaks = {
        Privacy: [
            {
                id: "telemetry",
                name: "Disable telemetry, tracking & targeted ads",
                selected: true,
            },
            {
                id: "tips",
                name: "Disable tips, tricks & suggestions",
                selected: true,
            },
            {
                id: "location",
                name: "Disable Windows location services",
                selected: false,
            },
        ],
        "AI Features": [
            {
                id: "copilot",
                name: "Disable Microsoft Copilot",
                selected: true,
            },
            { id: "recall", name: "Disable Windows Recall", selected: true },
            { id: "clicktodo", name: "Disable Click To Do", selected: true },
        ],
        System: [
            { id: "faststartup", name: "Disable Fast Startup", selected: true },
            { id: "dragtray", name: "Disable Drag Tray", selected: true },
        ],
        Explorer: [
            {
                id: "extensions",
                name: "Show file extensions for known types",
                selected: true,
            },
        ],
        Taskbar: [
            {
                id: "widgets",
                name: "Disable widgets on taskbar & lock screen",
                selected: true,
            },
            {
                id: "bingsearch_tb",
                name: "Disable Bing web search on taskbar",
                selected: true,
            },
        ],
    };

    let options = { restorePoint: true, restartExplorer: false };

    const defaultList = [
        "3dviewer",
        "alarms",
        "clipchamp",
        "devhome",
        "teams",
        "todo",
        "onenote",
        "powerautomate",
        "quickassist",
        "stickynotes",
        "xbox",
        "solitaire",
        "casualgames",
        "whatsapp",
        "linkedin",
    ];

    function selectDefaultApps() {
        apps = apps.map((a) => ({
            ...a,
            selected: defaultList.includes(a.id),
        }));
    }
    function clearApps() {
        apps = apps.map((a) => ({ ...a, selected: false }));
    }

    function nextStep() {
        if (mode === "default" && step === 0) {
            selectDefaultApps();
            step = 3;
        } else {
            step++;
        }
    }
    function prevStep() {
        if (mode === "default" && step === 3) {
            step = 0;
        } else {
            step--;
        }
    }

    async function applyChanges() {
        applying = true;
        statusMessage = "Applying changes…";
        statusOk = true;
        const payload = {
            Apps: apps.filter((a) => a.selected).map((a) => a.package),
            Tweaks: Object.values(tweaks)
                .flat()
                .filter((t) => t.selected)
                .map((t) => t.id),
            Options: options,
        };
        try {
            const result = await ApplyAdvancedDebloat(payload);
            statusMessage = "✅ Done! " + result;
            statusOk = true;
            pushLog("[Debloater] ✓ Changes applied: " + result, true);
        } catch (err) {
            statusMessage = "❌ Error: " + err;
            statusOk = false;
            pushLog("[Debloater] ✗ Failed: " + err, false);
        } finally {
            applying = false;
        }
    }

    $: selectedAppsCount = apps.filter((a) => a.selected).length;
    const stepLabels = ["Mode", "Apps", "Tweaks", "Review"];
</script>

<div class="page">
    <header class="page-header">
        <div>
            <h1 class="page-title">Win Debloater</h1>
            <p class="page-desc">
                Remove bloatware and optimize Windows settings
            </p>
        </div>
        <!-- Step progress -->
        <div class="steps">
            {#each stepLabels as lbl, i}
                <div
                    class="step-item {i === step ? 'active' : ''} {i < step
                        ? 'done'
                        : ''}"
                >
                    <span class="step-num">{i + 1}</span>
                    <span class="step-lbl">{lbl}</span>
                </div>
                {#if i < stepLabels.length - 1}
                    <div class="step-sep {i < step ? 'done' : ''}"></div>
                {/if}
            {/each}
        </div>
    </header>

    <div class="wizard card">
        <!-- Step 0: Mode -->
        {#if step === 0}
            <div class="step-pane">
                <p class="section-label">Select Mode</p>
                <div class="mode-grid">
                    <button
                        class="mode-card {mode === 'default' ? 'active' : ''}"
                        on:click={() => (mode = "default")}
                    >
                        <div class="mode-icon">⚡</div>
                        <div class="mode-body">
                            <span class="mode-name">Default Mode</span>
                            <span class="mode-desc"
                                >Recommended set – safe for most users.</span
                            >
                        </div>
                    </button>
                    <button
                        class="mode-card {mode === 'custom' ? 'active' : ''}"
                        on:click={() => (mode = "custom")}
                    >
                        <div class="mode-icon">⚙️</div>
                        <div class="mode-body">
                            <span class="mode-name">Custom Setup</span>
                            <span class="mode-desc"
                                >Manually pick apps and tweaks.</span
                            >
                        </div>
                    </button>
                </div>
                <div class="step-footer">
                    <span></span>
                    <button class="btn btn-primary" on:click={nextStep}
                        >Next →</button
                    >
                </div>
            </div>

            <!-- Step 1: Apps -->
        {:else if step === 1}
            <div class="step-pane">
                <div class="step-header">
                    <p class="section-label">
                        App Removal — {selectedAppsCount} selected
                    </p>
                    <div class="step-actions">
                        <button class="btn-ghost" on:click={selectDefaultApps}
                            >Default</button
                        >
                        <button class="btn-ghost" on:click={clearApps}
                            >Clear</button
                        >
                    </div>
                </div>
                <div class="app-table">
                    <div class="app-head">
                        <span></span><span>Name</span><span>Description</span
                        ><span>Package</span>
                    </div>
                    {#each apps as app}
                        <label class="app-row {app.selected ? 'sel' : ''}">
                            <input
                                type="checkbox"
                                bind:checked={app.selected}
                            />
                            <span class="app-name">{app.name}</span>
                            <span class="app-desc">{app.desc}</span>
                            <span class="app-pkg">{app.package}</span>
                        </label>
                    {/each}
                </div>
                <div class="step-footer">
                    <button class="btn-ghost" on:click={prevStep}>← Back</button
                    >
                    <button class="btn btn-primary" on:click={nextStep}
                        >Next →</button
                    >
                </div>
            </div>

            <!-- Step 2: Tweaks -->
        {:else if step === 2}
            <div class="step-pane">
                <p class="section-label">System Tweaks</p>
                <div class="tweaks-cats">
                    {#each Object.entries(tweaks) as [cat, items]}
                        <div class="cat-block card">
                            <p class="cat-label">{cat}</p>
                            {#each items as tweak}
                                <label class="tweak-row">
                                    <input
                                        type="checkbox"
                                        bind:checked={tweak.selected}
                                    />
                                    <span>{tweak.name}</span>
                                </label>
                            {/each}
                        </div>
                    {/each}
                </div>
                <div class="step-footer">
                    <button class="btn-ghost" on:click={prevStep}>← Back</button
                    >
                    <button class="btn btn-primary" on:click={nextStep}
                        >Next →</button
                    >
                </div>
            </div>

            <!-- Step 3: Review -->
        {:else if step === 3}
            <div class="step-pane">
                <p class="section-label">Review Changes</p>
                <div class="review-grid">
                    <div class="card review-panel">
                        <p class="section-label" style="margin-bottom: 10px;">
                            Queued Changes
                        </p>
                        <div class="review-list">
                            {#each apps.filter((a) => a.selected) as app}
                                <div class="review-item">
                                    <span class="tag-neutral item-badge"
                                        >Remove</span
                                    >
                                    {app.name}
                                </div>
                            {/each}
                            {#each Object.values(tweaks)
                                .flat()
                                .filter((t) => t.selected) as tweak}
                                <div class="review-item">
                                    <span class="tag-accent item-badge"
                                        >Apply</span
                                    >
                                    {tweak.name}
                                </div>
                            {/each}
                        </div>
                    </div>
                    <div class="card review-panel">
                        <p class="section-label" style="margin-bottom: 10px;">
                            Options
                        </p>
                        <label class="opt-row">
                            <input
                                type="checkbox"
                                bind:checked={options.restorePoint}
                            />
                            <span
                                >Create system restore point (recommended)</span
                            >
                        </label>
                        <label class="opt-row">
                            <input
                                type="checkbox"
                                bind:checked={options.restartExplorer}
                            />
                            <span>Restart Explorer after applying changes</span>
                        </label>
                    </div>
                </div>
                <div class="step-footer">
                    <button class="btn-ghost" on:click={prevStep}>← Back</button
                    >
                    <button class="btn btn-primary" on:click={applyChanges} disabled={applying}>
                        {applying ? "Applying…" : "Apply Changes"}
                    </button>
                </div>
                {#if statusMessage}
                    <div class="status-bar" class:ok={statusOk} class:err={!statusOk}>
                        {statusMessage}
                    </div>
                {/if}
            </div>
        {/if}
    </div>
</div>

<style>
    /* Steps indicator */
    .steps {
        display: flex;
        align-items: center;
        gap: 6px;
    }
    .step-item {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 3px;
        opacity: 0.35;
    }
    .step-item.active,
    .step-item.done {
        opacity: 1;
    }
    .step-num {
        width: 22px;
        height: 22px;
        border-radius: 50%;
        border: 1px solid var(--border-md);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 0.7rem;
        font-weight: 700;
        color: var(--text-dim);
    }
    .step-item.active .step-num {
        border-color: var(--accent);
        color: var(--accent);
        background: var(--accent-dim);
    }
    .step-item.done .step-num {
        border-color: var(--success);
        color: var(--success);
        background: var(--success-dim);
    }
    .step-lbl {
        font-size: 0.6rem;
        color: var(--text-lo);
    }
    .step-sep {
        width: 20px;
        height: 1px;
        background: var(--border);
        margin-bottom: 14px;
    }
    .step-sep.done {
        background: var(--success);
    }

    /* Wizard */
    .wizard {
        flex: 1;
        overflow: hidden;
    }
    .step-pane {
        height: 100%;
        display: flex;
        flex-direction: column;
        gap: 16px;
    }
    .step-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    .step-footer {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding-top: 14px;
        border-top: 1px solid var(--border);
        margin-top: auto;
    }
    .step-actions {
        display: flex;
        gap: 8px;
    }

    /* Mode Cards */
    .mode-grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 12px;
    }
    .mode-card {
        background: var(--bg);
        border: 1px solid var(--border);
        border-radius: var(--radius);
        padding: 20px;
        gap: 14px;
        display: flex;
        align-items: flex-start;
        cursor: pointer;
        font-family: var(--font);
        text-align: left;
        color: inherit;
        transition:
            border-color 0.15s,
            background 0.15s;
    }
    .mode-card:hover {
        border-color: var(--border-hi);
        background: var(--bg-hover);
    }
    .mode-card.active {
        border-color: var(--accent);
        background: var(--accent-dim);
    }
    .mode-icon {
        font-size: 1.3rem;
    }
    .mode-body {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }
    .mode-name {
        font-size: 0.9rem;
        font-weight: 700;
        color: var(--text);
    }
    .mode-desc {
        font-size: 0.76rem;
        color: var(--text-dim);
    }

    /* App Table */
    .app-table {
        flex: 1;
        overflow-y: auto;
        border: 1px solid var(--border);
        border-radius: var(--radius);
    }
    .app-head {
        display: grid;
        grid-template-columns: 24px 1fr 1.2fr 1.8fr;
        padding: 8px 14px;
        font-size: 0.68rem;
        font-weight: 600;
        color: var(--text-dim);
        text-transform: uppercase;
        letter-spacing: 0.5px;
        background: var(--bg);
        border-bottom: 1px solid var(--border);
        position: sticky;
        top: 0;
    }
    .app-row {
        display: grid;
        grid-template-columns: 24px 1fr 1.2fr 1.8fr;
        align-items: center;
        padding: 9px 14px;
        border-bottom: 1px solid var(--border);
        cursor: pointer;
        font-family: var(--font);
        gap: 8px;
        transition: background 0.12s;
    }
    .app-row:last-child {
        border-bottom: none;
    }
    .app-row:hover {
        background: var(--bg-hover);
    }
    .app-row.sel {
        background: var(--accent-dim);
    }
    .app-row input {
        accent-color: var(--accent);
        cursor: pointer;
        margin: 0;
    }
    .app-name {
        font-size: 0.82rem;
        font-weight: 600;
        color: var(--text);
    }
    .app-desc {
        font-size: 0.74rem;
        color: var(--text-dim);
    }
    .app-pkg {
        font-size: 0.65rem;
        color: var(--text-lo);
        font-family: "Consolas", monospace;
    }

    /* Tweaks */
    .tweaks-cats {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
        gap: 10px;
        overflow-y: auto;
        flex: 1;
    }
    .cat-block {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }
    .cat-label {
        font-size: 0.72rem;
        font-weight: 700;
        color: var(--text-dim);
        text-transform: uppercase;
        letter-spacing: 0.5px;
        margin: 0;
    }
    .tweak-row {
        display: flex;
        align-items: flex-start;
        gap: 8px;
        font-size: 0.82rem;
        color: var(--text-dim);
        cursor: pointer;
    }
    .tweak-row input {
        accent-color: var(--accent);
        width: 14px;
        height: 14px;
        margin-top: 2px;
        flex-shrink: 0;
    }

    /* Review */
    .review-grid {
        display: grid;
        grid-template-columns: 1.2fr 1fr;
        gap: 12px;
        flex: 1;
        overflow: hidden;
    }
    .review-panel {
        overflow-y: auto;
    }
    .review-list {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }
    .review-item {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 0.82rem;
        color: var(--text-dim);
    }
    .item-badge {
        font-size: 0.62rem;
        flex-shrink: 0;
    }
    .opt-row {
        display: flex;
        gap: 8px;
        align-items: flex-start;
        font-size: 0.82rem;
        color: var(--text-dim);
        cursor: pointer;
        margin-bottom: 10px;
    }
    .opt-row input {
        accent-color: var(--accent);
        margin-top: 2px;
        flex-shrink: 0;
    }

    /* Status bar (replaces alert) */
    .status-bar {
        margin-top: 8px;
        padding: 10px 16px;
        border-radius: var(--radius);
        font-size: 0.82rem;
        font-weight: 600;
        border: 1px solid var(--border-md);
        color: var(--text-dim);
        background: var(--bg-hover);
    }
    .status-bar.ok  { background: var(--success-dim); border-color: rgba(16,185,129,0.3); color: var(--success); }
    .status-bar.err { background: var(--danger-dim);  border-color: rgba(239,68,68,0.3);  color: var(--danger); }
</style>
