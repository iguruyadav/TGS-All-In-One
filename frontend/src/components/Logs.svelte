<script>
    import { onMount } from "svelte";
    import { GetDetailedSystemLogs } from "../../wailsjs/go/main/App";

    let logs = [];
    let loading = true;
    let error = "";
    let filterLevel = "All"; // "All", "Critical", "Error"

    async function loadLogs() {
        loading = true;
        error = "";
        try {
            logs = await GetDetailedSystemLogs();
        } catch (e) {
            error = e.message || String(e);
        }
        loading = false;
    }

    onMount(() => {
        loadLogs();
    });

    $: filteredLogs = logs.filter(l => filterLevel === "All" || l.Level === filterLevel);
</script>

<div class="logs-wrap slide-in">
    <div style="display:flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
        <div class="section-title" style="margin:0;">🚨 SYSTEM EVENT LOG ANALYZER</div>
        <div style="display:flex; gap:10px;">
            <select class="ping-input" style="padding:6px 12px; border-radius: 6px; background: rgba(0,0,0,0.3); border: 1px solid rgba(255,255,255,0.1); color: #fff; outline:none;" bind:value={filterLevel}>
                <option value="All">All Events</option>
                <option value="Critical">Critical Only</option>
                <option value="Error">Errors Only</option>
            </select>
            <button class="btn-execute" on:click={loadLogs} disabled={loading}>🔄 Refresh Logs</button>
        </div>
    </div>

    {#if loading}
        <div class="proc-loading">
            <span class="spin" style="display:inline-block; font-size:1.5rem; margin-bottom:10px;">⏳</span><br/>
            Scanning Windows Event Viewer (last 14 days)...
        </div>
    {:else if error}
        <div class="alert-item crit">❌ Failed to load logs: {error}</div>
    {:else if logs.length === 0}
        <div class="proc-loading" style="color:#10b981; border: 1px solid rgba(16,185,129,0.3); background: rgba(16,185,129,0.1); border-radius: 8px;">
            ✅ No critical or error events found in the last 14 days! Your system is perfectly stable.
        </div>
    {:else}
        <div class="proc-table">
            <div class="proc-header">
                <span style="min-width:140px; max-width:140px;">Time</span>
                <span style="min-width:90px; max-width:90px;">Level</span>
                <span style="min-width:140px; max-width:140px;">Source</span>
                <span style="min-width:70px; max-width:70px;">ID</span>
                <span style="flex:1;">Message / AI Suggestion</span>
            </div>
            {#each filteredLogs as log}
            <div class="proc-row">
                <span class="proc-pid" style="min-width:140px; max-width:140px;">{log.Time}</span>
                <span style="min-width:90px; max-width:90px;" class="st-badge" class:st-crit={log.Level==='Critical'} class:st-err={log.Level==='Error'}>● {log.Level}</span>
                <span class="proc-name" style="min-width:140px; max-width:140px; opacity:0.8;">{log.Source}</span>
                <span class="proc-pid" style="min-width:70px; max-width:70px;">{log.EventID}</span>
                <span class="log-msg" style="flex:1;">{log.Message}</span>
            </div>
            {/each}
        </div>
    {/if}
</div>

<style>
    .logs-wrap {
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 12px;
        padding: 24px 32px;
        display: flex;
        flex-direction: column;
        height: 100%;
        margin: 20px;
        box-shadow: var(--glass-shadow);
        backdrop-filter: var(--glass-blur);
    }
    .slide-in { animation: slideIn 0.3s cubic-bezier(0.4, 0, 0.2, 1); }
    @keyframes slideIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
    
    .section-title {
        font-size: 0.8rem; font-weight: 800; letter-spacing: 2px; color: var(--accent2);
    }
    
    .btn-execute { background: rgba(34,211,238,0.1); border: 1px solid rgba(34,211,238,0.4); color: #22D3EE; font-family: var(--font); font-size: 0.75rem; font-weight: 700; padding: 7px 16px; border-radius: 6px; cursor: pointer; transition: all 0.2s; }
    .btn-execute:hover:not(:disabled) { background: rgba(34,211,238,0.2); color: #fff; box-shadow: 0 0 10px rgba(34,211,238,0.3); }
    .btn-execute:disabled { opacity: 0.5; cursor: default; }
    
    .proc-table { display: flex; flex-direction: column; gap: 4px; overflow-y: auto; flex: 1; margin-top: 10px; padding-right: 6px; }
    .proc-header { display: flex; padding: 10px 12px; font-size: 0.68rem; font-weight: 800; letter-spacing: 0.8px; color: var(--text-dim); text-transform: uppercase; border-bottom: 1px solid var(--border-md); margin-bottom: 4px; }
    .proc-row { display: flex; align-items: flex-start; padding: 12px; border-radius: 8px; transition: background 0.15s; border: 1px solid transparent; background: rgba(0,0,0,0.2); }
    .proc-row:hover { background: rgba(255,255,255,0.04); border-color: rgba(255,255,255,0.08); }
    
    .proc-name { font-size: 0.75rem; color: var(--text); font-weight: 500; word-break: break-all; }
    .proc-pid { font-size: 0.75rem; color: var(--text-dim); font-family: monospace; }
    .log-msg { font-size: 0.78rem; color: #cbd5e1; line-height: 1.4; word-break: break-word; }
    .proc-loading { padding: 40px; text-align: center; font-size: 0.9rem; font-weight: 600; color: var(--text-dim); border: 1px dashed var(--border); border-radius: 12px; margin-top: 20px; }
    
    .alert-item { padding: 12px 16px; border-radius: 8px; font-size: 0.85rem; font-weight: 700; margin-top: 20px; }
    .alert-item.crit { background: rgba(239,68,68,0.12); border: 1px solid rgba(239,68,68,0.35); color: #ef4444; }
    
    .st-badge { font-size: 9px; font-weight: 800; letter-spacing: 1.2px; padding: 3px 8px; border-radius: 20px; white-space: nowrap; display: inline-block; text-align: center; margin-top: -2px; }
    .st-crit { color: #f43f5e; background: rgba(244,63,94,0.12); border: 1px solid rgba(244,63,94,0.3); }
    .st-err { color: #f59e0b; background: rgba(245,158,11,0.12); border: 1px solid rgba(245,158,11,0.3); }
    
    @keyframes spin { 100% { transform: rotate(360deg); } }
    .spin { animation: spin 1.5s linear infinite; }
</style>