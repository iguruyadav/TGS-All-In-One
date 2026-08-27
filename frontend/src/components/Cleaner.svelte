<script>
    import { RunCleanup, RunAdvancedClean, ScanNativeCleanup, ScanAdvancedCleanup } from "../../wailsjs/go/main/App";

    const targets = [
        {
            key: "cleanTemp",
            label: "Temporary Caches",
            meta: "Prefetch, Shader Cache, WER",
            code: "Temp",
            adv: false,
        },
        {
            key: "cleanBin",
            label: "Recycle Bin",
            meta: "System Bin & DNS Cache",
            code: "Bin",
            adv: false,
        },
        {
            key: "cleanUpdates",
            label: "Windows Update",
            meta: "Download Cache & Stale Files",
            code: "Updates",
            adv: false,
        },
        {
            key: "cleanSxS",
            label: "WinSxS Store",
            meta: "Component Store Optimization",
            code: "SxS",
            adv: false,
        },
        {
            key: "cleanVSS",
            label: "Shadow Copies",
            meta: "VSS Storage Allocation",
            code: "VSS",
            adv: false,
        },
        {
            key: "removeBloat",
            label: "System Bloat",
            meta: "Remove native bloatware",
            code: "Bloat",
            adv: false,
        },
        {
            key: "cleanJunk",
            label: "Junk Files",
            meta: "Crash dumps, thumbnails, WER logs",
            code: "Junk",
            adv: true,
        },
        {
            key: "cleanBrowser",
            label: "Browser Cache",
            meta: "Chrome, Edge, Firefox cache",
            code: "Browser",
            adv: true,
        },
        {
            key: "cleanWinLogs",
            label: "Windows Logs",
            meta: "CBS, DISM, IIS, WU logs",
            code: "WinLogs",
            adv: true,
        },
    ];

    let selected = {
        cleanTemp: true,
        cleanBin: true,
        cleanUpdates: false,
        cleanSxS: false,
        cleanVSS: false,
        removeBloat: false,
        cleanJunk: false,
        cleanBrowser: false,
        cleanWinLogs: false,
    };

    let status = "";
    let loading = false;
    let scanning = false;
    let scanResults = null;
    let cleanProgress = 0;   // 0..total tasks selected
    let cleanTotal   = 0;
    let cleanCurrent = "";   // label of current task

    async function scanTargets() {
        scanning = true;
        scanResults = null;
        status = "Scanning targets…";
        const tasksToRun = targets.filter(t => selected[t.key]);
        try {
            const results = [];
            for (const t of tasksToRun) {
                const fn = t.adv ? ScanAdvancedCleanup : ScanNativeCleanup;
                const resStr = await fn(t.code);
                let beforeSize = "0 B";
                let hasError = false;
                let errorMsg = "";
                try {
                    let data = JSON.parse(resStr);
                    if (data.error) {
                        hasError = true;
                        errorMsg = data.error;
                    } else {
                        beforeSize = `${data.size}`;
                    }
                } catch(e) {
                    hasError = true;
                    errorMsg = "Parse error";
                }
                results.push({ label: t.label, code: t.code, adv: t.adv, beforeSize, afterSize: null, status: "pending", hasError, errorMsg });
            }
            scanResults = results;
            status = "";
        } catch (e) {
            status = "Error scanning: " + e;
        } finally {
            scanning = false;
        }
    }

    async function runCleanup() {
        loading = true;
        cleanProgress = 0;
        cleanTotal = scanResults ? scanResults.length : 0;
        cleanCurrent = "";
        status = "Running cleanup…";
        try {
            for (let i = 0; i < scanResults.length; i++) {
                let res = scanResults[i];
                cleanCurrent = res.label;
                res.status = "cleaning";
                scanResults = [...scanResults]; // Reactivity
                
                // Clean
                const cleanFn = res.adv ? RunAdvancedClean : RunCleanup;
                await cleanFn(res.code);
                
                // Re-Scan for After Size
                const scanFn = res.adv ? ScanAdvancedCleanup : ScanNativeCleanup;
                const resStr = await scanFn(res.code);
                let afterSize = "0 B";
                try {
                    let data = JSON.parse(resStr);
                    if (data.error) {
                        afterSize = data.error;
                    } else {
                        afterSize = `${data.size}`;
                    }
                } catch(e) {
                    afterSize = "Error";
                }
                
                res.afterSize = afterSize;
                res.status = "done";
                cleanProgress++;
                scanResults = [...scanResults];
            }
            status = "Cleanup complete.";
        } catch (e) {
            status = "Error: " + e;
        } finally {
            loading = false;
        }
    }

    $: selectCount = targets.filter((t) => selected[t.key]).length;

    // Clear scan results if selection changes
    let lastSelected = JSON.stringify(selected);
    $: {
        let currentSelected = JSON.stringify(selected);
        if (currentSelected !== lastSelected) {
            scanResults = null;
            if (status.startsWith("Scanning") || status.startsWith("Error scanning")) status = "";
            lastSelected = currentSelected;
        }
    }
</script>

<div class="page">
    <header class="page-header">
        <div>
            <h1 class="page-title">Cleaner</h1>
            <p class="page-desc">
                Remove temporary files and free up system storage
            </p>
        </div>
        <div style="display:flex; gap:12px;">
            <button
                class="btn-enable"
                style="padding: 8px 16px; background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.1); color: var(--text);"
                on:click={scanTargets}
                disabled={loading || scanning || selectCount === 0}
            >
                {scanning ? "Scanning…" : "🔍 Scan Targets"}
            </button>
            <button
                class="btn btn-danger"
                on:click={runCleanup}
                disabled={loading || scanning || selectCount === 0 || !scanResults}
            >
                {loading ? "Running…" : "🧹 Clean Now"}
            </button>
        </div>
    </header>

    <div class="cleaner-layout">
        <div class="targets-col">
            <p class="section-label">Select Targets ({selectCount})</p>
            <div class="target-list card">
                {#each targets as t}
                    <label class="target-row" class:active={selected[t.key]}>
                        <input type="checkbox" bind:checked={selected[t.key]} />
                        <div class="t-info">
                            <span class="t-name">{t.label}</span>
                            <span class="t-meta">{t.meta}</span>
                        </div>
                        {#if t.adv}<span class="tag-neutral adv-tag"
                                >Advanced</span
                            >{/if}
                    </label>
                {/each}
            </div>
        </div>

        <div class="log-col">
            <p class="section-label">Output Log</p>
            <div class="log-card card">
                {#if loading}
                    <div class="progress-wrap">
                        <div class="prog-header">
                            <span class="prog-task">Cleaning {cleanProgress} of {cleanTotal}</span>
                            <span class="prog-pct">{cleanTotal > 0 ? Math.round((cleanProgress/cleanTotal)*100) : 0}%</span>
                        </div>
                        {#if cleanCurrent}
                        <div class="prog-current">➤ {cleanCurrent}…</div>
                        {/if}
                        <div class="prog-track">
                            <div class="prog-fill" style="width:{cleanTotal>0?(cleanProgress/cleanTotal)*100:0}%"></div>
                        </div>
                    </div>
                {:else if status}
                    <pre class="log-pre">{status}</pre>
                {:else if scanResults}
                    <div class="preview-wrap">
                        <div class="preview-header">
                            <span class="preview-title">📊 Scan & Clean Results</span>
                        </div>
                        <div class="results-cards">
                            {#each scanResults as res}
                                <div class="rcard" class:rc-cleaning={res.status === 'cleaning'} class:rc-done={res.status === 'done'}>
                                    <div class="rc-header">
                                        <span class="rc-title">{res.label}</span>
                                        {#if res.status === 'pending'}
                                            {#if res.hasError}
                                                <span class="rc-badge bg-gray">Not Scannable</span>
                                            {:else}
                                                <span class="rc-badge bg-blue">{res.beforeSize} Found</span>
                                            {/if}
                                        {:else if res.status === 'cleaning'}
                                            <span class="rc-badge bg-yellow"><span class="rc-spinner"></span> Cleaning...</span>
                                        {:else if res.status === 'done'}
                                            <span class="rc-badge bg-green">{res.hasError ? 'Skipped' : (res.afterSize === '0 B' ? res.beforeSize + ' Freed ✅' : 'Cleaned ✅')}</span>
                                        {/if}
                                    </div>
                                </div>
                            {/each}
                        </div>
                        {#if status !== "Cleanup complete."}
                        <p class="preview-hint" style="margin-top: 10px;">Click <strong>Clean Now</strong> to delete these files.</p>
                        {/if}
                    </div>
                {:else if selectCount > 0}
                    <!-- Show selected targets preview -->
                    <div class="preview-wrap">
                        <div class="preview-header">
                            <span class="preview-title">📋 Ready to clean {selectCount} target{selectCount!==1?'s':''}</span>
                        </div>
                        <div class="preview-list">
                            {#each targets.filter(t => selected[t.key]) as t}
                            <div class="preview-item">
                                <div class="pi-name">{t.label} {#if t.adv}<span class="tag-neutral adv-tag">Advanced</span>{/if}</div>
                                <div class="pi-meta">{t.meta}</div>
                            </div>
                            {/each}
                        </div>
                        <p class="preview-hint">Click <strong>Scan Targets</strong> to calculate sizes.</p>
                    </div>
                {:else}
                    <span class="log-empty">Select targets on the left and click Scan Targets.</span>
                {/if}
            </div>
        </div>
    </div>
</div>

<style>
    .cleaner-layout {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 16px;
        flex: 1;
        overflow: hidden;
    }
    .targets-col {
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }
    .log-col {
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }

    .target-list {
        padding: 8px;
        display: flex;
        flex-direction: column;
        gap: 4px;
        overflow-y: auto;
        flex: 1;
    }
    .target-row {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 10px 12px;
        border-radius: var(--radius);
        cursor: pointer;
        transition: background 0.12s;
    }
    .target-row:hover {
        background: var(--bg-hover);
    }
    .target-row.active {
        background: var(--accent-dim);
    }
    .target-row input {
        accent-color: var(--accent);
        width: 15px;
        height: 15px;
        flex-shrink: 0;
        cursor: pointer;
    }
    .t-info {
        display: flex;
        flex-direction: column;
        gap: 2px;
        flex: 1;
    }
    .t-name {
        font-size: 0.84rem;
        font-weight: 600;
        color: var(--text);
    }
    .t-meta {
        font-size: 0.7rem;
        color: var(--text-dim);
    }
    .adv-tag {
        font-size: 0.62rem;
        flex-shrink: 0;
    }

    .log-card {
        flex: 1;
        display: flex;
        flex-direction: column;
        overflow: hidden;
        padding: 16px;
    }
    .log-pre {
        margin: 0;
        white-space: pre-wrap;
        font-family: "Consolas", monospace;
        font-size: 0.8rem;
        color: var(--success);
        line-height: 1.6;
        overflow-y: auto;
    }
    .log-empty {
        margin: auto;
        color: var(--text-dim);
        font-size: 0.9rem;
    }

    /* ── PROGRESS BAR ────────────────────── */
    .progress-wrap { display: flex; flex-direction: column; gap: 8px; width: 100%; padding: 12px; }
    .prog-header { display: flex; justify-content: space-between; font-size: 0.85rem; font-weight: 600; }
    .prog-pct { color: var(--accent); }
    .prog-current { font-size: 0.75rem; color: var(--text-dim); font-family: monospace; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
    .prog-track {
        height: 8px; background: rgba(0,0,0,0.3); border-radius: 4px;
        overflow: hidden; border: 1px solid rgba(255,255,255,0.05);
    }
    .prog-fill {
        height: 100%; border-radius: 4px;
        background: linear-gradient(90deg, #7C3AED, #22D3EE);
        transition: width 0.4s ease;
        box-shadow: 0 0 8px rgba(124,58,237,0.5);
    }

    /* ── PREVIEW PANEL ────────────────────── */
    .preview-wrap { display: flex; flex-direction: column; gap: 10px; width: 100%; }
    .preview-header { border-bottom: 1px solid var(--border, rgba(255,255,255,.08)); padding-bottom: 8px; }
    .preview-title { font-size: 0.84rem; font-weight: 700; color: var(--accent, #7c3aed); }
    .preview-list { display: flex; flex-direction: column; gap: 6px; overflow-y: auto; max-height: 240px; }
    .preview-item {
        padding: 8px 10px; border-radius: 8px;
        background: rgba(124,58,237,.07); border: 1px solid rgba(124,58,237,.18);
    }
    .pi-name { font-size: 0.8rem; font-weight: 600; color: var(--text, #fff); display: flex; align-items: center; gap: 6px; }
    .pi-meta { font-size: 0.7rem; color: var(--text-dim, #888); margin-top: 2px; }
    .preview-hint { font-size: 0.75rem; color: var(--text-dim, #888); margin: 4px 0 0; }
    .preview-hint strong { color: var(--accent-text, #a78bfa); }

    /* ── RESULT CARDS ────────────────────── */
    .results-cards { display: flex; flex-direction: column; gap: 8px; max-height: 400px; overflow-y: auto; padding-right: 4px; margin-top: 4px;}
    .rcard { background: rgba(0,0,0,0.2); border: 1px solid rgba(255,255,255,0.05); border-radius: 8px; padding: 12px 14px; transition: all 0.2s; }
    .rcard:hover { background: rgba(255,255,255,0.03); }
    .rc-cleaning { border-color: rgba(34, 211, 238, 0.4); background: rgba(34, 211, 238, 0.05); }
    .rc-done { border-color: rgba(16, 185, 129, 0.4); background: rgba(16, 185, 129, 0.05); }
    
    .rc-header { display: flex; justify-content: space-between; align-items: center; }
    .rc-title { font-size: 0.85rem; font-weight: 600; color: var(--text); }
    
    .rc-badge { font-size: 0.75rem; font-weight: 600; padding: 4px 10px; border-radius: 20px; display: flex; align-items: center; gap: 6px; }
    .bg-gray { background: rgba(255,255,255,0.1); color: var(--text-dim); }
    .bg-blue { background: rgba(59, 130, 246, 0.15); color: #60a5fa; border: 1px solid rgba(59, 130, 246, 0.3); }
    .bg-yellow { background: rgba(245, 158, 11, 0.15); color: #fbbf24; border: 1px solid rgba(245, 158, 11, 0.3); }
    .bg-green { background: rgba(16, 185, 129, 0.15); color: #34d399; border: 1px solid rgba(16, 185, 129, 0.3); }
    
    .rc-spinner { width: 12px; height: 12px; border: 2px solid rgba(251, 191, 36, 0.3); border-top-color: #fbbf24; border-radius: 50%; animation: spin 1s linear infinite; }
</style>
