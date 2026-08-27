<script>
    import { onMount } from "svelte";
    import {
        GetSetupStatus,
        RunNativeAction,
        ApplyNetworkConfig,
        SetDefaultWallpaper,
    } from "../../wailsjs/go/main/App";
    import { pushLog } from "../stores/log.js";

    // ── Internal tabs ─────────────────────────────────
    let activeTab = "setup"; // "setup" | "network"

    // ── State ─────────────────────────────────────────
    let statusData = {
        PCName: "---",
        RDPStatus: "Checking…",
        Server121: "Disconnected",
        NASStorage: "Disconnected",
        TimeZone: "---",
    };
    let newPCName = "";
    let nasUser = "";
    let nasPass = "";
    let showNasDlg = false;
    let loading = false;
    let feedback = { msg: "", ok: true };
    let toastVisible = false;
    let toastTimer = null;

    // Network state
    let netAdapter = "Ethernet 0";
    let netAdapters = ["Ethernet 0", "Wi-Fi"];
    let fullIP = "174.156.5.";
    let netMask = "255.255.252.0";
    let netGw = "174.156.5.11";
    let netDns = "8.8.8.8, 8.8.4.4";

    onMount(loadStatus);

    async function loadStatus() {
        try {
            const d = await GetSetupStatus();
            statusData = { ...statusData, ...d };
            newPCName = statusData.PCName;
        } catch (e) {
            console.error(e);
        }
    }

    function setFeedback(msg, ok = true) {
        feedback = { msg, ok };
        // Show a prominent toast notification
        toastVisible = true;
        if (toastTimer) clearTimeout(toastTimer);
        if (ok) {
            toastTimer = setTimeout(() => {
                if (feedback.msg === msg) { feedback.msg = ""; toastVisible = false; }
            }, 6000); // stay 6 seconds
        }
    }

    async function act(action, args = []) {
        loading = true;
        setFeedback(`Running: ${action}…`);
        pushLog(`▶ ${action}…`);
        try {
            let res = await RunNativeAction(action, args);
            await loadStatus();
            setFeedback(`Done ✓`, true);
            if (res) {
                pushLog(`✓ ${action} completed: ${res}`, true);
            } else {
                pushLog(`✓ ${action} completed`, true);
            }
        } catch (e) {
            setFeedback(`Error: ${e}`, false);
            pushLog(`✗ ${action} failed: ${e}`, false);
        } finally {
            loading = false;
        }
    }

    function renamePc() {
        act("set-pc-name", [newPCName]);
    }
    function setTimezone() {
        act("set-timezone", []);
    }
    function enableRDP() {
        act("enable-rdp", []);
    }
    function disableRDP() {
        act("disable-rdp", []);
    }
    function addThisPc() {
        act("add-this-pc-icon", []);
    }
    async function setDefaultWall() {
        loading = true;
        setFeedback("Setting Triveni wallpaper...");
        pushLog(`▶ Setting Triveni default wallpaper…`);
        try {
            const res = await SetDefaultWallpaper();
            setFeedback(res, true);
            pushLog(`✓ Triveni wallpaper applied`, true);
        } catch (e) {
            setFeedback("Error: " + e, false);
            pushLog(`✗ Wallpaper error: ${e}`, false);
        } finally {
            loading = false;
        }
    }
    function addCreds121() {
        act("add-creds-z", []);
    }
    function map121() {
        act("map-z-drive", []);
    }
    function unmap121() {
        act("unmap-z-drive", []);
    }
    function unmapNAS() {
        act("unmap-y-drive", []);
    }
    function upgradeWin() {
        act("upgrade-windows", []);
    }
    function reboot() {
        if (confirm('Restart this PC now?\n\nAll unsaved work will be lost.')) {
            act("reboot", []);
        }
    }
    function activateWindows() {
        act("activate-windows", []);
    }

    function addCredsNAS() {
        if (!nasUser || !nasPass) {
            showNasDlg = true;
            return;
        }
        act("add-creds-y", [nasUser, nasPass]);
        showNasDlg = false;
        nasUser = "";
        nasPass = "";
    }
    function mapNAS() {
        act("map-y-drive", []);
    }

    async function CreateSchedule() {
        if (!fullIP || fullIP === "174.156.5.") {
            setFeedback("Enter full IP address (e.g. 174.156.5.12)", false);
            return;
        }
        loading = true;
        setFeedback("Applying network settings…");
        pushLog(`▶ Applying network: ${netAdapter} → ${fullIP}`);
        try {
            const res = await ApplyNetworkConfig(
                netAdapter,
                fullIP,
                netMask,
                netGw,
                netDns,
            );
            setFeedback(res + " (Reboot recommended)", true);
            pushLog(`✓ Network config applied on ${netAdapter}`, true);
        } catch (e) {
            setFeedback(`Error: ${e}`, false);
            pushLog(`✗ Network error: ${e}`, false);
        } finally {
            loading = false;
        }
    }
</script>

<!-- ═══════════════════════════════════════════════════════════ -->
<div class="page">
    <!-- ── Page header ───────────────────────────────────────── -->
    <div class="pg-header">
        <div>
            <h2 class="pg-title">⚙️ Setup &amp; Configuration</h2>
            <p class="pg-desc">
                System settings, network drives, and Windows upgrade
            </p>
        </div>
        {#if feedback.msg}
            <div
                class="fb"
                class:fb-ok={feedback.ok}
                class:fb-err={!feedback.ok}
            >
                {feedback.msg}
            </div>
        {/if}
    </div>

    <!-- ── Tab bar ───────────────────────────────────────────── -->
    <div class="tab-bar">
        <button
            class="tab-btn"
            class:active={activeTab === "setup"}
            on:click={() => (activeTab = "setup")}
        >
            🖥️ Setup
        </button>
        <button
            class="tab-btn"
            class:active={activeTab === "network"}
            on:click={() => (activeTab = "network")}
        >
            🌐 Network
        </button>
    </div>

    <!-- ════════════ SETUP TAB ═══════════════════════════════ -->
    {#if activeTab === "setup"}
        <div class="setup-grid">
            <!-- SYSTEM CONFIG ──────────────────────────────────── -->
            <div class="card">
                <div class="card-t">🖥️ System Configuration</div>

                <div class="setting-row">
                    <div class="sr-info">
                        <span class="sr-label">PC Name</span>
                        <span class="sr-sub"
                            >Current: <strong>{statusData.PCName}</strong></span
                        >
                    </div>
                    <div class="sr-ctrl gap">
                        <input
                            class="input sm"
                            bind:value={newPCName}
                            placeholder="New name"
                        />
                        <button
                            class="btn-act"
                            on:click={renamePc}
                            disabled={loading ||
                                newPCName === statusData.PCName ||
                                !newPCName}
                        >
                            Save &amp; Reboot
                        </button>
                    </div>
                </div>

                <div class="setting-row">
                    <div class="sr-info">
                        <span class="sr-label">Timezone</span>
                        <span class="sr-sub"
                            >{statusData.TimeZone || "---"}</span
                        >
                    </div>
                    <button
                        class="btn-act"
                        on:click={setTimezone}
                        disabled={loading}
                    >
                        Set IST (UTC+5:30)
                    </button>
                </div>

                <div class="setting-row">
                    <div class="sr-info">
                        <span class="sr-label">Remote Desktop (RDP)</span>
                        <span
                            class="sr-sub"
                            class:status-on={statusData.RDPStatus === "Enabled"}
                            class:status-off={statusData.RDPStatus ===
                                "Disabled"}
                        >
                            {statusData.RDPStatus}
                        </span>
                    </div>
                    <div class="sr-ctrl gap">
                        <button
                            class="btn-act ok"
                            on:click={enableRDP}
                            disabled={loading ||
                                statusData.RDPStatus === "Enabled"}
                            >Enable</button
                        >
                        <button
                            class="btn-act err"
                            on:click={disableRDP}
                            disabled={loading ||
                                statusData.RDPStatus === "Disabled"}
                            >Disable</button
                        >
                    </div>
                </div>

                <div class="setting-row">
                    <div class="sr-info">
                        <span class="sr-label">This PC Icon</span>
                        <span class="sr-sub">Add shortcut to Desktop</span>
                    </div>
                    <button
                        class="btn-act"
                        on:click={addThisPc}
                        disabled={loading}>Add Icon</button
                    >
                </div>

                <div class="setting-row">
                    <div class="sr-info">
                        <span class="sr-label">Triveni Wallpaper</span>
                        <span class="sr-sub"
                            >Apply default or custom wallpaper</span
                        >
                    </div>
                    <div class="sr-ctrl gap">
                        <button
                            class="btn-act act"
                            on:click={setDefaultWall}
                            disabled={loading}
                        >
                            🌅 Apply Triveni Wallpaper
                        </button>
                    </div>
                </div>
            </div>

            <!-- NETWORK DRIVES ─────────────────────────────────── -->
            <div class="card">
                <div class="card-t">🗄️ Network Drives</div>

                <div class="drive-row">
                    <div class="drive-info">
                        <span class="drive-letter">Z:</span>
                        <div>
                            <div class="sr-label">Server 121</div>
                            <div class="sr-sub">\\174.156.5.121</div>
                        </div>
                        <span
                            class="badge {statusData.Server121 === 'Connected'
                                ? 'badge-on'
                                : 'badge-off'}"
                        >
                            {statusData.Server121}
                        </span>
                    </div>
                    <div class="sr-ctrl gap">
                        <button
                            class="btn-act act"
                            on:click={addCreds121}
                            disabled={loading}>Add</button
                        >
                        <button
                            class="btn-act ok"
                            on:click={map121}
                            disabled={loading}>Map</button
                        >
                        <button
                            class="btn-act err"
                            on:click={unmap121}
                            disabled={loading}>Disconnect</button
                        >
                    </div>
                </div>

                <div class="drive-row">
                    <div class="drive-info">
                        <span class="drive-letter">Y:</span>
                        <div>
                            <div class="sr-label">NAS Storage</div>
                            <div class="sr-sub">\\174.156.4.3</div>
                        </div>
                        <span
                            class="badge {statusData.NASStorage === 'Connected'
                                ? 'badge-on'
                                : 'badge-off'}"
                        >
                            {statusData.NASStorage}
                        </span>
                    </div>
                    <div class="sr-ctrl gap">
                        <button
                            class="btn-act act"
                            on:click={() => (showNasDlg = true)}
                            disabled={loading}>Add</button
                        >
                        <button
                            class="btn-act ok"
                            on:click={mapNAS}
                            disabled={loading}>Map</button
                        >
                        <button
                            class="btn-act err"
                            on:click={unmapNAS}
                            disabled={loading}>Disconnect</button
                        >
                    </div>
                </div>

                {#if showNasDlg}
                    <div
                        class="dlg-overlay"
                        on:click|self={() => (showNasDlg = false)}
                    >
                        <div class="dlg">
                            <div class="dlg-t">
                                NAS Credentials — \\174.156.4.3
                            </div>
                            <label class="dlg-label">Username</label>
                            <input
                                class="input"
                                bind:value={nasUser}
                                placeholder="username"
                            />
                            <label class="dlg-label">Password</label>
                            <input
                                class="input"
                                type="password"
                                bind:value={nasPass}
                                placeholder="password"
                            />
                            <div class="dlg-actions">
                                <button
                                    class="btn-act ok"
                                    on:click={addCredsNAS}
                                    disabled={!nasUser || !nasPass}
                                    >Save Credentials</button
                                >
                                <button
                                    class="btn-act ghost"
                                    on:click={() => (showNasDlg = false)}
                                    >Cancel</button
                                >
                            </div>
                        </div>
                    </div>
                {/if}
            </div>

            <!-- WINDOWS UPGRADE (full-width) ───────────────────── -->
            <div class="card card-wide">
                <div class="card-t">🪟 Windows Upgrade — Home → Pro</div>
                <div class="upgrade-row">
                    <div>
                        <p class="upg-desc">
                            Apply the product key below to upgrade <strong
                                >Windows 10 Home → Pro</strong
                            >. Activation requires internet.
                        </p>
                        <div class="key-box">VK7JG-NPHTM-C97JM-9MPGT-3V66T</div>
                    </div>
                    <div class="sr-ctrl gap">
                        <button
                            class="btn-act act"
                            on:click={upgradeWin}
                            disabled={loading}
                        >
                            🔑 Apply Key &amp; Activate
                        </button>
                        <button
                            class="btn-act err"
                            on:click={reboot}
                            disabled={loading}
                        >
                            🔄 Restart PC
                        </button>
                    </div>
                </div>
            </div>

            <!-- SYSTEM ACTIONS card (full-width) -->

            <div class="card card-wide actions-card">
                <div class="card-t">⚡ System Actions</div>
                <div class="upgrade-row">
                    <div>
                        <p class="upg-desc">Quick system actions — reboot the PC or activate Windows license.</p>
                    </div>
                    <div class="sr-ctrl gap">
                        <button class="btn-act err" on:click={reboot} disabled={loading}>
                            🔄 Restart PC
                        </button>
                        <button class="btn-act act" on:click={activateWindows} disabled={loading}>
                            🪟 Activate Windows
                        </button>
                    </div>
                </div>
            </div>
        </div>
    {/if}

    <!-- ════════════ NETWORK TAB ═════════════════════════════ -->
    {#if activeTab === "network"}
        <div class="card net-card">
            <div class="card-t">🌐 Static IP Configuration</div>
            <p class="upg-desc">
                Configure static IP — <strong>Ethernet 0</strong> or
                <strong>Wi-Fi</strong> only
            </p>

            <div class="net-form">
                <div class="net-field">
                    <label class="net-label">Adapter</label>
                    <select class="input" bind:value={netAdapter}>
                        {#each netAdapters as a}<option value={a}>{a}</option
                            >{/each}
                    </select>
                </div>

                <div class="net-field">
                    <label class="net-label">IP Address</label>
                    <input
                        class="input"
                        bind:value={fullIP}
                        placeholder="174.156.5.___"
                    />
                </div>

                <div class="net-field">
                    <label class="net-label">Subnet Mask</label>
                    <input class="input" bind:value={netMask} />
                </div>

                <div class="net-field">
                    <label class="net-label">Default Gateway</label>
                    <input class="input" bind:value={netGw} />
                </div>

                <div class="net-field">
                    <label class="net-label">DNS Servers</label>
                    <input class="input" bind:value={netDns} />
                </div>
            </div>

            <button
                class="btn-apply"
                on:click={applyNet}
                disabled={loading || fullIP === "174.156.5."}
            >
                {loading ? "Applying..." : "⚡ Apply Network Settings"}
            </button>
        </div>
    {/if}

    <!-- TOAST NOTIFICATION -->
    {#if toastVisible && feedback.msg}
    <div class="setup-toast" class:toast-ok={feedback.ok} class:toast-err={!feedback.ok}>
        <span class="toast-icon">{feedback.ok ? '✅' : '❌'}</span>
        <span class="toast-msg">{feedback.msg}</span>
        <button class="toast-close" on:click={() => { toastVisible = false; feedback.msg = ''; }}>✕</button>
    </div>
    {/if}
</div>

<style>
    /* ── Page ─────────────────────────── */
    .pg-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 16px;
        flex-wrap: wrap;
        gap: 12px;
    }
    .pg-title {
        margin: 0 0 3px;
        font-size: 1.1rem;
        font-weight: 700;
        color: var(--text);
    }
    .pg-desc {
        margin: 0;
        font-size: 0.8rem;
        color: var(--text-dim);
    }

    /* ── Tab bar ──────────────────────── */
    .tab-bar {
        display: flex;
        gap: 4px;
        margin-bottom: 18px;
        border-bottom: 1px solid var(--border);
        padding-bottom: 0;
    }
    .tab-btn {
        font-family: var(--font);
        font-size: 0.82rem;
        font-weight: 600;
        padding: 8px 20px;
        background: transparent;
        border: none;
        border-bottom: 2px solid transparent;
        color: var(--text-dim);
        cursor: pointer;
        transition:
            color 0.15s,
            border-color 0.15s;
        margin-bottom: -1px;
    }
    .tab-btn:hover {
        color: var(--text);
    }
    .tab-btn.active {
        color: var(--accent-text);
        border-bottom-color: var(--accent);
    }

    /* ── Feedback ─────────────────────── */
    .fb {
        padding: 8px 14px;
        border-radius: var(--radius);
        font-size: 0.8rem;
        border: 1px solid;
    }
    .fb-ok {
        color: var(--success);
        border-color: rgba(76, 175, 114, 0.35);
        background: var(--success-dim);
    }
    .fb-err {
        color: var(--danger);
        border-color: rgba(224, 85, 85, 0.35);
        background: var(--danger-dim);
    }

    /* ── Grid ─────────────────────────── */
    .setup-grid {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 14px;
    }
    .card-wide {
        grid-column: span 2;
    }

    /* ── Card ─────────────────────────── */
    .card {
        background: rgba(30, 30, 30, 0.25); /* More transparent */
        border: 1px solid rgba(255, 255, 255, 0.08); /* Soft glass border */
        border-radius: var(--radius);
        padding: 22px;
        display: flex;
        flex-direction: column;
        gap: 16px;
        backdrop-filter: blur(30px); /* Deep frosted glass effect */
        -webkit-backdrop-filter: blur(30px);
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3); /* Softer, broader shadow */
    }
    .card-t {
        font-size: 0.75rem;
        font-weight: 800;
        text-transform: uppercase;
        letter-spacing: 0.8px;
        color: var(--text-dim);
        padding-bottom: 12px;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    }

    /* ── Setting rows ─────────────────── */
    .setting-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 12px;
    }
    .sr-info {
        display: flex;
        flex-direction: column;
        gap: 2px;
        min-width: 0;
    }
    .sr-label {
        font-size: 0.86rem;
        font-weight: 600;
        color: var(--text);
    }
    .sr-sub {
        font-size: 0.73rem;
        color: var(--text-dim);
    }
    .status-on {
        color: var(--success) !important;
        font-weight: 600;
    }
    .status-off {
        color: var(--danger);
        font-weight: 600;
    }
    .sr-ctrl {
        display: flex;
        align-items: center;
        flex-shrink: 0;
    }
    .gap {
        gap: 8px;
    }

    /* ── Buttons ──────────────────────── */
    .btn-act {
        font-family: var(--font);
        font-size: 0.78rem;
        font-weight: 600;
        padding: 7px 16px;
        border-radius: var(--radius-sm);
        background: var(--bg-hover);
        border: 1px solid var(--border-md);
        color: var(--text-dim);
        cursor: pointer;
        white-space: nowrap;
        transition: all 0.13s;
    }
    .btn-act:hover:not(:disabled) {
        background: var(--bg-active);
        border-color: var(--border-hi);
        color: var(--text);
    }
    .btn-act:disabled {
        opacity: 0.35;
        cursor: not-allowed;
    }
    .btn-act.ok {
        color: var(--success);
        border-color: rgba(76, 175, 114, 0.4);
    }
    .btn-act.ok:hover:not(:disabled) {
        background: var(--success-dim);
    }
    .btn-act.err {
        color: var(--danger);
        border-color: rgba(224, 85, 85, 0.4);
    }
    .btn-act.err:hover:not(:disabled) {
        background: var(--danger-dim);
    }
    .btn-act.act {
        background: var(--accent-dim);
        border-color: var(--accent);
        color: var(--accent-text);
    }
    .btn-act.ghost {
        background: transparent;
        border: 1px solid var(--border-md);
        color: var(--text-dim);
    }

    /* ── Inputs ───────────────────────── */
    .input {
        background: var(--bg);
        border: 1px solid var(--border-md);
        border-radius: var(--radius-sm);
        color: var(--text);
        font-family: var(--font);
        font-size: 0.83rem;
        padding: 7px 10px;
        outline: none;
        transition: border-color 0.15s;
        width: 100%;
    }
    .input:focus {
        border-color: var(--accent);
    }
    .input.sm {
        width: 160px;
    }

    /* ── Drive rows ───────────────────── */
    .drive-row {
        display: flex;
        flex-direction: column;
        gap: 8px;
        padding: 10px 12px;
        background: var(--bg);
        border: 1px solid var(--border);
        border-radius: var(--radius-sm);
    }
    .drive-info {
        display: flex;
        align-items: center;
        gap: 12px;
    }
    .drive-letter {
        font-size: 1.4rem;
        font-weight: 800;
        color: var(--accent-text);
        min-width: 32px;
    }
    .badge {
        margin-left: auto;
        font-size: 0.68rem;
        font-weight: 700;
        padding: 3px 10px;
        border-radius: 20px;
        border: 1px solid;
    }
    .badge-on {
        color: var(--success);
        border-color: rgba(76, 175, 114, 0.4);
        background: var(--success-dim);
    }
    .badge-off {
        color: var(--text-dim);
        border-color: var(--border-md);
        background: var(--bg-hover);
    }

    /* ── NAS dialog ───────────────────── */
    .dlg-overlay {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.45);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 9000;
        backdrop-filter: blur(8px);
    }
    .dlg {
        background: rgba(20, 20, 20, 0.45);
        border: 1px solid rgba(255, 255, 255, 0.15);
        border-radius: var(--radius-lg);
        padding: 28px;
        width: 320px;
        display: flex;
        flex-direction: column;
        gap: 12px;
        backdrop-filter: blur(40px);
        -webkit-backdrop-filter: blur(40px);
        box-shadow: 0 12px 40px rgba(0, 0, 0, 0.4);
    }
    .dlg-t {
        font-size: 1.05rem;
        font-weight: 700;
        color: var(--text);
        margin-bottom: 6px;
    }
    .dlg-label {
        font-size: 0.75rem;
        color: var(--text-dim);
    }
    .dlg-actions {
        display: flex;
        gap: 10px;
        margin-top: 10px;
    }

    /* ── Upgrade ──────────────────────── */
    .upgrade-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 20px;
        flex-wrap: wrap;
    }
    .upg-desc {
        margin: 0 0 10px;
        font-size: 0.82rem;
        color: var(--text-dim);
    }
    .key-box {
        background: var(--bg);
        border: 1px solid var(--border-md);
        border-radius: var(--radius-sm);
        padding: 10px 14px;
        font-family: "Consolas", monospace;
        font-size: 0.9rem;
        font-weight: 700;
        color: var(--accent-text);
        letter-spacing: 1px;
        text-align: center;
    }

    /* ── Network card & form ──────────── */
    .net-card {
        max-width: 700px;
    }
    .net-form {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 14px;
        margin-bottom: 8px;
    }
    .net-field {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }
    .net-label {
        font-size: 0.73rem;
        font-weight: 600;
        color: var(--text-dim);
    }
    .btn-apply {
        font-family: var(--font);
        font-size: 0.85rem;
        font-weight: 700;
        padding: 12px 30px;
        border-radius: var(--radius);
        background: var(--accent);
        border: 1px solid rgba(255, 255, 255, 0.2);
        color: white;
        cursor: pointer;
        transition: all 0.2s;
        align-self: flex-start;
        box-shadow: 0 4px 12px rgba(74, 143, 219, 0.3);
    }
    .btn-apply:hover:not(:disabled) {
        transform: translateY(-1px);
        box-shadow: 0 6px 16px rgba(74, 143, 219, 0.4);
    }
    .btn-apply:disabled {
        opacity: 0.35;
        cursor: not-allowed;
    }

    /* ── TOAST NOTIFICATION ─────────────────────────────── */
    .setup-toast {
        position: fixed;
        bottom: 24px;
        right: 24px;
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 12px 18px;
        border-radius: 12px;
        border: 1px solid rgba(255,255,255,.15);
        backdrop-filter: blur(12px);
        -webkit-backdrop-filter: blur(12px);
        font-size: 0.85rem;
        font-weight: 600;
        max-width: 420px;
        z-index: 9999;
        animation: toast-in 0.3s ease;
        box-shadow: 0 8px 32px rgba(0,0,0,.4);
    }
    @keyframes toast-in {
        from { opacity: 0; transform: translateY(16px); }
        to   { opacity: 1; transform: translateY(0); }
    }
    .toast-ok  { background: rgba(16,185,129,.2); border-color: rgba(16,185,129,.4); color: #6ee7b7; }
    .toast-err { background: rgba(239,68,68,.2);  border-color: rgba(239,68,68,.4);  color: #fca5a5; }
    .toast-icon { font-size: 1rem; flex-shrink: 0; }
    .toast-msg  { flex: 1; line-height: 1.4; }
    .toast-close {
        background: transparent; border: none; color: inherit; cursor: pointer;
        font-size: 0.9rem; padding: 0 4px; opacity: 0.7; flex-shrink: 0;
    }
    .toast-close:hover { opacity: 1; }
    .actions-card { margin-top: 8px; }
</style>

