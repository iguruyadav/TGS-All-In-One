<script>
    import { onMount } from "svelte";
    import { pushLog } from "../stores/log.js";
    import {
        GetStats, GetAudit, GetDashboardStatus,
        SetPowerPlan, GetSystemHealth,
        RunNativeAction, FlushDNS, RenewIP, WinsockReset, TCPIPReset, AllowICMPPing,
        GetAllProcesses, KillProcess, GetBootAnalysis, GetDriverHealth,
        Ping, ApplySecurity, HibernateHeavyApps, TrimProcessWorkingSets, ClearStandbyMemory,
        ExportSystemReport, PauseWindowsUpdate, GetCurrentPowerPlan,
        ToggleAnimations, ToggleBackgroundApps,
        ConfigurePageFile, ApplyPerfTweak, ScheduleAutoCleanup,
        GetProductKeys
    } from "../../wailsjs/go/main/App";

    // ── NETWORK SPEED DELTA tracking ─────────────────────────────────────────────
    let prevNetIn  = 0;
    let prevNetOut = 0;
    let netInKBs   = 0;
    let netOutKBs  = 0;

    let stats = null;
    let data  = {};
    let st    = {};   // DashboardStatus — live toggle states
    let statsInt;
    let health = { StabilityScore: 10, LastFailures: [], CriticalLogs: [] };
    let winKey = "Loading...";
    
    // Threat alert state
    let alerts = [];
    
    // Tabs: "specs" | "power" | "memory" | "network" | "processes" | "drivers" | "boot" | "tools"
    let activeTab = "specs";

    let pingTarget = "8.8.8.8";
    let pingResult = "";
    let pinging = false;

    let executing = false;
    let feedback = "";
    let activePlan = "balanced";
    
    // Process Manager
    let processes = [];
    let procFilter = "";
    let procsLoading = false;
    
    // Driver Health
    let drivers = [];
    let driversLoading = false;
    
    // Boot Analysis
    let bootItems = [];
    let bootLoading = false;

    // Health loading state
    let healthLoading = false;
    
    // Auto Cleanup
    let scheduleHours = 24;
    let updatePauseDays = 7;

    async function loadData() {
        try {
            [data, st] = await Promise.all([GetAudit(), GetDashboardStatus()]);
            activePlan = st?.PowerPlan || "balanced";
            GetProductKeys().then(k => winKey = k).catch(() => winKey = "Unavailable");
        } catch (_) {}
    }

    async function refreshStatus() {
        try {
            st = await GetDashboardStatus();
            activePlan = st?.PowerPlan || "balanced";
        } catch (_) {}
    }

    async function loadHealth() {
        healthLoading = true;
        try {
            health = await GetSystemHealth();
            saveSparkPoint(health.StabilityScore ?? 10);
        } catch (_) {}
        healthLoading = false;
    }

    // Live threat alerts — watch stats reactively
    $: {
        const newAlerts = [];
        if (cpuPct > 85) newAlerts.push({ type: 'warn', msg: `⚠️ CPU at ${cpuPct}% — High load!` });
        if (ramPct > 85) newAlerts.push({ type: 'warn', msg: `⚠️ RAM at ${ramPct}% — Low memory!` });
        if (diskPct > 90) newAlerts.push({ type: 'crit', msg: `🚨 Disk C: at ${diskPct}% — Almost full!` });
        alerts = newAlerts;
    }

    async function loadProcesses() {
        procsLoading = true;
        try { processes = await GetAllProcesses(); } catch(_) { processes = []; }
        procsLoading = false;
    }

    async function loadDrivers() {
        driversLoading = true;
        try { drivers = await GetDriverHealth(); } catch(_) { drivers = []; }
        driversLoading = false;
    }

    async function loadBoot() {
        bootLoading = true;
        try { bootItems = await GetBootAnalysis(); } catch(_) { bootItems = []; }
        bootLoading = false;
    }

    let healthInt;
    onMount(() => {
        loadData();
        loadSparkHistory();
        updateStats();
        loadHealth();
        statsInt   = setInterval(updateStats, 2000);
        healthInt  = setInterval(loadHealth, 30000); // refresh health every 30s
        return () => {
            clearInterval(statsInt);
            clearInterval(healthInt);
        };
    });

    // ── NETWORK SPEED DELTA ───────────────────────────────────────────────────
    async function updateStats() {
        try {
            const s = await GetStats();
            // Calculate delta KB/s from cumulative counters
            const inMB  = Number(s?.Network?.In  ?? 0);
            const outMB = Number(s?.Network?.Out ?? 0);
            if (prevNetIn > 0) {
                netInKBs  = Math.max(0, Math.round((inMB  - prevNetIn)  * 1024));
                netOutKBs = Math.max(0, Math.round((outMB - prevNetOut) * 1024));
            }
            prevNetIn  = inMB;
            prevNetOut = outMB;
            stats = s;
        } catch (_) {}
    }

    // ── HEALTH SPARKLINE — store 7-day history in localStorage ───────────────
    const SPARKLINE_KEY = 'tgs_health_history';
    let sparkPoints = [];

    function loadSparkHistory() {
        try {
            const raw = localStorage.getItem(SPARKLINE_KEY);
            sparkPoints = raw ? JSON.parse(raw) : [];
        } catch (_) { sparkPoints = []; }
    }

    function saveSparkPoint(score) {
        const now = new Date();
        sparkPoints = [...sparkPoints, { t: now.toLocaleDateString('en-IN', {day:'2-digit',month:'short'}), v: score }]
            .slice(-14); // keep last 14 readings
        try { localStorage.setItem(SPARKLINE_KEY, JSON.stringify(sparkPoints)); } catch (_) {}
    }

    function buildSparkPath(points, w = 120, h = 32) {
        if (points.length < 2) return '';
        const vals = points.map(p => p.v);
        const min = Math.min(...vals); const max = Math.max(...vals);
        const range = max - min || 1;
        const xs = points.map((_, i) => (i / (points.length - 1)) * w);
        const ys = vals.map(v => h - ((v - min) / range) * (h - 4) - 2);
        return xs.map((x, i) => `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${ys[i].toFixed(1)}`).join(' ');
    }

    // ── LIVE STATS ─────────────────────────────────────────────────────────────
    $: cpuPct    = Math.round(stats?.CPU            ?? 0);
    $: ramPct    = Math.round(stats?.RAM?.Percent   ?? 0);
    $: ramUsed   = stats?.RAM?.Used       || "—";
    $: ramFree   = stats?.RAM?.Available  || "—";
    $: vmPct     = stats?.VirtualMem?.Percent ?? 0;
    $: vmUsed    = stats?.VirtualMem?.Used    || "—";
    $: vmFree    = stats?.VirtualMem?.Available || "—";
    $: diskPct   = Math.round(stats?.Disk ?? 0);
    $: netIn     = Number(stats?.Network?.In  ?? 0).toFixed(1);
    $: netOut    = Number(stats?.Network?.Out ?? 0).toFixed(1);

    // ── AUDIT DATA ─────────────────────────────────────────────────────────────
    $: sys       = data?.System      || {};
    $: hw        = data?.Hardware    || {};
    $: net       = data?.Network     || {};
    $: sec       = data?.Security    || {};
    $: ramTotal  = hw?.RAMTotal      || "—";
    $: storages  = hw?.Storage       || [];
    $: cpuModel  = hw?.CPU           || "—";

    function pctColor(p) {
        if (p >= 85) return "#ef4444";
        if (p >= 60) return "#f59e0b";
        if (p >= 30) return "#a78bfa";
        return "#10b981";
    }

    function showFeedback(msg) {
        feedback = msg;
        setTimeout(() => feedback = "", 4000);
    }

    // ── ACTIONS ────────────────────────────────────────────────────────────────
    async function doPowerPlan(plan) {
        if (executing) return;
        executing = true;
        try {
            const res = await SetPowerPlan(plan);
            showFeedback("✅ " + res);
            pushLog("[Dashboard › CPU & Power] Power Plan → " + plan.toUpperCase() + ": " + res, true);
            await refreshStatus();
        } catch (e) {
            const err = "Power Plan failed: " + (e?.message || e);
            showFeedback("❌ " + err);
            pushLog("[Dashboard › CPU & Power] " + err, false);
        }
        executing = false;
    }

    async function runPing() {
        if (pinging || !pingTarget) return;
        pinging = true;
        pingResult = "Pinging " + pingTarget + "...";
        try {
            const res = await Ping(pingTarget);
            pingResult = res;
            pushLog("[Dashboard › Network] Ping " + pingTarget + ": " + (res?.split('\n')[2]?.trim() || "Done"), true);
        } catch(e) {
            pingResult = "Ping failed: " + e;
            pushLog("[Dashboard › Network] Ping " + pingTarget + " FAILED: " + e, false);
        }
        pinging = false;
    }

    // Single unified action runner — manages executing lock for ALL buttons
    async function doAction(fn, label) {
        if (executing) return;
        executing = true;
        try {
            const res = await fn();
            const msg = (typeof res === 'string' ? res : JSON.stringify(res)) || "Done.";
            showFeedback("✅ " + msg);
            pushLog("[Dashboard] " + (label || "") + ": " + msg, true);
            await refreshStatus();
        } catch (e) {
            const err = (e?.message || String(e));
            showFeedback("❌ Failed: " + err);
            pushLog("[Dashboard] " + (label || "") + " FAILED: " + err, false);
        }
        executing = false;
    }

    async function doSecurity(actionKey, actionVal, label) {
        if (executing) return;
        executing = true;
        try {
            const res = await ApplySecurity(actionKey, actionVal);
            showFeedback("✅ " + res);
            pushLog("[Dashboard › Security] " + label + ": " + res, true);
            await loadData();
        } catch (e) {
            const err = (e?.message || String(e));
            showFeedback("❌ Failed: " + err);
            pushLog("[Dashboard › Security] " + label + " FAILED: " + err, false);
        }
        executing = false;
    }

</script>

<div class="dash">
    <!-- ── TOP STAT CARDS ── -->
    <div class="stat-row">
        <!-- CPU -->
        <div class="stat-card">
            <div class="sc-top">
                <span class="sc-label">CPU</span>
                <span class="sc-value" style="color:{pctColor(cpuPct)}">{cpuPct}%</span>
            </div>
            <div class="sc-sub">{cpuModel.split('@')[0] || '—'}</div>
            <div class="sc-bot"><span class="tag-accent" style="font-size: 0.65rem;">{activePlan.toUpperCase()}</span></div>
            <div class="sc-bar"><div class="sc-fill" style="width:{cpuPct}%; background:{pctColor(cpuPct)}"></div></div>
        </div>

        <!-- RAM -->
        <div class="stat-card">
            <div class="sc-top">
                <span class="sc-label">RAM USAGE</span>
                <span class="sc-value" style="color:{pctColor(ramPct)}">{ramPct}%</span>
            </div>
            <div class="sc-sub">{ramUsed} used</div>
            <div class="sc-bot">{ramFree} free</div>
            <div class="sc-bar"><div class="sc-fill" style="width:{ramPct}%; background:{pctColor(ramPct)}"></div></div>
        </div>

        <!-- UPTIME -->
        <div class="stat-card">
            <div class="sc-top">
                <span class="sc-label">UPTIME</span>
                <span class="sc-value" style="color:#a78bfa">{stats?.Uptime || '—'}</span>
            </div>
            <div class="sc-sub">Last boot</div>
            <div class="sc-bot" style="font-family:monospace; font-size:0.7rem;">{stats?.LastBoot || '—'}</div>
        </div>

        <!-- STORAGE -->
        <div class="stat-card">
            <div class="sc-top">
                <span class="sc-label">STORAGE C:</span>
                <span class="sc-value" style="color:{pctColor(diskPct)}">{diskPct}%</span>
            </div>
            <div class="sc-sub">{storages[0]?.Label || 'C:'}</div>
            <div class="sc-bot">{100 - diskPct}% free</div>
            <div class="sc-bar"><div class="sc-fill" style="width:{diskPct}%; background:{pctColor(diskPct)}"></div></div>
        </div>

        <!-- NETWORK -->
        <div class="stat-card">
            <div class="sc-top">
                <span class="sc-label">NETWORK</span>
                <span class="net-status" class:on={net?.Internet} class:off={!net?.Internet}>
                    {net?.Internet ? 'Online' : 'Offline'}
                </span>
            </div>
            <div class="sc-sub net-speed">
                <span class:fast={netInKBs > 500}>↓ {netInKBs < 1024 ? netInKBs + ' KB/s' : (netInKBs/1024).toFixed(1)+' MB/s'}</span>
                <span class:fast={netOutKBs > 500}>↑ {netOutKBs < 1024 ? netOutKBs + ' KB/s' : (netOutKBs/1024).toFixed(1)+' MB/s'}</span>
            </div>
            <div class="sc-sub">{net?.IP || '—'}</div>
            <div class="sc-sub" style="font-size:0.68rem;color:var(--text-lo)">{net?.Adapter || '—'}</div>
        </div>
    </div>

    <!-- ── HEALTH SCORE + ALERTS ── -->
    {#if alerts.length > 0}
    <div class="alert-bar">
        {#each alerts as al}
        <div class="alert-item" class:crit={al.type==='crit'}>{al.msg}</div>
        {/each}
    </div>
    {/if}

    <div class="health-strip">
        <div class="hs-left">
            <span class="hs-score" style="color:{(health.StabilityScore??0)>=7.5?'#10b981':(health.StabilityScore??0)>=5?'#f59e0b':'#ef4444'}">
                {(health.StabilityScore??0).toFixed(1)}<span class="hs-max">/10</span>
            </span>
            <span class="hs-grade">Stability</span>
        </div>
        <div class="hs-mid">
            <div class="hs-bar"><div class="hs-fill" style="width:{((health.StabilityScore??0)*10)}%; background:{(health.StabilityScore??0)>=7.5?'#10b981':(health.StabilityScore??0)>=5?'#f59e0b':'#ef4444'}"></div></div>
            <span class="hs-summary">{health.Summary || 'System health data loading…'}</span>
            {#if stats?.LastBoot}
            <span class="hs-boottime">⏰ Last boot: {stats.LastBoot}</span>
            {/if}
        </div>
        {#if sparkPoints.length >= 2}
        {@const sparkMin = Math.min(...sparkPoints.map(p => p.v))}
        {@const sparkMax = Math.max(...sparkPoints.map(p => p.v))}
        {@const sparkRange = sparkMax - sparkMin || 1}
        <div class="spark-wrap" title="7-day health trend">
            <svg class="spark-svg" width="120" height="32" viewBox="0 0 120 32">
                <path d={buildSparkPath(sparkPoints)} fill="none" stroke="#7C3AED" stroke-width="1.5" stroke-linejoin="round" stroke-linecap="round" opacity="0.8"/>
                <circle cx="120" cy={32 - ((sparkPoints[sparkPoints.length-1].v - sparkMin) / sparkRange) * 28 - 2} r="3" fill="#22D3EE"/>
            </svg>
            <span class="spark-label">{sparkPoints.length} readings</span>
        </div>
        {/if}
        <button class="hs-export" on:click={()=>doAction(ExportSystemReport,'Export Report')} disabled={executing}>📋 Export Report</button>
    </div>

    <!-- ── TABS ── -->
    <div class="dash-tabs">
        <button class="d-tab" class:active={activeTab==="specs"} on:click={()=>activeTab="specs"}>📄 Specs</button>
        <button class="d-tab" class:active={activeTab==="power"} on:click={()=>activeTab="power"}>⚡ CPU & Power</button>
        <button class="d-tab" class:active={activeTab==="memory"} on:click={()=>activeTab="memory"}>🧠 Memory</button>
        <button class="d-tab" class:active={activeTab==="network"} on:click={()=>activeTab="network"}>🌐 Network</button>
        <button class="d-tab" class:active={activeTab==="processes"} on:click={()=>{activeTab="processes";loadProcesses();}}>⚙️ Processes</button>
        <button class="d-tab" class:active={activeTab==="drivers"} on:click={()=>{activeTab="drivers";loadDrivers();}}>🔌 Drivers</button>
        <button class="d-tab" class:active={activeTab==="boot"} on:click={()=>{activeTab="boot"; if(bootItems.length===0) loadBoot();}}>🚀 Boot</button>
        <button class="d-tab" class:active={activeTab==="tools"} on:click={()=>activeTab="tools"}>🛠 Tools</button>
    </div>

    <!-- ── TAB CONTENT ── -->
    <div class="tab-content">
        
        {#if activeTab === "specs"}
            <!-- SPECS TAB -->
            <div class="specs-grid">
                <!-- SYSTEM -->
                <div class="spec-card">
                    <div class="g-label">🖥️ SYSTEM</div>
                    <div class="divider"></div>
                    <div class="sp-row"><span class="sp-lbl">Computer Name</span> <span class="sp-val">{sys?.Name || '—'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">Logged User</span> <span class="sp-val">{sys?.User || '—'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">Domain</span> <span class="sp-val">{sys?.Domain || '—'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">System Model</span> <span class="sp-val">{sys?.Model || sys?.Manufacturer || '—'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">Operating System</span> <span class="sp-val">{sys?.OS || '—'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">Uptime</span> <span class="sp-val" style="color:#a78bfa; font-weight:700;">{stats?.Uptime || '—'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">Last Boot</span> <span class="sp-val">{stats?.LastBoot || '—'}</span></div>
                </div>

                <!-- PROCESSOR -->
                <div class="spec-card">
                    <div class="g-label">⚡ PROCESSOR</div>
                    <div class="divider"></div>
                    <div class="sp-row"><span class="sp-lbl">CPU</span> <span class="sp-val" style="color: #c4b5fd;">{hw?.CPU || '—'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">Usage (Live)</span> <span class="sp-val" style="color:{pctColor(cpuPct)}; font-weight: 800;">{cpuPct}%</span></div>
                    <div class="sp-row"><span class="sp-lbl">Power Plan</span> <span class="sp-val">{sys?.PowerPlan || 'Balanced'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">Game Mode</span> <span class="sp-val">{sys?.GameMode || 'Disabled'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">GPU Scheduling</span> <span class="sp-val">{sys?.GPUScheduling || 'Disabled'}</span></div>
                </div>

                <!-- MEMORY -->
                <div class="spec-card">
                    <div class="g-label">🧠 MEMORY</div>
                    <div class="divider"></div>
                    <div class="sp-row"><span class="sp-lbl">Total RAM</span> <span class="sp-val">{hw?.RAMTotal || '—'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">Used (Live)</span> <span class="sp-val" style="color:{pctColor(ramPct)}">{ramUsed} ({ramPct}%)</span></div>
                    <div class="sp-row"><span class="sp-lbl">Available</span> <span class="sp-val">{ramFree || '—'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">DIMM Slots</span> <span class="sp-val">{hw?.RAMSlots || '—'}</span></div>
                    {#if hw?.RAMDetails && hw.RAMDetails.length > 0}
                        {#each hw.RAMDetails as dimm, i}
                            <div class="sp-row"><span class="sp-lbl">Slot {i+1}</span> <span class="sp-val">{dimm || '—'}</span></div>
                        {/each}
                    {/if}
                </div>

                <!-- STORAGE DEVICES -->
                <div class="spec-card">
                    <div class="g-label">💽 STORAGE DEVICES</div>
                    <div class="divider"></div>
                    <div class="sp-row"><span class="sp-lbl">C: Usage (Live)</span> <span class="sp-val" style="color:{pctColor(diskPct)}; font-weight: 800;">{diskPct}%</span></div>
                    {#if storages.length > 0}
                        {#each storages as store}
                            <div class="sp-row">
                                <span class="sp-lbl">{store.Label || store.Name || 'Unknown Drive'}</span> 
                                <span class="sp-val">{store.Type || 'SSD'} {store.Size || ''}</span>
                            </div>
                        {/each}
                    {:else}
                        <div class="sp-row"><span class="sp-lbl">Primary Drive</span> <span class="sp-val">SSD</span></div>
                    {/if}
                </div>

                <!-- NETWORK -->
                <div class="spec-card">
                    <div class="g-label">🌐 NETWORK</div>
                    <div class="divider"></div>
                    <div class="sp-row"><span class="sp-lbl">IP Address</span> <span class="sp-val">{net?.IP || '—'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">MAC Address</span> <span class="sp-val">{net?.MAC || '—'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">Default Gateway</span> <span class="sp-val">{net?.Gateway || '—'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">Adapter</span> <span class="sp-val">{net?.Adapter || '—'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">Internet</span> <span class="sp-val" style="color: {net?.Internet ? '#10b981' : '#ef4444'}">{net?.Internet ? '✓ Connected' : '✗ Offline'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">Download (live)</span> <span class="sp-val">↓ {netInKBs < 1024 ? netInKBs + ' KB/s' : (netInKBs/1024).toFixed(1)+' MB/s'}</span></div>
                    <div class="sp-row"><span class="sp-lbl">Upload (live)</span> <span class="sp-val">↑ {netOutKBs < 1024 ? netOutKBs + ' KB/s' : (netOutKBs/1024).toFixed(1)+' MB/s'}</span></div>
                </div>

                <!-- SECURITY -->
                <div class="spec-card">
                    <div class="g-label">🔒 SECURITY</div>
                    <div class="divider"></div>

                    <!-- Real-time Protection -->
                    <div class="sp-row">
                        <span class="sp-lbl">Real-time Protect</span>
                        <div class="al-acts" style="display:flex;gap:6px;">
                            <button class="btn-enable" on:click={()=>doAction(()=>RunNativeAction('open-win-security',[]),'Open Windows Security')}>Open</button>
                        </div>
                    </div>

                    <!-- Firewall -->
                    <div class="sp-row">
                        <div class="al-name-row">
                            <span class="sp-lbl">Firewall</span>
                            <span class="st-badge" class:st-on={sec?.Firewall === true || sec?.Firewall === 'Enabled'} class:st-off={sec?.Firewall === false || sec?.Firewall === 'Disabled'}>
                                {sec?.Firewall === true || sec?.Firewall === 'Enabled' ? '● ENABLED' : '● DISABLED'}
                            </span>
                        </div>
                        <div class="al-acts" style="display:flex;gap:6px;">
                            <button class="btn-disable" on:click={()=>doSecurity('Firewall', 'Off', 'Turn Off Windows Firewall')}>Disable</button>
                            <button class="btn-enable" on:click={()=>doSecurity('Firewall', 'On', 'Turn On Windows Firewall')}>Enable</button>
                        </div>
                    </div>

                    <!-- Firewall Rules -->
                    <div class="sp-row">
                        <span class="sp-lbl">Firewall Rules</span>
                        <div class="al-acts" style="display:flex;gap:6px;">
                            <button class="btn-enable" on:click={()=>doAction(()=>RunNativeAction('open-firewall-rules',[]),'Open Firewall Rules')}>Open</button>
                        </div>
                    </div>
                </div>
            </div>
            
            <!-- TOP PROCESSES -->
            <div class="top-procs">
                <div class="tp-label">☑ TOP PROCESSES (LIVE)</div>
                <div class="tp-list">
                    {#if stats?.TopProcesses && stats.TopProcesses.length > 0}
                        {#each stats.TopProcesses.slice(0, 4) as proc}
                            <div class="tp-item"><span class="tp-name">{proc.Name}</span> <span class="tp-mem"><b>{proc.Mem}</b> MB</span></div>
                        {/each}
                    {:else}
                        <div class="tp-item"><span class="tp-name">system.exe</span> <span class="tp-mem"><b>—</b> MB</span></div>
                        <div class="tp-item"><span class="tp-name">browser.exe</span> <span class="tp-mem"><b>—</b> MB</span></div>
                    {/if}
                </div>
            </div>

        {:else if activeTab === "power"}
            <!-- CPU & POWER TAB -->
            <div class="power-wrap">
                <div class="section-title">POWER PLAN <span class="st-active">(Active: {activePlan.toUpperCase()})</span></div>
                <div class="power-cards">
                    <button class="p-btn" class:active={activePlan==='powersaver'} on:click={()=>doPowerPlan('powersaver')}>
                        🚶 Power Saver
                        <span class="p-hint">Max battery — reduce CPU clock</span>
                    </button>
                    <button class="p-btn" class:active={activePlan==='balanced'} on:click={()=>doPowerPlan('balanced')}>
                        ⚖ Balanced
                        <span class="p-hint">Auto-scale CPU — default Windows</span>
                    </button>
                    <button class="p-btn" class:active={activePlan==='high'} on:click={()=>doPowerPlan('high')}>
                        ⚡ High Performance
                        <span class="p-hint">Max CPU — no throttling</span>
                    </button>
                    <button class="p-btn" class:active={activePlan==='ultimate'} on:click={()=>doPowerPlan('ultimate')}>
                        🔥 Ultimate
                        <span class="p-hint">Full CPU potential — gaming & pro</span>
                    </button>
                </div>

                <div class="divider" style="margin: 16px 0;"></div>
                <button class="p-btn" on:click={() => doAction(HibernateHeavyApps, 'Game Mode Panic Switch')} style="border-color: #ef4444; color: #f87171; width:100%; display:flex; flex-direction:column; align-items:center; opacity: 0.9;">
                    🎮 Game Mode (Panic Switch)
                    <span class="p-hint" style="color:var(--text-dim); margin-top:4px;">Instantly suspends top 5 heaviest non-critical background apps securely</span>
                </button>

                <div class="section-title" style="margin-top: 20px;">VISUAL EFFECTS</div>
                <div class="action-list">
                    <!-- Windows Animations -->
                    <div class="al-item">
                        <div class="al-info">
                            <div class="al-name-row">
                                <span class="al-name">Windows Animations</span>
                                <span class="st-badge" class:st-on={st?.AnimationsEnabled} class:st-off={!st?.AnimationsEnabled}>
                                    {st?.AnimationsEnabled ? '● ENABLED' : '● DISABLED'}
                                </span>
                            </div>
                            <span class="al-desc">
                                <b>Disable:</b> Removes fade/slide/snap effects — instant UI. Faster feel on low-end PCs.<br/>
                                <b>Enable:</b> Smooth window transitions, minimize/restore animations, fade tooltips.
                            </span>
                        </div>
                        <div class="al-acts">
                            <button class="btn-disable" on:click={()=>doAction(()=>ToggleAnimations(false), 'Visual Effects › Animations')}>Disable</button>
                            <button class="btn-enable" on:click={()=>doAction(()=>ToggleAnimations(true), 'Visual Effects › Animations')}>Enable</button>
                        </div>
                    </div>
                    <!-- Transparency / Blur -->
                    <div class="al-item">
                        <div class="al-info">
                            <div class="al-name-row">
                                <span class="al-name">Transparency / Blur</span>
                                <span class="st-badge" class:st-on={st?.TransparencyEnabled} class:st-off={!st?.TransparencyEnabled}>
                                    {st?.TransparencyEnabled ? '● ENABLED' : '● DISABLED'}
                                </span>
                            </div>
                            <span class="al-desc">
                                <b>Disable:</b> Removes frosted glass blur from taskbar &amp; menus — reduces GPU usage.<br/>
                                <b>Enable:</b> Semi-transparent taskbar and Start menu with backdrop blur effect.
                            </span>
                        </div>
                        <div class="al-acts">
                            <button class="btn-disable" on:click={()=>doAction(()=>ApplyPerfTweak('transparency',false),'Visual Effects › Transparency')}>Disable</button>
                            <button class="btn-enable" on:click={()=>doAction(()=>ApplyPerfTweak('transparency',true),'Visual Effects › Transparency')}>Enable</button>
                        </div>
                    </div>
                    <!-- Drop Shadows -->
                    <div class="al-item">
                        <div class="al-info">
                            <div class="al-name-row">
                                <span class="al-name">Drop Shadows</span>
                                <span class="st-badge st-neutral">● REGISTRY</span>
                            </div>
                            <span class="al-desc">
                                <b>Disable:</b> Removes shadow under windows and desktop icons — cleaner look, tiny perf gain.<br/>
                                <b>Enable:</b> Adds depth shadow under active windows and icon labels.
                            </span>
                        </div>
                        <div class="al-acts">
                            <button class="btn-disable" on:click={()=>doAction(()=>ApplyPerfTweak('dropshadow',false),'Visual Effects › Drop Shadows')}>Disable</button>
                            <button class="btn-enable" on:click={()=>doAction(()=>ApplyPerfTweak('dropshadow',true),'Visual Effects › Drop Shadows')}>Enable</button>
                        </div>
                    </div>
                    <!-- Taskbar Thumbnail Previews -->
                    <div class="al-item">
                        <div class="al-info">
                            <div class="al-name-row">
                                <span class="al-name">Taskbar Thumbnail Previews</span>
                                <span class="st-badge st-neutral">● REGISTRY</span>
                            </div>
                            <span class="al-desc">
                                <b>Disable:</b> Removes hover window preview on taskbar — saves memory, reduces load.<br/>
                                <b>Enable:</b> Shows live mini-preview of open windows when hovering the taskbar icon.
                            </span>
                        </div>
                        <div class="al-acts">
                            <button class="btn-disable" on:click={()=>doAction(()=>ApplyPerfTweak('thumbpreviews',false),'Visual Effects › Thumbnails')}>Disable</button>
                            <button class="btn-enable" on:click={()=>doAction(()=>ApplyPerfTweak('thumbpreviews',true),'Visual Effects › Thumbnails')}>Enable</button>
                        </div>
                    </div>
                </div>
                <div class="section-title">SYSTEM TWEAKS</div>
                <div class="action-list">
                    <!-- Game Mode -->
                    <div class="al-item">
                        <div class="al-info">
                            <div class="al-name-row">
                                <span class="al-name">Game Mode</span>
                                <span class="st-badge" class:st-on={st?.GameModeEnabled} class:st-off={!st?.GameModeEnabled}>
                                    {st?.GameModeEnabled ? '● ENABLED' : '● DISABLED'}
                                </span>
                            </div>
                            <span class="al-desc">
                                <b>Disable:</b> Normal CPU/GPU scheduler — other background apps compete equally.<br/>
                                <b>Enable:</b> Prioritises the foreground game for CPU &amp; GPU time, reduces stutters.
                            </span>
                        </div>
                        <div class="al-acts">
                            <button class="btn-disable" on:click={()=>doAction(()=>ApplyPerfTweak('gamemode',false),'System Tweaks › Game Mode')}>Disable</button>
                            <button class="btn-enable" on:click={()=>doAction(()=>ApplyPerfTweak('gamemode',true),'System Tweaks › Game Mode')}>Enable</button>
                        </div>
                    </div>
                    <!-- HAGS -->
                    <div class="al-item">
                        <div class="al-info">
                            <div class="al-name-row">
                                <span class="al-name">HW GPU Scheduling (HAGS)</span>
                                <span class="st-badge" class:st-on={st?.HAGSEnabled} class:st-off={!st?.HAGSEnabled}>
                                    {st?.HAGSEnabled ? '● ENABLED' : '● DISABLED'}
                                </span>
                            </div>
                            <span class="al-desc">
                                <b>Disable:</b> CPU manages GPU work queue — more compatible, less latency risk.<br/>
                                <b>Enable:</b> GPU manages its own work queue — reduces latency by ~1 frame. <em>Reboot required.</em>
                            </span>
                        </div>
                        <div class="al-acts">
                            <button class="btn-disable" on:click={()=>doAction(()=>ApplyPerfTweak('hags',false),'System Tweaks › HAGS')}>Disable</button>
                            <button class="btn-enable" on:click={()=>doAction(()=>ApplyPerfTweak('hags',true),'System Tweaks › HAGS')}>Enable</button>
                        </div>
                    </div>
                    <!-- Background Apps -->
                    <div class="al-item">
                        <div class="al-info">
                            <div class="al-name-row">
                                <span class="al-name">Background Apps</span>
                                <span class="st-badge" class:st-on={st?.BackgroundAppsEnabled} class:st-off={!st?.BackgroundAppsEnabled}>
                                    {st?.BackgroundAppsEnabled ? '● ENABLED' : '● DISABLED'}
                                </span>
                            </div>
                            <span class="al-desc">
                                <b>Disable:</b> Prevents UWP/Store apps from running, fetching data, or sending notifications in background. Saves RAM &amp; battery.<br/>
                                <b>Enable:</b> Allows apps like Mail, News, Calendar to update &amp; sync silently.
                            </span>
                        </div>
                        <div class="al-acts">
                            <button class="btn-disable" on:click={()=>doAction(()=>ToggleBackgroundApps(false),'System Tweaks › Background Apps')}>Disable</button>
                            <button class="btn-enable" on:click={()=>doAction(()=>ToggleBackgroundApps(true),'System Tweaks › Background Apps')}>Enable</button>
                        </div>
                    </div>
                    <!-- Focus Assist -->
                    <div class="al-item">
                        <div class="al-info">
                            <div class="al-name-row">
                                <span class="al-name">Focus Assist (Do Not Disturb)</span>
                                <span class="st-badge st-neutral">● REGISTRY</span>
                            </div>
                            <span class="al-desc">
                                <b>Disable:</b> All notification banners appear normally — no suppression.<br/>
                                <b>Enable:</b> Silently blocks notification popups while you work or game.
                            </span>
                        </div>
                        <div class="al-acts">
                            <button class="btn-disable" on:click={()=>doAction(()=>ApplyPerfTweak('focusassist',false),'System Tweaks › Focus Assist')}>Disable</button>
                            <button class="btn-enable" on:click={()=>doAction(()=>ApplyPerfTweak('focusassist',true),'System Tweaks › Focus Assist')}>Enable</button>
                        </div>
                    </div>
                </div>
            </div>

        {:else if activeTab === "memory"}
            <!-- MEMORY TAB -->
            <div class="power-wrap">
                <div class="section-title">FREE RAM IMMEDIATELY</div>
                <div class="power-cards">
                    <button class="ram-btn" on:click={()=>doAction(ClearStandbyMemory, 'Memory › Clear Standby')} disabled={executing}>
                        🧹 <div><b>Clear Standby Memory</b><br/><span>Flushes the standby list, making RAM available for active processes.</span></div>
                    </button>
                    <button class="ram-btn" on:click={()=>doAction(TrimProcessWorkingSets, 'Memory › Trim Working Sets')} disabled={executing}>
                        ✂️ <div><b>Trim Working Sets</b><br/><span>Forces all running applications to release cached memory pages.</span></div>
                    </button>
                </div>

                <div class="section-title">MEMORY SERVICES</div>
                <div class="action-list">
                    <!-- SysMain -->
                    <div class="al-item">
                        <div class="al-info">
                            <div class="al-name-row">
                                <span class="al-name">SysMain / Superfetch</span>
                                <span class="st-badge" class:st-on={st?.SysMainEnabled} class:st-off={!st?.SysMainEnabled}>
                                    {st?.SysMainEnabled ? '● ENABLED' : '● DISABLED'}
                                </span>
                            </div>
                            <span class="al-desc">
                                <b>Disable:</b> Stops preloading apps into RAM — recommended on SSDs, reduces disk writes.<br/>
                                <b>Enable:</b> Preloads frequently used apps into RAM for faster launch — better on HDDs.
                            </span>
                        </div>
                        <div class="al-acts">
                            <button class="btn-disable" on:click={()=>doAction(()=>ApplyPerfTweak('sysmain',false),'Memory › SysMain')}>Disable</button>
                            <button class="btn-enable" on:click={()=>doAction(()=>ApplyPerfTweak('sysmain',true),'Memory › SysMain')}>Enable</button>
                        </div>
                    </div>
                    <!-- Memory Compression -->
                    <div class="al-item">
                        <div class="al-info">
                            <div class="al-name-row">
                                <span class="al-name">Memory Compression</span>
                                <span class="st-badge" class:st-on={st?.MemCompressionEnabled} class:st-off={!st?.MemCompressionEnabled}>
                                    {st?.MemCompressionEnabled ? '● ENABLED' : '● DISABLED'}
                                </span>
                            </div>
                            <span class="al-desc">
                                <b>Disable:</b> Windows uses disk pagefile more — may cause more disk I/O but less CPU.<br/>
                                <b>Enable:</b> Compresses in-memory pages using CPU — reduces pagefile usage, keeps more data in RAM.
                            </span>
                        </div>
                        <div class="al-acts">
                            <button class="btn-disable" on:click={()=>doAction(()=>ApplyPerfTweak('memcompression',false),'Memory › Compression')}>Disable</button>
                            <button class="btn-enable" on:click={()=>doAction(()=>ApplyPerfTweak('memcompression',true),'Memory › Compression')}>Enable</button>
                        </div>
                    </div>
                    <!-- Prefetch -->
                    <div class="al-item">
                        <div class="al-info">
                            <div class="al-name-row">
                                <span class="al-name">Prefetch</span>
                                <span class="st-badge" class:st-on={st?.PrefetchEnabled} class:st-off={!st?.PrefetchEnabled}>
                                    {st?.PrefetchEnabled ? '● ENABLED' : '● DISABLED'}
                                </span>
                            </div>
                            <span class="al-desc">
                                <b>Disable:</b> Stops pre-caching app data from disk — useful on SSDs. Frees disk activity.<br/>
                                <b>Enable:</b> Pre-caches app launch data — speeds up repeated app starts, especially on HDDs.
                            </span>
                        </div>
                        <div class="al-acts">
                            <button class="btn-disable" on:click={()=>doAction(()=>ApplyPerfTweak('prefetch',false),'Memory › Prefetch')}>Disable</button>
                            <button class="btn-enable" on:click={()=>doAction(()=>ApplyPerfTweak('prefetch',true),'Memory › Prefetch')}>Enable</button>
                        </div>
                    </div>
                    <!-- Large System Cache -->
                    <div class="al-item">
                        <div class="al-info">
                            <div class="al-name-row">
                                <span class="al-name">Large System Cache</span>
                                <span class="st-badge st-neutral">● REGISTRY</span>
                            </div>
                            <span class="al-desc">
                                <b>Disable (Desktop):</b> Optimises RAM for running applications — standard Windows desktop mode.<br/>
                                <b>Enable (Server):</b> Allocates more RAM to filesystem cache — better for file servers, databases.
                            </span>
                        </div>
                        <div class="al-acts">
                            <button class="btn-disable" on:click={()=>doAction(()=>ApplyPerfTweak('largecache',false),'Memory › Large Cache')}>Disable</button>
                            <button class="btn-enable" on:click={()=>doAction(()=>ApplyPerfTweak('largecache',true),'Memory › Large Cache')}>Enable</button>
                        </div>
                    </div>
                </div>

                <div class="section-title" style="margin-top: 20px;">VIRTUAL MEMORY — PAGEFILE</div>
                <div class="pf-box">
                    <div class="pf-stat">Current: <b>{stats?.VirtualMem?.Total || 'System Managed'}</b></div>
                    <div class="pf-btns">
                        <button class="pf-btn" on:click={()=>doAction(()=>ConfigurePageFile(0,0), 'Memory › Pagefile Auto')}>Auto (System)</button>
                        <button class="pf-btn" on:click={()=>doAction(()=>ConfigurePageFile(4096,4096), 'Memory › Pagefile 4GB')}>4 GB</button>
                        <button class="pf-btn" on:click={()=>doAction(()=>ConfigurePageFile(8192,8192), 'Memory › Pagefile 8GB')}>8 GB</button>
                        <button class="pf-btn" on:click={()=>doAction(()=>ConfigurePageFile(16384,16384), 'Memory › Pagefile 16GB')}>16 GB</button>
                    </div>
                </div>
            </div>

        {:else if activeTab === "network"}
            <!-- NETWORK TAB -->
            <div class="power-wrap">
                <div class="section-title">CONNECTION INFO</div>
                <div class="net-grid">
                    <div class="net-box">
                        <span class="nb-lbl">IP ADDRESS</span>
                        <span class="nb-val">{net?.IP || '—'}</span>
                    </div>
                    <div class="net-box">
                        <span class="nb-lbl">MAC ADDRESS</span>
                        <span class="nb-val">{net?.MAC || '—'}</span>
                    </div>
                    <div class="net-box">
                        <span class="nb-lbl">GATEWAY</span>
                        <span class="nb-val">{net?.Gateway || '—'}</span>
                    </div>
                    <div class="net-box">
                        <span class="nb-lbl">STATUS</span>
                        <span class="nb-val" style="color: {net?.Internet ? '#10b981' : '#ef4444'}">
                            {net?.Internet ? '✓ Connected' : '✗ Offline'}
                        </span>
                    </div>
                </div>

                <div class="section-title">PING TEST</div>
                <div class="ping-row">
                    <input type="text" class="ping-input" bind:value={pingTarget} />
                    <button class="ping-btn" on:click={runPing} disabled={pinging}>
                        {#if pinging}
                            <span class="spin">⏳</span> Pinging...
                        {:else}
                            Ping
                        {/if}
                    </button>
                    <div class="ping-presets">
                        <button class="pp-btn" on:click={()=>{pingTarget='8.8.8.8'; runPing();}}>8.8.8.8</button>
                        <button class="pp-btn" on:click={()=>{pingTarget='1.1.1.1'; runPing();}}>1.1.1.1</button>
                        <button class="pp-btn" on:click={()=>{pingTarget='google.com'; runPing();}}>google.com</button>
                    </div>
                </div>
                {#if pingResult}
                    <div class="ping-res">{pingResult}</div>
                {/if}

                <div class="section-title">NETWORK TOOLS</div>
                <div class="action-list">
                    <div class="al-item">
                        <div class="al-info">
                            <span class="al-name">Allow ICMP Ping</span>
                            <span class="al-desc">Adds a Windows Firewall inbound rule to allow ICMPv4 — lets other machines ping this PC.</span>
                        </div>
                        <button class="btn-execute" on:click={()=>doAction(AllowICMPPing,'Network › Allow ICMP Ping')}>Execute</button>
                    </div>
                    <div class="al-item">
                        <div class="al-info">
                            <span class="al-name">Flush DNS Cache</span>
                            <span class="al-desc">Runs <code>ipconfig /flushdns</code> — clears cached DNS records, fixes wrong-IP resolution issues.</span>
                        </div>
                        <button class="btn-execute" on:click={()=>doAction(FlushDNS,'Network › Flush DNS')}>Execute</button>
                    </div>
                    <div class="al-item">
                        <div class="al-info">
                            <span class="al-name">Renew IP (DHCP)</span>
                            <span class="al-desc">Runs <code>ipconfig /release</code> then <code>/renew</code> — requests a fresh IP from the DHCP router.</span>
                        </div>
                        <button class="btn-execute" on:click={()=>doAction(RenewIP,'Network › Renew IP')}>Execute</button>
                    </div>
                    <div class="al-item">
                        <div class="al-info">
                            <span class="al-name">Winsock Reset</span>
                            <span class="al-desc">Runs <code>netsh winsock reset</code> — fixes corrupt socket catalog. Resolves "Unable to connect" errors. <em>Reboot required.</em></span>
                        </div>
                        <button class="btn-execute" on:click={()=>doAction(WinsockReset,'Network › Winsock Reset')}>Execute</button>
                    </div>
                    <div class="al-item">
                        <div class="al-info">
                            <span class="al-name">TCP/IP Stack Reset</span>
                            <span class="al-desc">Runs <code>netsh int ip reset</code> &mdash; full TCP/IP stack reset. Fixes deep network corruption. <em>Reboot required.</em></span>
                        </div>
                        <button class="btn-execute" on:click={()=>doAction(TCPIPReset,'Network › TCP/IP Reset')}>Execute</button>
                    </div>
                </div>
            </div>
        {:else if activeTab === "processes"}
            <!-- PROCESS MANAGER TAB -->
            <div class="power-wrap">
                <div style="display:flex; align-items:center; gap:12px; margin-bottom:12px;">
                    <div class="section-title" style="margin:0;">⚙️ ALL RUNNING PROCESSES</div>
                    <input class="ping-input" style="flex:1; max-width:280px;" placeholder="Filter by name..." bind:value={procFilter}/>
                    <button class="btn-execute" on:click={loadProcesses} disabled={procsLoading}>🔄 Refresh</button>
                </div>
                <div class="proc-table">
                    <div class="proc-header">
                        <span>Process Name</span><span>PID</span><span>RAM (MB)</span><span>Action</span>
                    </div>
                    {#if procsLoading}
                        <div class="proc-loading">Loading processes...</div>
                    {:else}
                        {#each processes.filter(p => !procFilter || p.Name.toLowerCase().includes(procFilter.toLowerCase())).slice(0, 100) as proc}
                        <div class="proc-row">
                            <span class="proc-name">{proc.Name}</span>
                            <span class="proc-pid">{proc.PID}</span>
                            <span class="proc-mem" style="color:{proc.Mem>500?'#ef4444':proc.Mem>200?'#f59e0b':'#10b981'}">{proc.Mem}</span>
                            <button class="btn-kill" on:click={()=>doAction(()=>KillProcess(proc.PID), 'Kill '+proc.Name)} disabled={executing}>Kill</button>
                        </div>
                        {/each}
                    {/if}
                </div>
            </div>

        {:else if activeTab === "drivers"}
            <!-- DRIVER HEALTH TAB -->
            <div class="power-wrap">
                <div style="display:flex; align-items:center; gap:12px; margin-bottom:12px;">
                    <div class="section-title" style="margin:0;">🔌 DRIVER HEALTH CHECKER</div>
                    <button class="btn-execute" on:click={loadDrivers} disabled={driversLoading}>🔄 Scan</button>
                </div>
                {#if driversLoading}
                    <div class="proc-loading">Scanning drivers...</div>
                {:else if drivers.length === 0}
                    <div class="proc-loading" style="color:#10b981;">✅ All drivers appear healthy — no issues found.</div>
                {:else}
                    <div class="proc-table">
                        <div class="proc-header">
                            <span>Driver Name</span><span>Class</span><span>Version</span><span>Status</span>
                        </div>
                        {#each drivers as d}
                        <div class="proc-row">
                            <span class="proc-name">{d.Name}</span>
                            <span class="proc-pid">{d.Class}</span>
                            <span class="proc-pid" style="font-size:0.7rem;">{d.Version}</span>
                            <span class="st-badge" class:st-on={d.Status==='OK'} class:st-off={d.Status==='Error'} class:st-neutral={d.Status==='Degraded'}>● {d.Status}</span>
                        </div>
                        {/each}
                    </div>
                {/if}
            </div>

        {:else if activeTab === "boot"}
            <!-- BOOT TIME ANALYZER TAB -->
            <div class="power-wrap">
                <div style="display:flex; align-items:center; gap:12px; margin-bottom:12px;">
                    <div class="section-title" style="margin:0;">🚀 BOOT STARTUP ANALYZER</div>
                    <button class="btn-execute" on:click={loadBoot} disabled={bootLoading}>🔄 Scan Boot</button>
                </div>
                {#if bootLoading}
                    <div class="proc-loading">Analyzing startup items...</div>
                {:else if bootItems.length === 0}
                    <div class="proc-loading">No startup items found or scan not started.</div>
                {:else}
                    <div class="proc-table">
                        <div class="proc-header">
                            <span>Application</span><span>Impact</span><span>Timing</span>
                        </div>
                        {#each bootItems as b}
                        <div class="proc-row">
                            <span class="proc-name">{b.App}</span>
                            <span class="st-badge" class:st-on={b.Impact==='Low'} class:st-neutral={b.Impact==='Medium'} class:st-off={b.Impact==='High'}>● {b.Impact}</span>
                            <span class="proc-pid" style="color:var(--text-lo);">{b.Delay}</span>
                        </div>
                        {/each}
                    </div>
                {/if}
            </div>

        {:else if activeTab === "tools"}
            <!-- TOOLS TAB: Windows Update + Auto Scheduler -->
            <div class="power-wrap">
                <div class="section-title">🪟 WINDOWS UPDATE CONTROL</div>
                <div class="action-list">
                    <div class="al-item">
                        <div class="al-info">
                            <div class="al-name-row">
                                <span class="al-name">Pause Windows Update</span>
                                <span class="st-badge st-neutral">● REGISTRY + SERVICE</span>
                            </div>
                            <span class="al-desc">Prevents forced updates for a set number of days. Useful before important presentations or gaming sessions.</span>
                        </div>
                        <div class="al-acts" style="flex-direction:column; gap:6px; align-items:flex-end;">
                            <div style="display:flex; align-items:center; gap:8px;">
                                <span style="font-size:0.78rem; color:var(--text-dim);">Days:</span>
                                <select class="ping-input" style="width:80px; padding:6px 8px;" bind:value={updatePauseDays}>
                                    <option value={7}>7</option>
                                    <option value={14}>14</option>
                                    <option value={21}>21</option>
                                    <option value={35}>35</option>
                                </select>
                            </div>
                            <div style="display:flex; gap:6px;">
                                <button class="btn-disable" on:click={()=>doAction(()=>PauseWindowsUpdate(updatePauseDays), 'Pause Windows Update')}>Pause</button>
                                <button class="btn-enable" on:click={()=>doAction(()=>PauseWindowsUpdate(0), 'Resume Windows Update')}>Resume</button>
                            </div>
                        </div>
                    </div>
                </div>

                <div class="divider" style="margin: 16px 0;"></div>

                <div class="section-title">⏰ AUTO CLEANUP SCHEDULER</div>
                <div class="action-list">
                    <div class="al-item">
                        <div class="al-info">
                            <div class="al-name-row">
                                <span class="al-name">Scheduled RAM + Junk Cleanup</span>
                                <span class="st-badge st-neutral">● WINDOWS TASK SCHEDULER</span>
                            </div>
                            <span class="al-desc">Creates a silent background Windows Task that auto-clears Temp files, Recycle Bin, and DNS cache on a recurring schedule.</span>
                        </div>
                        <div class="al-acts" style="flex-direction:column; gap:6px; align-items:flex-end;">
                            <div style="display:flex; align-items:center; gap:8px;">
                                <span style="font-size:0.78rem; color:var(--text-dim);">Every:</span>
                                <select class="ping-input" style="width:90px; padding:6px 8px;" bind:value={scheduleHours}>
                                    <option value={6}>6 hrs</option>
                                    <option value={12}>12 hrs</option>
                                    <option value={24}>24 hrs</option>
                                    <option value={48}>48 hrs</option>
                                </select>
                            </div>
                            <div style="display:flex; gap:6px;">
                                <button class="btn-enable" on:click={()=>doAction(()=>ScheduleAutoCleanup(scheduleHours),'Enable Auto Cleanup')}>Enable</button>
                                <button class="btn-disable" on:click={()=>doAction(()=>ScheduleAutoCleanup(0),'Disable Auto Cleanup')}>Disable</button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

                <div class="divider" style="margin: 16px 0;"></div>

                <div class="section-title">🔄 RESTART WINDOWS SERVICES</div>
                <div class="action-list">
                    <div class="al-item">
                        <div class="al-info">
                            <span class="al-name">🖨️ Print Spooler</span>
                            <span class="al-desc">Fixes printer stuck jobs and "Printer not found" errors. Safe to restart anytime.</span>
                        </div>
                        <button class="btn-execute" on:click={()=>doAction(()=>RunNativeAction('restart-service',['Spooler']),'Restart Print Spooler')}>Restart</button>
                    </div>
                    <div class="al-item">
                        <div class="al-info">
                            <span class="al-name">🔄 Windows Update</span>
                            <span class="al-desc">Fixes stuck Windows Update downloads and installations. Resets the update agent.</span>
                        </div>
                        <button class="btn-execute" on:click={()=>doAction(()=>RunNativeAction('restart-service',['wuauserv']),'Restart Windows Update')}>Restart</button>
                    </div>
                    <div class="al-item">
                        <div class="al-info">
                            <span class="al-name">🖥️ Remote Desktop (RDP)</span>
                            <span class="al-desc">Fixes dropped RDP sessions or connection refused errors without rebooting.</span>
                        </div>
                        <button class="btn-execute" on:click={()=>doAction(()=>RunNativeAction('restart-service',['TermService']),'Restart RDP Service')}>Restart</button>
                    </div>
                    <div class="al-item">
                        <div class="al-info">
                            <span class="al-name">📡 DHCP Client</span>
                            <span class="al-desc">Fixes "No Internet Access" and IP assignment issues without running ipconfig commands.</span>
                        </div>
                        <button class="btn-execute" on:click={()=>doAction(()=>RunNativeAction('restart-service',['Dhcp']),'Restart DHCP Client')}>Restart</button>
                    </div>
                    <div class="al-item">
                        <div class="al-info">
                            <span class="al-name">🔐 Windows Defender (WinDefend)</span>
                            <span class="al-desc">Restarts the Defender anti-malware service if real-time protection is stuck or reporting errors.</span>
                        </div>
                        <button class="btn-execute" on:click={()=>doAction(()=>RunNativeAction('restart-service',['WinDefend']),'Restart Windows Defender')}>Restart</button>
                    </div>
                </div>

        {/if}
    </div>

    <!-- Feedback Toast -->
    {#if feedback}
        <div class="f-toast slide-up">{feedback}</div>
    {/if}

</div>

<style>
    /* ── LAYOUT ── */
    .dash {
        display: flex;
        flex-direction: column;
        gap: 16px;
        min-height: 100%;
        width: 100%;
    }

    /* ── STAT CARDS ── */
    .stat-row {
        display: flex; gap: 12px;
    }
    .stat-card {
        flex: 1; background: rgba(255,255,255,0.025);
        border: 1px solid rgba(255,255,255,0.06); border-radius: 10px;
        padding: 16px; display: flex; flex-direction: column; gap: 4px;
        position: relative; overflow: hidden;
    }
    .sc-top { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
    .sc-label { font-size: 0.65rem; font-weight: 800; letter-spacing: 1.5px; color: var(--text-dim); }
    .sc-value { font-size: 1.35rem; font-weight: 800; }
    .sc-sub { font-size: 0.75rem; color: var(--text-dim); }
    .sc-bot { font-size: 0.72rem; color: var(--text-lo); margin-top: auto; }
    .sc-bar {
        position: absolute; bottom: 0; left: 0; right: 0; height: 3px; background: rgba(255,255,255,0.05);
    }
    .sc-fill { height: 100%; transition: width 0.3s ease; }

    .net-status { font-size: 0.8rem; font-weight: 800; }
    .net-status.on { color: #10b981; text-shadow: 0 0 10px rgba(16,185,129,0.3); }
    .net-status.off { color: #ef4444; text-shadow: 0 0 10px rgba(239,68,68,0.3); }

    /* ── TABS ── */
    .dash-tabs {
        display: flex; gap: 8px; border-bottom: 1px solid rgba(255,255,255,0.07);
        padding-bottom: 2px;
    }
    .d-tab {
        font-family: inherit; font-size: 0.8rem; font-weight: 700;
        padding: 8px 16px; background: transparent; border: none;
        color: var(--text-dim); cursor: pointer; border-radius: 20px;
        transition: all 0.2s;
    }
    .d-tab:hover { background: rgba(255,255,255,0.05); color: var(--text); }
    .d-tab.active { background: rgba(124,58,237,0.15); color: #c4b5fd; border: 1px solid rgba(124,58,237,0.3); }

    /* ── TAB CONTENT CONTAINERS ── */
    .tab-content {
        flex: 1; padding: 4px 0 20px 0;
    }

    /* ── SPECS GRID ── */
    .specs-grid {
        display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px;
    }
    .spec-card {
        background: rgba(255,255,255,0.02); border: 1px solid var(--border);
        border-radius: 10px; padding: 16px 20px;
    }
    .g-label { font-size: 0.7rem; font-weight: 800; letter-spacing: 1px; color: var(--text-lo); margin-bottom: 8px; }
    .divider { height: 1px; background: var(--border); margin-bottom: 12px; }
    .sp-row { display: flex; justify-content: space-between; margin-bottom: 8px; font-size: 0.82rem; }
    .sp-lbl { color: var(--text-dim); }
    .sp-val { font-weight: 600; color: var(--text); }

    /* ── POWER & MEMORY WRAPPER ── */
    .power-wrap {
        background: rgba(255,255,255,0.02);
        border: 1px solid var(--border);
        border-radius: 10px;
        padding: 24px 32px;
        display: flex;
        flex-direction: column;
        gap: 20px;
    }
    .section-title {
        font-size: 0.7rem; font-weight: 800; letter-spacing: 2px; color: var(--text-lo);
        text-align: center; border-bottom: 1px solid var(--border-md);
        padding-bottom: 8px; margin-bottom: 8px; margin-top: 10px;
    }

    /* ── HORIZONTAL CARDS (POWER PLAN / CLEAR RAM) ── */
    .power-cards {
        display: flex; justify-content: center; gap: 14px; flex-wrap: wrap; margin-bottom: 10px;
    }
    .p-btn {
        padding: 12px 24px; border-radius: 24px;
        background: rgba(255,255,255,0.03); border: 1px solid var(--border);
        color: var(--text-dim); font-size: 0.82rem; font-weight: 700; cursor: pointer;
        transition: all 0.2s;
    }
    .p-btn:hover { background: rgba(255,255,255,0.08); color: var(--text); }
    .p-btn.active { background: rgba(124,58,237,0.15); border-color: #7C3AED; color: #fff; }

    .ram-btn {
        display: flex; align-items: center; gap: 12px;
        padding: 12px 24px; border-radius: 10px;
        background: rgba(255,255,255,0.03); border: 1px solid var(--border);
        color: var(--text); text-align: left; cursor: pointer; transition: all 0.2s;
    }
    .ram-btn:hover { background: rgba(255,255,255,0.08); border-color: rgba(255,255,255,0.2); }
    .ram-btn:disabled { opacity: 0.5; cursor: default; }
    .ram-btn b { font-size: 0.85rem; color: var(--text); }
    .ram-btn span { font-size: 0.7rem; color: var(--text-dim); }

    /* ── LIST ROWS (VISUAL EFFECTS, TWEAKS) ── */
    .action-list {
        display: flex; flex-direction: column; gap: 12px;
    }
    .al-item {
        display: flex; justify-content: space-between; align-items: center;
        background: rgba(255,255,255,0.015); border: 1px solid var(--border);
        padding: 14px 20px; border-radius: 8px; transition: background 0.2s;
    }
    .al-item:hover { background: rgba(255,255,255,0.03); }
    .al-info { display: flex; flex-direction: column; gap: 3px; }
    .al-name { font-size: 0.85rem; font-weight: 700; color: var(--text); }
    .al-desc { font-size: 0.7rem; color: var(--text-dim); }
    .al-acts { display: flex; gap: 8px; align-items: center; }

    .btn-disable, .btn-enable, .btn-execute {
        font-family: var(--font); font-size: 0.75rem; font-weight: 700;
        padding: 7px 16px; border-radius: 6px; cursor: pointer; transition: all 0.2s;
    }
    .btn-disable { background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.1); color: var(--text-dim); }
    .btn-disable:hover { background: rgba(255,255,255,0.1); color: var(--text); }
    .btn-enable { background: rgba(124,58,237,0.1); border: 1px solid rgba(124,58,237,0.4); color: #c4b5fd; }
    .btn-enable:hover { background: rgba(124,58,237,0.2); color: #fff; }
    .btn-execute { background: rgba(34,211,238,0.1); border: 1px solid rgba(34,211,238,0.4); color: #22D3EE; }
    .btn-execute:hover { background: rgba(34,211,238,0.2); color: #fff; }

    /* ── NETWORK GRID & PING ── */
    .net-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1px; background: var(--border); border: 1px solid var(--border); border-radius: 8px; overflow: hidden; }
    .net-box { background: var(--bg-card); display: flex; flex-direction: column; align-items: center; padding: 20px; gap: 6px; }
    .nb-lbl { font-size: 0.65rem; font-weight: 800; letter-spacing: 1px; color: var(--text-lo); }
    .nb-val { font-size: 0.95rem; font-weight: 700; color: var(--text); }

    .ping-row { display: flex; gap: 10px; align-items: center; justify-content: center; }
    .ping-input { background: rgba(0,0,0,0.3); border: 1px solid var(--border-md); color: #fff; padding: 10px 14px; border-radius: 6px; width: 200px; font-family: monospace; }
    .ping-btn { background: #7C3AED; color: #fff; border: none; padding: 10px 24px; border-radius: 6px; font-weight: 700; cursor: pointer; transition: all 0.2s; }
    .ping-btn:hover:not(:disabled) { box-shadow: 0 0 14px rgba(124,58,237,0.5); }
    .ping-btn:disabled { opacity: 0.5; cursor: default; }
    .ping-presets { display: flex; gap: 6px; margin-left: 10px; }
    .pp-btn { background: rgba(255,255,255,0.05); border: 1px solid var(--border); color: var(--text-dim); padding: 6px 12px; border-radius: 4px; font-size: 0.75rem; cursor: pointer; }
    .pp-btn:hover { background: rgba(255,255,255,0.1); color: var(--text); }
    .ping-res { text-align: center; margin-top: 10px; font-family: monospace; font-size: 0.8rem; color: #10b981; background: rgba(0,0,0,0.3); padding: 12px; border-radius: 6px; }

    /* ── PAGEFILE ── */
    .pf-box { display: flex; flex-direction: column; align-items: center; gap: 16px; margin-bottom: 20px; }
    .pf-stat { font-size: 0.85rem; color: var(--text-dim); }
    .pf-stat b { font-size: 0.95rem; color: var(--text); }
    .pf-btns { display: flex; gap: 8px; flex-wrap: wrap; justify-content: center; }
    .pf-btn { background: rgba(255,255,255,0.03); border: 1px solid var(--border); color: var(--text-dim); padding: 10px 20px; border-radius: 20px; font-size: 0.8rem; font-weight: 600; cursor: pointer; transition: all 0.2s; }
    .pf-btn:hover { background: rgba(255,255,255,0.1); color: var(--text); }

    /* ── TOAST ── */
    .f-toast {
        position: fixed; bottom: 30px; left: 50%; transform: translateX(-50%);
        background: #1e293b; border: 1px solid #334155; padding: 12px 24px;
        border-radius: 30px; font-size: 0.85rem; font-weight: 700; color: #fff;
        box-shadow: 0 10px 30px rgba(0,0,0,0.5); z-index: 1000;
    }
    .slide-up { animation: slideUp 0.3s ease-out forwards; }
    @keyframes slideUp { from { opacity: 0; transform: translate(-50%, 20px); } to { opacity: 1; transform: translate(-50%, 0); } }
    .spin { display: inline-block; animation: spin 1s linear infinite; }
    @keyframes spin { 100% { transform: rotate(360deg); } }

    /* ── STATUS BADGES ── */
    .al-name-row { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
    .st-badge {
        font-size: 9px; font-weight: 800; letter-spacing: 1.2px;
        padding: 2px 7px; border-radius: 20px; white-space: nowrap;
    }
    .st-on  { color: #10b981; background: rgba(16,185,129,0.12); border: 1px solid rgba(16,185,129,0.3); }
    .st-off { color: #ef4444; background: rgba(239,68,68,0.12);  border: 1px solid rgba(239,68,68,0.3);  }
    .st-neutral { color: #a78bfa; background: rgba(167,139,250,0.1); border: 1px solid rgba(167,139,250,0.25); }
    .st-active { color: #a78bfa; font-size: 0.7rem; font-weight: 600; opacity: 0.85; }
    /* ── POWER PLAN HINTS ── */
    .p-btn { display: flex; flex-direction: column; align-items: center; gap: 2px; }
    .p-hint { font-size: 0.65rem; opacity: 0.6; font-weight: 400; }

    /* ── TOP PROCESSES ── */
    .top-procs {
        display: flex; align-items: center; gap: 16px; margin-top: 10px;
        background: rgba(255,255,255,0.02); padding: 12px 20px; border-radius: 8px;
        border: 1px solid var(--border); width: 100%; flex-wrap: wrap;
    }
    .tp-label { font-size: 0.72rem; font-weight: 800; color: #a78bfa; white-space: nowrap; margin-bottom: 4px; }
    .tp-list { display: flex; gap: 14px; flex-wrap: wrap; flex: 1; }
    .tp-item {
        display: flex; align-items: center; gap: 8px; font-size: 0.75rem;
        background: rgba(0,0,0,0.2); padding: 6px 12px; border-radius: 20px;
        white-space: nowrap; border: 1px solid rgba(255,255,255,0.05);
    }
    .tp-name { color: var(--text-dim); }
    .tp-mem { color: var(--text); }

    /* ── HEALTH STRIP ── */
    .health-strip {
        display: flex; align-items: center; gap: 16px;
        background: rgba(255,255,255,0.03); border: 1px solid var(--border);
        border-radius: 10px; padding: 10px 18px;
    }
    .hs-left { display: flex; flex-direction: column; align-items: center; min-width: 70px; }
    .hs-score { font-size: 1.8rem; font-weight: 900; line-height: 1; }
    .hs-max { font-size: 0.8rem; opacity: 0.5; }
    .hs-grade { font-size: 0.7rem; font-weight: 700; color: var(--text-dim); text-transform: uppercase; letter-spacing: 1px; margin-top: 2px; }
    .hs-mid { flex: 1; display: flex; flex-direction: column; gap: 5px; }
    .hs-bar { height: 6px; background: rgba(255,255,255,0.08); border-radius: 3px; overflow: hidden; }
    .hs-fill { height: 100%; border-radius: 3px; transition: width 0.5s ease; }
    .hs-summary { font-size: 0.75rem; color: var(--text-dim); }
    .hs-export { background: rgba(124,58,237,0.15); border: 1px solid rgba(124,58,237,0.4); color: #a78bfa; padding: 8px 14px; border-radius: 8px; font-size: 0.78rem; font-weight: 700; cursor: pointer; white-space: nowrap; transition: all 0.2s; }
    .hs-export:hover:not(:disabled) { background: rgba(124,58,237,0.3); }
    .hs-export:disabled { opacity: 0.5; cursor: default; }

    /* ── ALERT BAR ── */
    .alert-bar { display: flex; flex-wrap: wrap; gap: 8px; }
    .alert-item { background: rgba(245,158,11,0.12); border: 1px solid rgba(245,158,11,0.35); color: #f59e0b; padding: 7px 14px; border-radius: 8px; font-size: 0.8rem; font-weight: 700; }
    .alert-item.crit { background: rgba(239,68,68,0.12); border-color: rgba(239,68,68,0.35); color: #ef4444; }

    /* ── PROCESS TABLE ── */
    .proc-table { display: flex; flex-direction: column; gap: 2px; max-height: 420px; overflow-y: auto; }
    .proc-header { display: grid; grid-template-columns: 2fr 1fr 1fr 80px; padding: 6px 12px; font-size: 0.68rem; font-weight: 800; letter-spacing: 0.8px; color: var(--text-lo); text-transform: uppercase; border-bottom: 1px solid var(--border); }
    .proc-row { display: grid; grid-template-columns: 2fr 1fr 1fr 80px; align-items: center; padding: 6px 12px; border-radius: 6px; transition: background 0.1s; }
    .proc-row:hover { background: rgba(255,255,255,0.04); }
    .proc-name { font-size: 0.82rem; color: var(--text); font-weight: 500; }
    .proc-pid { font-size: 0.75rem; color: var(--text-dim); font-family: monospace; }
    .proc-mem { font-size: 0.82rem; font-weight: 700; font-family: monospace; }
    .proc-loading { padding: 24px; text-align: center; font-size: 0.85rem; color: var(--text-dim); }
    .btn-kill { background: rgba(239,68,68,0.1); border: 1px solid rgba(239,68,68,0.3); color: #ef4444; padding: 4px 10px; border-radius: 5px; font-size: 0.72rem; font-weight: 700; cursor: pointer; transition: all 0.15s; }
    .btn-kill:hover:not(:disabled) { background: rgba(239,68,68,0.25); }
    .btn-kill:disabled { opacity: 0.4; cursor: default; }

    /* ── NETWORK SPEED ── */
    .net-speed { display: flex; gap: 10px; font-family: monospace; font-size: 0.82rem; font-weight: 700; }
    .net-speed span { color: var(--text-dim); transition: color 0.3s; }
    .net-speed span.fast { color: #22D3EE; text-shadow: 0 0 8px rgba(34,211,238,0.4); }

    /* ── HEALTH SPARKLINE ── */
    .spark-wrap { display: flex; flex-direction: column; align-items: flex-end; gap: 2px; }
    .spark-svg { display: block; overflow: visible; }
    .spark-label { font-size: 0.6rem; color: var(--text-lo); text-align: right; }
    .hs-boottime { font-size: 0.68rem; color: var(--text-lo); font-family: monospace; margin-top: 2px; }

</style>
