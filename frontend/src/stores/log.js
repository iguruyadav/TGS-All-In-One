import { writable } from 'svelte/store';

/** @type {import('svelte/store').Writable<Array<{msg: string, ok: boolean, time: string}>>} */
export const logStore = writable([]);

/**
 * Push a message to the global log panel (appends at bottom, newest last).
 * @param {string} msg
 * @param {boolean} ok
 */
export function pushLog(msg, ok = true) {
    const time = new Date().toLocaleTimeString('en-IN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    logStore.update(logs => {
        const next = [...logs, { msg, ok, time }];
        return next.slice(-100); // keep last 100 entries
    });
}

/**
 * Clear all messages from the global log panel.
 */
export function clearLogs() {
    logStore.set([]);
}
/**
 * Export current logs as a downloadable text file.
 * @param {Array<{msg: string, ok: boolean, time: string}>} logs
 */
export function exportLogs(logs) {
    const lines = logs.map(e => `[${e.time}] ${e.ok ? 'OK' : 'ERR'} ${e.msg}`).join('\n');
    const blob = new Blob([lines], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `tgs_logs_${new Date().toISOString().slice(0, 10)}.txt`;
    a.click();
    URL.revokeObjectURL(url);
}
