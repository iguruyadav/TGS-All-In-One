<script>
    import { onMount, onDestroy, tick } from "svelte";
    import { pushLog } from "../stores/log.js";
    import {
        GetAudit, RunNativeAction, GetWindowsServiceStatus,
        InstallWindowsService, UninstallWindowsService,
        GetRDPStatus, GetBrowserStatus, GetUSBStatus,
        VerifyAdminPassword, SetAdminPassword, HasAdminPassword
    } from "../../wailsjs/go/main/App";

    let activeTab = "hardware";
    let auditData = null;
    let loading = true;
    let uploading = false;
    let scheduleUploading = false;
    let busy = {};
    let msg = "";
    let msgOk = true;
    let monitorStatus = "INACTIVE";
    let alertEmail = "";
    let smtpServer = "smtp.gmail.com";
    let smtpUser = "";
    let smtpPass = "";
    let smtpFromName = "TGS Security";

    // ── Hardware form fields ────────────────────────────────────────────────
    let assignedUser = "";
    let pcName = "";
    let network = "";
    let mac = "";
    let cpu = "";
    let mboard = "";
    let ramTotal = "";
    let slotsUsed = "";
    let slot1 = "";
    let slot2 = "";
    let st1 = "";
    let st2 = "";
    let st3 = "";
    let st4 = "";
    let ttCode = "TT0"; // default "TT0" — user can also type
    let d1Brand = "";
    let d1Model = "";
    let d1Size = "";
    let d1Code = "TGS-D-";
    let d2Brand = "";
    let d2Model = "";
    let d2Size = "";
    let d2Code = "TGS-D-";
    let mouseBrand = "";
    let mouseModel = "";
    let kbBrand = "";
    let kbModel = "";
    let headBrand = "";
    let headModel = "";
    let webcam = "";

    // ── Security state ──────────────────────────────────────────────────────
    let usbStatus = "ALLOWED";
    let rdpStatus = "ALLOWED";
    let schedStatus = "INACTIVE";
    let browserStatus = "ALLOWED";
    let usbPollInterval = null;  // real-time USB polling timer
    let domainInput = "*@outlook.com";
    let domainList = [
        "*@triveni-tech.in",
        "*@triveniglobalsoft.com",
        "*@supervedic.com",
    ];
    let selectedDomain = "";

    // ── Admin Password Modal state ──────────────────────────────────────────
    let pwdModalOpen   = false;   // whether the modal is visible
    let pwdInput       = "";      // user's typed password
    let pwdError       = "";      // error text shown in modal
    let pwdVerifying   = false;   // spinner while calling Go backend
    let pwdResolve     = null;    // resolve() of the pending Promise
    let pwdReject      = null;    // reject() of the pending Promise
    let pwdInputEl     = null;    // bind for auto-focus

    // Change-password sub-panel state
    let showChangePwd  = false;
    let cpCurrent      = "";
    let cpNew          = "";
    let cpConfirm      = "";
    let cpMsg          = "";
    let cpMsgOk        = true;
    let cpBusy         = false;

    // ── All dropdown options ─────────────────────────────────────────────────
    const ttSuggestions = Array.from(
        { length: 200 },
        (_, i) => `TT-${String(i + 1).padStart(3, "0")}`,
    );
    const dispBrands = [
        "Acer",
        "AOC",
        "BenQ",
        "Dell",
        "Lenovo",
        "LG",
        "Samsung",
        "Other",
    ];
    const dispSizes = [
        "18.5 inch",
        "19.5 inch",
        "20 inch",
        "21.5 inch",
        "22 inch",
        "24 inch",
        "27 inch",
    ];
    const mouseBrands = [
        "Logitech",
        "Dell",
        "HP",
        "Lenovo",
        "Fingers",
        "Other",
    ];
    const mouseByBrand = {
        Logitech: ["M90", "B100", "M171"],
        Dell: ["MS116", "B100"],
        HP: ["300"],
        Lenovo: ["SM8823"],
        Fingers: ["SH", "MOJUUO", "OOPH133"],
        Other: ["Standard"],
    };
    const kbBrands = ["Logitech", "Dell", "HP", "Lenovo", "Fingers", "Other"];
    const kbByBrand = {
        Logitech: ["K120", "K200", "MK200"],
        Dell: ["KB212", "KB216"],
        HP: ["PR1001U"],
        Lenovo: ["SK-8823"],
        Fingers: ["Standard"],
        Other: ["Standard"],
    };
    const headBrands = ["Logitech", "Lenovo", "JBL", "HP", "Sony", "Other"];
    const headModels = ["H110", "H111", "H390", "H340", "110", "Standard"];
    const webcamList = ["Logitech", "Lenovo", "HP", "Dell", "Inbuilt", "Other"];

    $: mouseSuggestions =
        mouseBrand && mouseByBrand[mouseBrand] ? mouseByBrand[mouseBrand] : [];
    $: kbSuggestions = kbBrand && kbByBrand[kbBrand] ? kbByBrand[kbBrand] : [];

    // ── Init ────────────────────────────────────────────────────────────────
    onMount(async () => {
        try {
            auditData = await GetAudit();
            pcName = auditData.System?.Name || "";
            mac = auditData.Network?.MAC || "";
            cpu = auditData.Hardware?.CPU || "";
            mboard = auditData.Hardware?.MB || "";
            const ip = auditData.Network?.IP || "";
            const ad = auditData.Network?.Adapter || "";
            network = ad ? `${ad} (${ip})` : ip;
            ramTotal = auditData.Hardware?.RAMTotal || "";
            slotsUsed = auditData.Hardware?.RAMSlots || "";
            const ram = auditData.Hardware?.RAMDetails || [];
            slot1 = ram[0] || "";
            slot2 = ram[1] || "";
            const stg = auditData.Hardware?.Storage || [];
            st1 = stg[0]?.Label || "";
            st2 = stg[1]?.Label || "";
            st3 = stg[2]?.Label || "";
            st4 = stg[3]?.Label || "";

            rdpStatus = auditData.Security?.RDPStatus || "ALLOWED";
            browserStatus = auditData.Security?.BrowserPolicy || "ALLOWED";
            usbStatus = auditData.Security?.USBStatus || "ALLOWED";
            schedStatus = auditData.Security?.ScheduleStatus || "INACTIVE";

            pushLog("✓ Audit scan complete", true);
        } catch (e) {
            pushLog("✗ " + e, false);
        } finally {
            loading = false;
        }

        // Load monitor status from real Windows Service
        try {
            monitorStatus = await GetWindowsServiceStatus();
            // Also try to load config for pre-filling email fields
            const r = await RunNativeAction("get-monitor-config", []);
            if (r) {
                const parts = r.split("|TGS|");
                if (parts.length > 1) {
                    const cfg = parts[1];
                    const cfgLines = cfg.split("\n");
                    for (let line of cfgLines) {
                        const m = line.match(/^([^=]+)=(.*)$/);
                        if (m) {
                            if (m[1] === "AlertEmail") alertEmail = m[2];
                            else if (m[1] === "SMTPServer") smtpServer = m[2];
                            else if (m[1] === "SMTPUser") smtpUser = m[2];
                            else if (m[1] === "SMTPPass") smtpPass = m[2];
                            else if (m[1] === "FromName") smtpFromName = m[2];
                        }
                    }
                }
            }
        } catch (e) {
            console.log("Monitor status load error:", e);
        }

        // ── Real-time Status polling every 4 seconds ──────────────
        usbPollInterval = setInterval(async () => {
            try {
                const uLive = await GetUSBStatus();
                if (uLive && (uLive === "BLOCKED" || uLive === "ALLOWED")) usbStatus = uLive;

                const rLive = await GetRDPStatus();
                if (rLive && (rLive === "BLOCKED" || rLive === "ALLOWED")) rdpStatus = rLive;

                const bLive = await GetBrowserStatus();
                if (bLive && (bLive === "BLOCKED" || bLive === "ALLOWED")) browserStatus = bLive;
            } catch (_) {}
        }, 4000);
    });

    onDestroy(() => {
        if (usbPollInterval) clearInterval(usbPollInterval);
    });


    async function rescan() {
        loading = true;
        auditData = null;
        try {
            auditData = await GetAudit();
            pushLog("✓ Re-scan done", true);
        } catch (e) {
            pushLog("✗ " + e, false);
        } finally {
            loading = false;
        }
    }

    // ── Admin Password Gate ──────────────────────────────────────────────────
    // Returns a Promise that resolves when the user types the correct password.
    // Rejects (silently) when they cancel.
    function requireAdminPassword(actionLabel = "this action") {
        return new Promise((resolve, reject) => {
            pwdInput    = "";
            pwdError    = "";
            pwdVerifying = false;
            pwdResolve  = resolve;
            pwdReject   = reject;
            pwdModalOpen = true;
            // Auto-focus the input after DOM updates
            tick().then(() => { if (pwdInputEl) pwdInputEl.focus(); });
        });
    }

    async function pwdSubmit() {
        if (!pwdInput.trim()) { pwdError = "Please enter the admin password."; return; }
        pwdVerifying = true;
        try {
            const ok = await VerifyAdminPassword(pwdInput.trim());
            if (ok) {
                pwdModalOpen = false;
                pwdResolve && pwdResolve(true);
            } else {
                pwdError = "❌ Incorrect password. Try again.";
                pwdInput = "";
                pushLog("[Security] ⚠ Wrong admin password attempt", false);
                tick().then(() => { if (pwdInputEl) pwdInputEl.focus(); });
            }
        } catch (e) {
            pwdError = "Error verifying password: " + (e?.message || e);
        }
        pwdVerifying = false;
    }

    function pwdCancel() {
        pwdModalOpen = false;
        pwdInput = "";
        pwdError = "";
        pwdReject && pwdReject("cancelled");
    }

    function pwdKeydown(e) {
        if (e.key === "Enter") pwdSubmit();
        if (e.key === "Escape") pwdCancel();
    }

    // ── Security actions ─────────────────────────────────────────────────────
    let toast = "";    // floating fixed toast text
    let toastOk = true;
    let toastTimer = null;

    function showToast(text, ok) {
        toast = text;
        toastOk = ok;
        msg = text; msgOk = ok;
        if (toastTimer) clearTimeout(toastTimer);
        toastTimer = setTimeout(() => { toast = ""; }, 6000);
    }

    // Actions that require admin password before executing
    const PROTECTED_ACTIONS = new Set([
        "allow-usb",
        "allow-rdp-clip",
        "reset-browser-policy",
    ]);

    async function secRun(key, label, stateKey, successVal, arg = "") {
        // Gate: protected actions require admin password
        if (PROTECTED_ACTIONS.has(key)) {
            try {
                await requireAdminPassword(label);
            } catch (_) {
                // User cancelled — do nothing
                return;
            }
        }

        busy = { ...busy, [key]: true };
        try {
            const result = await RunNativeAction(key, arg ? [arg] : []);
            if (stateKey === "usb") usbStatus = successVal;
            if (stateKey === "rdp") rdpStatus = successVal;
            if (stateKey === "brw") browserStatus = successVal;
            showToast(`✅ ${label}: ${result || 'Done'}`, true);
            pushLog("[Security] ✓ " + label + (result ? ': ' + result : ''), true);
            // Re-read real status from Go after a short delay
            setTimeout(async () => {
                try {
                    if (stateKey === "usb") usbStatus = await GetUSBStatus();
                    if (stateKey === "rdp") rdpStatus = await GetRDPStatus();
                    if (stateKey === "brw") browserStatus = await GetBrowserStatus();
                } catch (_) {}
            }, 1500);
        } catch (e) {
            showToast(`❌ ${label} failed: ${e?.message || e}`, false);
            pushLog("[Security] ✗ " + label + ': ' + (e?.message || e), false);
        }
        busy = { ...busy, [key]: false };
    }

    // ── Change Password ───────────────────────────────────────────────────────
    async function changePassword() {
        cpMsg = "";
        if (!cpNew.trim() || cpNew.length < 6) {
            cpMsg = "⚠ New password must be at least 6 characters."; cpMsgOk = false; return;
        }
        if (cpNew !== cpConfirm) {
            cpMsg = "⚠ New passwords do not match."; cpMsgOk = false; return;
        }
        cpBusy = true;
        try {
            const result = await SetAdminPassword(cpCurrent, cpNew);
            cpMsg = result; cpMsgOk = true;
            cpCurrent = ""; cpNew = ""; cpConfirm = "";
            pushLog("[Security] ✓ Admin password changed", true);
        } catch (e) {
            cpMsg = "❌ " + (e?.message || e); cpMsgOk = false;
        }
        cpBusy = false;
    }

    async function enableSchedule() {
        busy = { ...busy, sched: true };
        try {
            const r = await RunNativeAction("create-audit-schedule", []);
            schedStatus = "ACTIVE";
            msg = "✓ " + r;
            msgOk = true;
            pushLog("✓ Schedule enabled: " + r, true);
        } catch (e) {
            msg = "✗ " + e;
            msgOk = false;
            pushLog("✗ " + e, false);
        }
        busy = { ...busy, sched: false };
    }
    async function disableSchedule() {
        busy = { ...busy, sched: true };
        try {
            const r = await RunNativeAction("delete-audit-schedule", []);
            schedStatus = "INACTIVE";
            msg = "✓ " + r;
            msgOk = true;
            pushLog("✓ Schedule disabled", true);
        } catch (e) {
            msg = "✗ " + e;
            msgOk = false;
            pushLog("✗ " + e, false);
        }
        busy = { ...busy, sched: false };
    }


    // ── TGS WORLD Windows Service Monitor ───────────────────────────────────
    async function enableMonitor() {
        busy = { ...busy, monitor: true };
        try {
            const domains = domainList.join(",");
            const result = await InstallWindowsService(
                alertEmail, smtpServer, smtpUser, smtpPass, domains, smtpFromName
            );
            monitorStatus = "ACTIVE";
            showToast("✅ TGS WORLD Service: " + result, true);
            pushLog("[Security] ✓ TGS WORLD Windows Service installed & running", true);
        } catch (e) {
            showToast("❌ Service install failed: " + (e?.message || e), false);
            pushLog("[Security] ✗ Service install failed: " + (e?.message || e), false);
        }
        busy = { ...busy, monitor: false };
    }

    async function disableMonitor() {
        busy = { ...busy, monitor: true };
        try {
            const result = await UninstallWindowsService();
            monitorStatus = "INACTIVE";
            showToast("✅ TGS WORLD Service stopped: " + result, true);
            pushLog("[Security] ✓ TGS WORLD Windows Service removed", true);
        } catch (e) {
            showToast("❌ Service removal failed: " + (e?.message || e), false);
            pushLog("[Security] ✗ Service removal failed: " + (e?.message || e), false);
        }
        busy = { ...busy, monitor: false };
    }

    async function refreshServiceStatus() {
        try {
            monitorStatus = await GetWindowsServiceStatus();
        } catch (e) { monitorStatus = "INACTIVE"; }
    }

    function addDomain() {
        const d = domainInput.trim();
        if (d && !domainList.includes(d)) {
            domainList = [...domainList, d];
            domainInput = "";
        }
    }
    function removeSelectedDomain() {
        if (selectedDomain) {
            domainList = domainList.filter((d) => d !== selectedDomain);
            selectedDomain = "";
        } else if (domainList.length > 0) {
            domainList = domainList.slice(0, -1);
        }
    }

    // ── Upload ───────────────────────────────────────────────────────────────
    async function handleUpload() {
        if (!assignedUser.trim()) {
            msg = "⚠ Please enter Assigned To name.";
            msgOk = false;
            return;
        }
        uploading = true;
        msg = "";
        msgOk = true;
        const now = new Date();
        const p = (n) => String(n).padStart(2, "0");
        const dt = `${now.getFullYear()}-${p(now.getMonth() + 1)}-${p(now.getDate())} ${p(now.getHours())}:${p(now.getMinutes())}:${p(now.getSeconds())}`;
        const mbBrand = auditData?.Hardware?.MBBrand || "";
        const mbModel = auditData?.Hardware?.MBModel || "";
        const stg = auditData?.Hardware?.Storage || [];
        const sType = (i) => stg[i]?.Label?.split(" ")[0] || "";
        const sSize = (i) => {
            const l = stg[i]?.Label || "";
            const s = l.indexOf(" ");
            return s >= 0 ? l.substring(s + 1) : "";
        };
        const payload = {
            Targets: ["HardwareAudit", "SecurityCheck"],
            AssignedUser: assignedUser.trim(),
            PCName: pcName,
            IP: auditData?.Network?.IP || "N/A",
            MAC: mac,
            CPU: cpu,
            MBBrand: mbBrand,
            MBModel: mbModel,
            RAMType: "DDR4",
            RAM1: (auditData?.Hardware?.RAMDetails || [])[0] || "",
            RAM2: (auditData?.Hardware?.RAMDetails || [])[1] || "",
            RAMTotal: ramTotal,
            St1Type: sType(0),
            St1Size: sSize(0),
            St2Type: sType(1),
            St2Size: sSize(1),
            St3Type: sType(2),
            St3Size: sSize(2),
            St4Type: sType(3),
            St4Size: sSize(3),
            TTCode: ttCode,
            MouseBrand: mouseBrand,
            MouseModel: mouseModel,
            KeyBrand: kbBrand,
            KeyModel: kbModel,
            HeadBrand: headBrand,
            HeadModel: headModel,
            Webcam: webcam,
            Disp1Brand: d1Brand,
            Disp1Model: d1Model,
            Disp1Size: d1Size,
            Disp1Code: d1Code,
            Disp2Brand: d2Brand,
            Disp2Model: d2Model,
            Disp2Size: d2Size,
            Disp2Code: d2Code,
            USBStatus: usbStatus,
            RDPStatus: rdpStatus,
            BrowserDomain: browserStatus,
            ScheduleStatus: schedStatus,
        };
        try {
            await UploadAuditData(payload);
            msg = "✓ Audit data saved to Google Sheet!";
            msgOk = true;
            pushLog("✓ Audit uploaded", true);
        } catch (e) {
            msg = "✗ Upload failed: " + e;
            msgOk = false;
            pushLog("[Audit] ✗ Upload failed: " + e, false);
        } finally {
            uploading = false;
        }
    }

    async function handleTestSchedule() {
        if (!assignedUser.trim()) {
            msg = "⚠ Please enter Assigned To name.";
            msgOk = false;
            return;
        }
        scheduleUploading = true;
        msg = "";
        msgOk = true;

        const payload = {
            Targets: ["SCHEDULE Audit"],
            AssignedUser: assignedUser.trim(),
            PCName: pcName,
            IP: auditData?.Network?.IP || "N/A",
            USBStatus: usbStatus,
            RDPStatus: rdpStatus,
            BrowserDomain: browserStatus,
            ScheduleStatus: schedStatus,
            TTCode: "Scheduled Success",
        };

        try {
            await UploadAuditData(payload);
            msg = "✓ Test Schedule Audit saved to Google Sheet!";
            msgOk = true;
            pushLog("✓ Test Schedule Audit uploaded", true);
        } catch (e) {
            msg = "✗ Upload failed: " + e;
            msgOk = false;
        } finally {
            scheduleUploading = false;
        }
    }

    // ── COPY TO CLIPBOARD ─────────────────────────────────────────────────────
    let copiedField = "";
    function copyField(key, value) {
        if (!value) return;
        navigator.clipboard.writeText(value).then(() => {
            copiedField = key;
            setTimeout(() => { copiedField = ""; }, 1800);
        }).catch(() => {});
    }
</script>

<!-- ════════════════════════════════════════════════════════════════ -->
<div class="page">
    <!-- Header -->
    <div class="page-header">
        <div>
            <h1 class="page-title">📋 Audit &amp; Security</h1>
            <p class="page-desc">
                Auto-scan PC specs · Tag assets · Apply security policies ·
                Upload to sheet
            </p>
        </div>
        <button class="btn btn-ghost" on:click={rescan} disabled={loading}>
            {loading ? "⟳ Scanning…" : "🔄 Re-Scan"}
        </button>
    </div>

    {#if msg}
        <div class="banner" class:ok={msgOk} class:err={!msgOk}>
            <span>{msg}</span>
            <button class="xb" on:click={() => (msg = "")}>✕</button>
        </div>
    {/if}

    <!-- Fixed floating toast removed — now using inline bp-result panel -->


    <!-- Tabs -->
    <div class="tabbar">
        <button
            class="tabbt"
            class:on={activeTab === "hardware"}
            on:click={() => (activeTab = "hardware")}
        >
            🔧 Hardware Audit
        </button>
        <button
            class="tabbt"
            class:on={activeTab === "security"}
            on:click={() => (activeTab = "security")}
        >
            🔒 Security Check
        </button>
    </div>

    <!-- ════════ HARDWARE AUDIT ════════════════════════════════════════ -->
    {#if activeTab === "hardware"}
        {#if loading}
            <div class="scan-loading card">
                <div class="pulse-ring"></div>
                <p>Scanning system hardware &amp; peripherals…</p>
            </div>
        {:else}
            <div class="tabcontent">
                <!-- ROW 1: System Info + Memory/Storage -->
                <div class="hw-grid">
                    <div class="card hw-card">
                        <div class="g-label">🖥️ System Information</div>
                        <div class="field-row">
                            <label class="flbl"
                                >Assigned To <span class="req">*</span></label
                            >
                            <input
                                class="input"
                                bind:value={assignedUser}
                                placeholder="Employee name…"
                            />
                        </div>
                        <div class="field-row">
                            <label class="flbl">PC Name</label>
                            <div class="copy-row">
                                <input class="input ro" value={pcName} readonly />
                                <button class="copy-btn" on:click={() => copyField('pc', pcName)} title="Copy PC Name">
                                    {copiedField === 'pc' ? '✓' : '📋'}
                                </button>
                            </div>
                        </div>
                        <div class="field-row">
                            <label class="flbl">Network</label>
                            <div class="copy-row">
                                <input class="input ro" value={network} readonly />
                                <button class="copy-btn" on:click={() => copyField('net', network)} title="Copy IP/Network">
                                    {copiedField === 'net' ? '✓' : '📋'}
                                </button>
                            </div>
                        </div>
                        <div class="field-row">
                            <label class="flbl">MAC Address</label>
                            <div class="copy-row">
                                <input class="input ro" value={mac} readonly />
                                <button class="copy-btn" on:click={() => copyField('mac', mac)} title="Copy MAC">
                                    {copiedField === 'mac' ? '✓' : '📋'}
                                </button>
                            </div>
                        </div>
                        <div class="field-row">
                            <label class="flbl">CPU</label>
                            <input class="input ro" value={cpu} readonly />
                        </div>
                        <div class="field-row">
                            <label class="flbl">Motherboard</label>
                            <input class="input ro" value={mboard} readonly />
                        </div>
                        <div class="divider"></div>
                        <div class="g-label" style="margin-top:2px">
                            🏷️ Asset Tagging
                        </div>
                        <!-- TT Code: type freely or pick from list -->
                        <div class="field-row">
                            <label class="flbl">TT Code</label>
                            <input
                                class="input"
                                list="tt-list"
                                bind:value={ttCode}
                                placeholder="TT0"
                            />
                            <datalist id="tt-list">
                                {#each ttSuggestions as t}<option value={t}
                                        >{t}</option
                                    >{/each}
                            </datalist>
                        </div>
                    </div>

                    <div class="card hw-card">
                        <div class="g-label">🧠 Memory &amp; Storage</div>
                        <div class="field-row">
                            <label class="flbl">Total RAM</label><input
                                class="input ro"
                                value={ramTotal}
                                readonly
                            />
                        </div>
                        <div class="field-row">
                            <label class="flbl">Slots Used</label><input
                                class="input ro"
                                value={slotsUsed}
                                readonly
                            />
                        </div>
                        <div class="field-row">
                            <label class="flbl">Slot 1</label><input
                                class="input ro"
                                value={slot1}
                                readonly
                            />
                        </div>
                        <div class="field-row">
                            <label class="flbl">Slot 2</label><input
                                class="input ro"
                                value={slot2}
                                readonly
                            />
                        </div>
                        <div class="divider"></div>
                        <div class="field-row">
                            <label class="flbl">Storage 1</label><input
                                class="input ro"
                                value={st1}
                                readonly
                            />
                        </div>
                        <div class="field-row">
                            <label class="flbl">Storage 2</label><input
                                class="input ro"
                                value={st2}
                                readonly
                            />
                        </div>
                        <div class="field-row">
                            <label class="flbl">Storage 3</label><input
                                class="input ro"
                                value={st3}
                                readonly
                            />
                        </div>
                        <div class="field-row">
                            <label class="flbl">Storage 4</label><input
                                class="input ro"
                                value={st4}
                                readonly
                            />
                        </div>
                    </div>
                </div>

                <!-- ROW 2: Peripherals + Display 1 + Display 2 -->
                <div class="hw-grid hw-grid-3">
                    <!-- Peripherals — auto-detected + editable -->
                    <div class="card hw-card">
                        <div class="g-label">🖱️ Peripherals</div>

                        <!-- Mouse Brand — editable combobox -->
                        <div class="field-row">
                            <label class="flbl">Mouse Brand</label>
                            <input
                                class="input"
                                list="mouse-brands"
                                bind:value={mouseBrand}
                                placeholder="Type or select…"
                            />
                            <datalist id="mouse-brands">
                                {#each mouseBrands as b}<option value={b}
                                        >{b}</option
                                    >{/each}
                            </datalist>
                        </div>
                        <div class="field-row">
                            <label class="flbl">Mouse Model</label>
                            <input
                                class="input"
                                list="mouse-models"
                                bind:value={mouseModel}
                                placeholder="Type or select…"
                            />
                            <datalist id="mouse-models">
                                {#each mouseSuggestions as m}<option value={m}
                                        >{m}</option
                                    >{/each}
                            </datalist>
                        </div>
                        <div class="divider"></div>
                        <div class="field-row">
                            <label class="flbl">KB Brand</label>
                            <input
                                class="input"
                                list="kb-brands"
                                bind:value={kbBrand}
                                placeholder="Type or select…"
                            />
                            <datalist id="kb-brands">
                                {#each kbBrands as b}<option value={b}
                                        >{b}</option
                                    >{/each}
                            </datalist>
                        </div>
                        <div class="field-row">
                            <label class="flbl">KB Model</label>
                            <input
                                class="input"
                                list="kb-models"
                                bind:value={kbModel}
                                placeholder="Type or select…"
                            />
                            <datalist id="kb-models">
                                {#each kbSuggestions as m}<option value={m}
                                        >{m}</option
                                    >{/each}
                            </datalist>
                        </div>
                        <div class="divider"></div>
                        <div class="field-row">
                            <label class="flbl">Headset Brand</label>
                            <input
                                class="input"
                                list="head-brands"
                                bind:value={headBrand}
                                placeholder="Type or select…"
                            />
                            <datalist id="head-brands">
                                {#each headBrands as b}<option value={b}
                                        >{b}</option
                                    >{/each}
                            </datalist>
                        </div>
                        <div class="field-row">
                            <label class="flbl">Headset Model</label>
                            <input
                                class="input"
                                list="head-models"
                                bind:value={headModel}
                                placeholder="Type or select…"
                            />
                            <datalist id="head-models">
                                {#each headModels as m}<option value={m}
                                        >{m}</option
                                    >{/each}
                            </datalist>
                        </div>
                        <div class="field-row">
                            <label class="flbl">Webcam</label>
                            <input
                                class="input"
                                list="webcam-list"
                                bind:value={webcam}
                                placeholder="Type or select…"
                            />
                            <datalist id="webcam-list">
                                {#each webcamList as w}<option value={w}
                                        >{w}</option
                                    >{/each}
                            </datalist>
                        </div>
                    </div>

                    <!-- Display 1 — auto-detected, fully editable -->
                    <div class="card hw-card">
                        <div class="g-label">🖥️ Display 1</div>
                        <div class="field-row">
                            <label class="flbl">Brand</label>
                            <input
                                class="input"
                                list="d1-brands"
                                bind:value={d1Brand}
                                placeholder="Type or select…"
                            />
                            <datalist id="d1-brands">
                                {#each dispBrands as b}<option value={b}
                                        >{b}</option
                                    >{/each}
                            </datalist>
                        </div>
                        <div class="field-row">
                            <label class="flbl">Model</label>
                            <input
                                class="input"
                                bind:value={d1Model}
                                placeholder="e.g. E1912"
                            />
                        </div>
                        <div class="field-row">
                            <label class="flbl">Size</label>
                            <input
                                class="input"
                                list="d1-sizes"
                                bind:value={d1Size}
                                placeholder="e.g. 21.5 inch"
                            />
                            <datalist id="d1-sizes">
                                {#each dispSizes as s}<option value={s}
                                        >{s}</option
                                    >{/each}
                            </datalist>
                        </div>
                        <div class="field-row">
                            <label class="flbl">Unique Code</label>
                            <input
                                class="input"
                                bind:value={d1Code}
                                placeholder="TGS-D-"
                            />
                        </div>
                    </div>

                    <!-- Display 2 -->
                    <div class="card hw-card">
                        <div class="g-label">
                            🖥️ Display 2 <span class="opt-tag">Optional</span>
                        </div>
                        <div class="field-row">
                            <label class="flbl">Brand</label>
                            <input
                                class="input"
                                list="d2-brands"
                                bind:value={d2Brand}
                                placeholder="Type or select…"
                            />
                            <datalist id="d2-brands">
                                {#each dispBrands as b}<option value={b}
                                        >{b}</option
                                    >{/each}
                            </datalist>
                        </div>
                        <div class="field-row">
                            <label class="flbl">Model</label>
                            <input
                                class="input"
                                bind:value={d2Model}
                                placeholder="e.g. E1912"
                            />
                        </div>
                        <div class="field-row">
                            <label class="flbl">Size</label>
                            <input
                                class="input"
                                list="d2-sizes"
                                bind:value={d2Size}
                                placeholder="e.g. 21.5 inch"
                            />
                            <datalist id="d2-sizes">
                                {#each dispSizes as s}<option value={s}
                                        >{s}</option
                                    >{/each}
                            </datalist>
                        </div>
                        <div class="field-row">
                            <label class="flbl">Unique Code</label>
                            <input
                                class="input"
                                bind:value={d2Code}
                                placeholder="TGS-D-"
                            />
                        </div>
                    </div>
                </div>

                <!-- Upload -->
                <div style="display: flex; gap: 12px; margin-top: 10px;">
                    <button
                        class="upload-btn btn btn-primary"
                        style="flex: 1;"
                        on:click={handleUpload}
                        disabled={uploading || scheduleUploading}
                    >
                        {uploading
                            ? "⟳ Saving to Sheet…"
                            : "💾 Upload Audit Data"}
                    </button>
                    <button
                        class="upload-btn btn btn-ghost"
                        style="flex: 1;"
                        on:click={handleTestSchedule}
                        disabled={uploading || scheduleUploading}
                    >
                        {scheduleUploading
                            ? "⟳ Testing Schedule…"
                            : "⏰ Test Schedule Upload"}
                    </button>
                </div>
            </div>
        {/if}

        <!-- ════════ SECURITY CHECK ════════════════════════════════════════ -->
    {:else}
        <div class="tabcontent">
            <!-- USB -->
            <div class="card sec-card">
                <div class="sec-row-header">
                    <div>
                        <div class="g-label">🔌 USB STORAGE BLOCK</div>
                        <div
                            class="sec-status"
                            class:s-ok={usbStatus === "BLOCKED"}
                            class:s-warn={usbStatus !== "BLOCKED"}
                        >
                            {#if busy["block-usb"] || busy["allow-usb"]}
                                <span class="usb-spinner">⏳</span> Processing...
                            {:else}
                                Status: <strong>{usbStatus}</strong>
                            {/if}
                        </div>
                        <div class="sched-info">Blocks all USB mass storage (pendrives, external HDDs) via USBSTOR service.</div>
                    </div>
                    <div class="btn-cluster">
                        <button
                            class="btn btn-danger"
                            on:click={() =>
                                secRun(
                                    "block-usb",
                                    "USB Blocked",
                                    "usb",
                                    "BLOCKED",
                                )}
                            disabled={busy["block-usb"] || busy["allow-usb"]}>
                            {busy["block-usb"] ? "⏳ BLOCKING..." : "🔒 BLOCK USB"}
                        </button>
                        <button
                            class="btn-allow"
                            on:click={() =>
                                secRun(
                                    "allow-usb",
                                    "USB Allowed",
                                    "usb",
                                    "ALLOWED",
                                )}
                            disabled={busy["allow-usb"]}>
                            {busy["allow-usb"] ? "⏳ ALLOWING..." : "✅ ALLOW USB"}
                        </button>
                    </div>
                </div>
            </div>

            <!-- RDP -->
            <div class="card sec-card">
                <div class="sec-row-header">
                    <div>
                        <div class="g-label">🖥️ RDP CLIPBOARD SHARE</div>
                        <div
                            class="sec-status"
                            class:s-ok={rdpStatus === "BLOCKED"}
                            class:s-warn={rdpStatus !== "BLOCKED"}
                        >
                            {#if busy["block-rdp-clip"] || busy["allow-rdp-clip"]}
                                <span class="usb-spinner">⏳</span> Processing...
                            {:else}
                                Status: <strong>{rdpStatus}</strong>
                            {/if}
                        </div>
                    </div>
                    <div class="btn-cluster">
                        <button
                            class="btn btn-danger"
                            on:click={() =>
                                secRun(
                                    "block-rdp-clip",
                                    "RDP Blocked",
                                    "rdp",
                                    "BLOCKED",
                                )}
                            disabled={busy["block-rdp-clip"] || busy["allow-rdp-clip"]}>
                            {busy["block-rdp-clip"] ? "⏳ BLOCKING..." : "BLOCK RDP CLIP"}
                        </button>
                        <button
                            class="btn-allow"
                            on:click={() =>
                                secRun(
                                    "allow-rdp-clip",
                                    "RDP Allowed",
                                    "rdp",
                                    "ALLOWED",
                                )}
                            disabled={busy["allow-rdp-clip"]}>ALLOW RDP</button
                        >
                    </div>
                </div>
            </div>

            <!-- Schedule - Coming Soon -->
            <div class="card sec-card coming-soon-card">
                <div class="sec-row-header">
                    <div>
                        <div class="g-label" style="display:flex;align-items:center;gap:10px;">
                            ⏰ AUTO-SECURITY SCHEDULE
                            <span class="coming-soon-badge">🚧 Coming Soon</span>
                        </div>
                        <div class="sec-status s-warn" style="opacity:0.5;">Status: DISABLED</div>
                        <div class="sched-info" style="opacity:0.5;">This feature is under development</div>
                    </div>
                    <div class="btn-cluster">
                        <button class="btn btn-ghost" disabled style="opacity:0.4;cursor:not-allowed;">COMING SOON</button>
                    </div>
                </div>
                <div class="sched-hint warn" style="opacity:0.6;">
                    🚧 Auto-Security Schedule will be available in a future update.
                </div>
            </div>

            <!-- Browser Policy -->
            <div class="card sec-card">
                <div class="g-label">🌐 BROWSER POLICY</div>
                <div
                    class="sec-status"
                    class:s-ok={browserStatus === "BLOCKED"}
                    class:s-warn={browserStatus !== "BLOCKED"}
                >
                    {#if busy["apply-browser-policy"] || busy["reset-browser-policy"]}
                        <span class="usb-spinner">⏳</span> Processing...
                    {:else}
                        Status: <strong>{browserStatus}</strong>
                    {/if}
                </div>
                <div class="divider"></div>
                <div class="domain-controls">
                    <div class="domain-input-row">
                        <input
                            class="input domain-in"
                            bind:value={domainInput}
                            placeholder="*@domain.com"
                            on:keydown={(e) => e.key === "Enter" && addDomain()}
                        />
                        <button class="btn-enable" on:click={addDomain}
                            >+ ADD</button
                        >
                        <button
                            class="btn btn-danger"
                            on:click={removeSelectedDomain}>− REMOVE</button
                        >
                    </div>
                    <div class="domain-split">
                        <!-- Clickable list so user can select and remove a specific domain -->
                        <div class="domain-listbox">
                            {#each domainList as d}
                                <div
                                    class="domain-item"
                                    class:selected={selectedDomain === d}
                                    on:click={() =>
                                        (selectedDomain =
                                            selectedDomain === d ? "" : d)}
                                    on:keydown={() => {}}
                                >
                                    {d}
                                </div>
                            {/each}
                            {#if domainList.length === 0}
                                <div class="domain-empty">
                                    No domains added yet
                                </div>
                            {/if}
                        </div>
                        <button
                            class="btn-neutral save-list-btn"
                            on:click={() => {
                                msg = "✓ Domain list saved";
                                msgOk = true;
                                pushLog("Domain list saved", true);
                            }}
                        >
                            SAVE LIST
                        </button>
                    </div>
                </div>
                <div class="browser-btns">
                    <button
                        class="btn btn-primary apply-btn"
                        disabled={busy["apply-browser-policy"]}
                        on:click={() =>
                            secRun(
                                "apply-browser-policy",
                                "Browser Policy Applied",
                                "brw",
                                "BLOCKED",
                                domainList.join(","),
                            )}
                    >
                        {busy["apply-browser-policy"] ? "⏳ Applying…" : "✅ APPLY FIX"}
                    </button>
                    <button
                        class="btn btn-danger reset-btn"
                        disabled={busy["reset-browser-policy"]}
                        on:click={() => {
                            secRun(
                                "reset-browser-policy",
                                "Browser Policy Reset",
                                "brw",
                                "ALLOWED",
                            );
                        }}
                    >
                        {busy["reset-browser-policy"] ? "⏳ Resetting…" : "↩ RESET RULES"}
                    </button>
                </div>

                <!-- Inline result panel — appears below buttons after action -->
                {#if toast}
                    <div class="bp-result" class:bp-ok={toastOk} class:bp-err={!toastOk}>
                        <div class="bp-result-icon">{toastOk ? "✅" : "❌"}</div>
                        <div class="bp-result-text">
                            <strong>{toastOk ? "Success" : "Error"}</strong>
                            <span>{toast.replace(/^[✅❌]\s*/, "")}</span>
                        </div>
                        <button class="bp-close" on:click={() => toast = ""}>✕</button>
                    </div>
                {/if}
            </div>
        </div>

        <!-- Change Admin Password Card -->
        <div class="card sec-card">
            <div class="sec-row-header">
                <div>
                    <div class="g-label" style="display:flex;align-items:center;gap:8px;">
                        🔐 ADMIN PASSWORD PROTECTION
                        <span class="pwd-badge">⚠️ Protected</span>
                    </div>
                    <div class="sec-status" style="color:var(--accent-text);font-size:0.78rem;margin-top:4px;">
                        Allow/Reset actions require admin password. Default: <code style="background:rgba(0,0,0,0.3);padding:2px 6px;border-radius:4px;">TGS@Admin</code>
                    </div>
                </div>
                <button class="btn-neutral" on:click={() => showChangePwd = !showChangePwd}>
                    {showChangePwd ? '▲ Hide' : '🔑 Change Password'}
                </button>
            </div>

            {#if showChangePwd}
                <div class="chpwd-form">
                    <div class="divider" style="margin:14px 0 12px"></div>
                    <div class="chpwd-grid">
                        <div class="field-row">
                            <label class="flbl">Current Pwd</label>
                            <input class="input" type="password" bind:value={cpCurrent} placeholder="Current password…" />
                        </div>
                        <div class="field-row">
                            <label class="flbl">New Pwd</label>
                            <input class="input" type="password" bind:value={cpNew} placeholder="Min. 6 characters…" />
                        </div>
                        <div class="field-row">
                            <label class="flbl">Confirm</label>
                            <input class="input" type="password" bind:value={cpConfirm} placeholder="Repeat new password…" />
                        </div>
                    </div>
                    {#if cpMsg}
                        <div class="chpwd-msg" class:chpwd-ok={cpMsgOk} class:chpwd-err={!cpMsgOk}>{cpMsg}</div>
                    {/if}
                    <button class="btn btn-primary" style="margin-top:10px;" on:click={changePassword} disabled={cpBusy}>
                        {cpBusy ? '⏳ Saving…' : '💾 Save New Password'}
                    </button>
                </div>
            {/if}
        </div>

        <!-- Login Alert Monitor Card - Coming Soon -->
        <div class="card sec-card monitor-card coming-soon-card">
            <div class="g-label" style="display:flex;align-items:center;gap:10px;">
                🔔 Login Alert Monitor
                <span class="coming-soon-badge">🚧 Coming Soon</span>
            </div>
            <div class="sec-status s-warn" style="opacity:0.5;">Monitor: DISABLED</div>
            <div class="divider"></div>
            <p class="monitor-desc" style="opacity:0.5;">
                Detects when any browser attempts to sign in with a blocked domain
                and sends a Windows notification + email alert.
            </p>
            <div class="monitor-btns" style="opacity:0.4;pointer-events:none;">
                <button class="btn btn-primary apply-btn" disabled>🔔 ENABLE MONITOR (Service)</button>
                <button class="btn btn-danger reset-btn" disabled>🔕 DISABLE MONITOR</button>
            </div>
            <div class="sched-hint warn" style="opacity:0.6;">
                🚧 Login Alert Monitor will be available in a future update.
            </div>
        </div>

{/if}
</div>

<!-- ════════════════ ADMIN PASSWORD MODAL ════════════════════ -->
{#if pwdModalOpen}
<div
    class="pwd-overlay"
    on:click|self={pwdCancel}
    on:keydown={pwdKeydown}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
>
    <div class="pwd-modal">
        <!-- Header -->
        <div class="pwd-modal-header">
            <div class="pwd-lock-icon">🔒</div>
            <div>
                <div class="pwd-modal-title">Admin Password Required</div>
                <div class="pwd-modal-sub">This action is protected. Enter the admin password to continue.</div>
            </div>
        </div>

        <!-- Input -->
        <div class="pwd-input-wrap">
            <input
                class="pwd-input"
                type="password"
                bind:this={pwdInputEl}
                bind:value={pwdInput}
                placeholder="Enter admin password…"
                autocomplete="off"
                disabled={pwdVerifying}
                on:keydown={pwdKeydown}
            />
            {#if pwdError}
                <div class="pwd-error">{pwdError}</div>
            {/if}
        </div>

        <!-- Buttons -->
        <div class="pwd-btns">
            <button class="pwd-cancel-btn" on:click={pwdCancel} disabled={pwdVerifying}>
                Cancel
            </button>
            <button class="pwd-confirm-btn" on:click={pwdSubmit} disabled={pwdVerifying}>
                {#if pwdVerifying}
                    <span class="pwd-spinner-ring"></span> Verifying…
                {:else}
                    🔓 Unlock
                {/if}
            </button>
        </div>

        <div class="pwd-hint">🔑 Default password: <strong>TGS@Admin</strong> — change it in the Admin Password card.</div>
    </div>
</div>
{/if}


<!-- ════════════════════════════════════════════════════════════════ -->
<style>
    /* ── Tab bar ───────────────────────────────────────────────────────── */
    .tabbar {
        display: flex;
        gap: 4px;
        margin-bottom: 20px;
        background: rgba(0, 0, 0, 0.25);
        border: 1px solid var(--border);
        border-radius: var(--radius);
        padding: 4px;
    }
    .tabbt {
        flex: 1;
        padding: 10px 20px;
        background: transparent;
        border: none;
        border-radius: calc(var(--radius) - 2px);
        color: var(--text-dim);
        font-family: var(--font);
        font-size: 0.85rem;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
    }
    .tabbt:hover {
        background: var(--bg-hover);
        color: var(--text);
    }
    .tabbt.on {
        background: var(--bg-active);
        color: var(--text);
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
        border: 1px solid var(--border-md);
    }

    .tabcontent {
        display: flex;
        flex-direction: column;
        gap: 16px;
    }

    /* ── Grids ─────────────────────────────────────────────────────────── */
    .hw-grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 14px;
    }
    .hw-grid-3 {
        display: grid;
        grid-template-columns: 1fr 1fr 1fr;
        gap: 14px;
    }
    .hw-card {
        display: flex;
        flex-direction: column;
        gap: 10px;
        padding: 16px 18px;
    }

    /* ── Field row ─────────────────────────────────────────────────────── */
    .field-row {
        display: flex;
        align-items: center;
        gap: 0.8vw;
    }
    .flbl {
        flex: 0 0 90px;
        font-size: clamp(0.65rem, 0.8vw, 0.78rem);
        color: var(--text-dim);
        white-space: nowrap;
    }
    .field-row input {
        flex: 1;
        min-width: 0;
        font-size: clamp(0.75rem, 1vw, 0.85rem);
        padding: clamp(6px, 1vh, 10px) 12px;
    }

    .req {
        color: var(--warn);
    }

    /* input with datalist still looks native but matches theme */
    .input.ro {
        color: var(--text-dim);
        cursor: default;
    }

    /* ── Labels, divider ───────────────────────────────────────────────── */
    .g-label {
        font-size: 0.72rem;
        font-weight: 700;
        color: var(--text-dim);
        text-transform: uppercase;
        letter-spacing: 0.8px;
        margin-bottom: 4px;
    }
    .divider {
        height: 1px;
        background: var(--border);
        margin: 6px 0;
    }

    /* ── Auto badge ────────────────────────────────────────────────────── */
    .auto-badge {
        display: inline-block;
        font-size: 0.62rem;
        padding: 2px 7px;
        border-radius: 10px;
        background: rgba(76, 175, 114, 0.12);
        color: var(--success);
        border: 1px solid rgba(76, 175, 114, 0.25);
        margin-left: 6px;
        font-weight: 600;
        text-transform: none;
        letter-spacing: 0;
        vertical-align: middle;
    }

    /* ── Coming Soon badge & card ───────────────────────────────────────── */
    .coming-soon-badge {
        display: inline-flex;
        align-items: center;
        font-size: 0.65rem;
        font-weight: 700;
        padding: 3px 10px;
        border-radius: 20px;
        background: rgba(255, 180, 0, 0.15);
        color: #ffb400;
        border: 1px solid rgba(255, 180, 0, 0.35);
        letter-spacing: 0.04em;
        text-transform: uppercase;
    }
    .coming-soon-card {
        opacity: 0.65;
        pointer-events: none;
        position: relative;
        user-select: none;
        filter: grayscale(30%);
    }
    .opt-tag {
        display: inline-block;
        font-size: 0.62rem;
        padding: 2px 7px;
        border-radius: 10px;
        background: var(--bg-hover);
        color: var(--text-lo);
        border: 1px solid var(--border);
        margin-left: 6px;
        font-weight: 400;
        text-transform: none;
        letter-spacing: 0;
        vertical-align: middle;
    }

    /* ── Scan loading ──────────────────────────────────────────────────── */
    .scan-loading {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 20px;
        padding: 60px;
        color: var(--text-dim);
        font-size: 0.9rem;
    }
    .pulse-ring {
        width: 40px;
        height: 40px;
        border-radius: 50%;
        border: 3px solid rgba(74, 143, 219, 0.25);
        border-top-color: var(--accent);
        animation: spin 1s linear infinite;
    }
    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }

    /* ── Banner ────────────────────────────────────────────────────────── */
    .banner {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 10px 16px;
        border-radius: var(--radius);
        font-size: 0.84rem;
        margin-bottom: 4px;
    }
    .banner.ok {
        background: var(--success-dim);
        border: 1px solid rgba(76, 175, 114, 0.3);
        color: var(--success);
    }
    .banner.err {
        background: var(--danger-dim);
        border: 1px solid rgba(224, 85, 85, 0.3);
        color: var(--danger);
    }
    .xb {
        background: none;
        border: none;
        color: inherit;
        cursor: pointer;
        opacity: 0.7;
    }
    .xb:hover {
        opacity: 1;
    }

    /* ── Browser Policy inline result panel ── */
    .bp-result {
        display: flex;
        align-items: flex-start;
        gap: 12px;
        margin-top: 12px;
        padding: 12px 16px;
        border-radius: 8px;
        animation: bpSlideDown 0.25s ease;
        width: 100%;
    }
    @keyframes bpSlideDown {
        from { opacity: 0; transform: translateY(-8px); }
        to   { opacity: 1; transform: translateY(0); }
    }
    .bp-ok  { background: rgba(16,185,129,0.12); border: 1px solid rgba(16,185,129,0.4); }
    .bp-err { background: rgba(239,68,68,0.12);  border: 1px solid rgba(239,68,68,0.4);  }
    .bp-result-icon { font-size: 1.3rem; line-height: 1; flex-shrink: 0; }
    .bp-result-text {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 2px;
    }
    .bp-ok  .bp-result-text strong { color: #10b981; font-size: 0.82rem; letter-spacing: 0.5px; text-transform: uppercase; }
    .bp-err .bp-result-text strong { color: #ef4444; font-size: 0.82rem; letter-spacing: 0.5px; text-transform: uppercase; }
    .bp-result-text span { font-size: 0.8rem; opacity: 0.8; line-height: 1.4; }
    .bp-close {
        background: none; border: none; cursor: pointer;
        opacity: 0.5; font-size: 0.9rem; padding: 0 4px;
        color: inherit; flex-shrink: 0;
    }
    .bp-close:hover { opacity: 1; }

    /* ── Upload button ─────────────────────────────────────────────────── */
    .upload-btn {
        width: 100%;
        padding: 15px;
        font-size: 0.95rem;
        letter-spacing: 0.5px;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 10px;
        box-shadow: 0 4px 20px rgba(74, 143, 219, 0.25);
        margin-bottom: 8px;
    }
    .upload-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    /* ── Security cards ────────────────────────────────────────────────── */
    .sec-card {
        padding: 18px 22px;
    }
    .sec-row-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 20px;
        flex-wrap: wrap;
    }
    .sec-status {
        font-size: 0.85rem;
        font-weight: 700;
        margin-top: 4px;
    }
    .s-ok {
        color: var(--success);
    }
    .s-warn {
        color: var(--danger);
    }

    .btn-cluster {
        display: flex;
        gap: 10px;
        flex-wrap: wrap;
        align-items: center;
    }

    .btn-allow {
        background: rgba(76, 175, 114, 0.12);
        border: 1px solid rgba(76, 175, 114, 0.4);
        color: var(--success);
        font-size: 0.8rem;
        font-family: var(--font);
        font-weight: 700;
        padding: 8px 18px;
        border-radius: var(--radius);
        cursor: pointer;
        transition: all 0.2s;
    }
    .btn-allow:hover {
        background: rgba(76, 175, 114, 0.25);
    }
    .btn-allow:disabled {
        opacity: 0.4;
        cursor: not-allowed;
    }
    .btn-enable {
        background: rgba(46, 139, 87, 0.18);
        border: 1px solid rgba(46, 139, 87, 0.5);
        color: #5dbb8a;
        font-size: 0.8rem;
        font-family: var(--font);
        font-weight: 700;
        padding: 8px 18px;
        border-radius: var(--radius);
        cursor: pointer;
        transition: all 0.2s;
    }
    .btn-enable:hover {
        background: rgba(46, 139, 87, 0.3);
    }
    .btn-enable:disabled {
        opacity: 0.4;
        cursor: not-allowed;
    }
    .btn-neutral {
        background: rgba(255, 255, 255, 0.06);
        border: 1px solid var(--border-md);
        color: var(--text-dim);
        font-size: 0.8rem;
        font-family: var(--font);
        font-weight: 600;
        padding: 8px 18px;
        border-radius: var(--radius);
        cursor: pointer;
        transition: all 0.2s;
    }
    .btn-neutral:hover {
        background: var(--bg-hover);
        color: var(--text);
    }

    /* ── Schedule info ─────────────────────────────────────────────────── */
    .sched-info {
        font-size: 0.78rem;
        color: var(--text-dim);
        margin-top: 4px;
    }
    .sched-hint {
        margin-top: 12px;
        padding: 10px 14px;
        background: rgba(76, 175, 114, 0.07);
        border: 1px solid rgba(76, 175, 114, 0.2);
        border-radius: var(--radius);
        font-size: 0.8rem;
        color: var(--text-dim);
        line-height: 1.5;
    }
    .sched-hint.warn {
        background: rgba(212, 145, 58, 0.08);
        border-color: rgba(212, 145, 58, 0.25);
        color: var(--warn);
    }

    /* ── Browser Policy ────────────────────────────────────────────────── */
    .domain-controls {
        display: flex;
        flex-direction: column;
        gap: 10px;
        margin-top: 8px;
    }
    .domain-input-row {
        display: flex;
        gap: 8px;
        align-items: center;
        flex-wrap: wrap;
    }
    .domain-in {
        flex: 1;
        max-width: 380px;
    }

    .domain-split {
        display: flex;
        gap: 14px;
        align-items: flex-start;
    }
    .domain-listbox {
        flex: 1;
        max-width: 420px;
        min-height: 110px;
        max-height: 150px;
        background: rgba(0, 0, 0, 0.3);
        border: 1px solid var(--border-md);
        border-radius: var(--radius);
        overflow-y: auto;
        padding: 4px;
    }
    .domain-item {
        padding: 6px 10px;
        cursor: pointer;
        border-radius: 4px;
        font-size: 0.82rem;
        font-family: "Consolas", monospace;
        color: var(--text-dim);
        transition: background 0.15s;
    }
    .domain-item:hover {
        background: var(--bg-hover);
        color: var(--text);
    }
    .domain-item.selected {
        background: var(--accent-dim);
        color: var(--accent-text);
    }
    .domain-empty {
        padding: 10px;
        color: var(--text-lo);
        font-size: 0.8rem;
        text-align: center;
    }

    .save-list-btn {
        white-space: nowrap;
        margin-top: 4px;
        align-self: flex-start;
    }

    .browser-btns {
        display: flex;
        gap: 12px;
        margin-top: 14px;
    }
    .apply-btn {
        flex: 1;
        padding: 13px;
        font-size: 0.9rem;
        justify-content: center;
        font-weight: 700;
    }
    .reset-btn {
        flex: 1;
        padding: 12px;
        font-size: 0.9rem;
        justify-content: center;
        font-weight: 700;
    }

    /* Copy buttons */
    .copy-row {
        display: flex; gap: 6px; align-items: center; width: 100%;
    }
    .copy-row .input { flex: 1; }
    .copy-btn {
        flex-shrink: 0; width: 34px; height: 34px;
        background: rgba(124,58,237,0.08); border: 1px solid rgba(124,58,237,0.3);
        color: var(--accent-text); border-radius: var(--radius-sm);
        cursor: pointer; font-size: 0.82rem; transition: all 0.15s;
        display: flex; align-items: center; justify-content: center;
    }
    .copy-btn:hover { background: rgba(124,58,237,0.2); }

    /* admin-pwd-badge */
    .pwd-badge {
        display:inline-flex;align-items:center;font-size:0.62rem;font-weight:700;
        padding:3px 10px;border-radius:20px;background:rgba(245,158,11,0.15);
        color:var(--warn);border:1px solid rgba(245,158,11,0.4);text-transform:uppercase;
    }
    .chpwd-form{display:flex;flex-direction:column;gap:8px;}
    .chpwd-grid{display:flex;flex-direction:column;gap:8px;}
    .chpwd-msg{padding:8px 14px;border-radius:var(--radius-sm);font-size:0.82rem;margin-top:4px;}
    .chpwd-ok{background:var(--success-dim);color:var(--success);border:1px solid rgba(16,185,129,0.3);}
    .chpwd-err{background:var(--danger-dim);color:var(--danger);border:1px solid rgba(239,68,68,0.3);}
    .pwd-overlay{position:fixed;inset:0;z-index:9999;background:rgba(0,0,0,0.72);backdrop-filter:blur(8px);display:flex;align-items:center;justify-content:center;animation:pwdFadeIn 0.18s ease;}
    @keyframes pwdFadeIn{from{opacity:0;}to{opacity:1;}}
    .pwd-modal{background:rgba(20,24,42,0.98);border:1px solid var(--border-hi);border-radius:var(--radius-xl);box-shadow:0 30px 80px rgba(0,0,0,0.7),inset 0 1px 0 rgba(255,255,255,0.06);padding:32px 36px 28px;width:420px;max-width:90vw;display:flex;flex-direction:column;gap:20px;animation:pwdSlideUp 0.22s cubic-bezier(0.34,1.56,0.64,1);}
    @keyframes pwdSlideUp{from{opacity:0;transform:translateY(24px) scale(0.96);}to{opacity:1;transform:translateY(0) scale(1);}}
    .pwd-modal-header{display:flex;align-items:center;gap:16px;}
    .pwd-lock-icon{font-size:2.2rem;width:52px;height:52px;flex-shrink:0;background:linear-gradient(135deg,rgba(124,58,237,0.25),rgba(34,211,238,0.1));border:1px solid var(--border-hi);border-radius:14px;display:flex;align-items:center;justify-content:center;animation:lockPulse 2s infinite;}
    @keyframes lockPulse{0%,100%{box-shadow:0 0 0 0 rgba(124,58,237,0.3);}50%{box-shadow:0 0 0 8px rgba(124,58,237,0);}}
    .pwd-modal-title{font-size:1.05rem;font-weight:700;color:var(--text);}
    .pwd-modal-sub{font-size:0.78rem;color:var(--text-dim);margin-top:4px;line-height:1.4;}
    .pwd-input-wrap{display:flex;flex-direction:column;gap:8px;}
    .pwd-input{width:100%;background:rgba(0,0,0,0.35);border:1.5px solid var(--border-md);border-radius:var(--radius);color:var(--text);font-size:1rem;padding:14px 16px;outline:none;transition:border-color 0.2s,box-shadow 0.2s;box-sizing:border-box;}
    .pwd-input:focus{border-color:var(--accent);box-shadow:0 0 0 3px rgba(124,58,237,0.2);}
    .pwd-input:disabled{opacity:0.5;cursor:not-allowed;}
    .pwd-error{font-size:0.8rem;color:var(--danger);padding:6px 10px;background:var(--danger-dim);border:1px solid rgba(239,68,68,0.3);border-radius:var(--radius-sm);animation:pwdErrShake 0.35s ease;}
    @keyframes pwdErrShake{0%,100%{transform:translateX(0);}20%{transform:translateX(-6px);}40%{transform:translateX(6px);}60%{transform:translateX(-4px);}80%{transform:translateX(4px);}}
    .pwd-btns{display:flex;gap:10px;}
    .pwd-cancel-btn{flex:1;padding:12px;background:rgba(255,255,255,0.05);border:1px solid var(--border-md);border-radius:var(--radius);color:var(--text-dim);font-family:var(--font);font-size:0.88rem;font-weight:600;cursor:pointer;transition:all 0.2s;}
    .pwd-cancel-btn:hover{background:rgba(255,255,255,0.1);color:var(--text);}
    .pwd-cancel-btn:disabled{opacity:0.4;cursor:not-allowed;}
    .pwd-confirm-btn{flex:2;padding:12px;background:var(--grad);border:none;border-radius:var(--radius);color:#fff;font-family:var(--font);font-size:0.9rem;font-weight:700;cursor:pointer;letter-spacing:0.04em;transition:all 0.2s;display:flex;align-items:center;justify-content:center;gap:8px;box-shadow:0 4px 20px rgba(124,58,237,0.4);}
    .pwd-confirm-btn:hover{opacity:0.9;transform:translateY(-1px);}
    .pwd-confirm-btn:disabled{opacity:0.5;cursor:not-allowed;transform:none;}
    .pwd-spinner-ring{display:inline-block;width:16px;height:16px;border:2px solid rgba(255,255,255,0.3);border-top-color:#fff;border-radius:50%;animation:spin 0.7s linear infinite;flex-shrink:0;}
    .pwd-hint{font-size:0.72rem;color:var(--text-lo);text-align:center;line-height:1.4;}
    .pwd-hint strong{color:var(--text-dim);}
</style>
