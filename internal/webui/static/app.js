// Wave Dashboard - Main Application JS

// --- HTML Escaping (shared utility) ---
function escapeHTML(s) {
    return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

// --- Toast Notification System ---
function showToast(message, type, duration) {
    type = type || 'error';
    duration = duration || 5000;
    var container = document.getElementById('toast-container');
    if (!container) return;
    var toast = document.createElement('div');
    toast.className = 'toast toast-' + type;
    toast.textContent = message;
    toast.addEventListener('click', function() { dismissToast(toast); });
    container.appendChild(toast);
    setTimeout(function() { dismissToast(toast); }, duration);
}

function dismissToast(toast) {
    if (toast.classList.contains('toast-dismiss')) return;
    toast.classList.add('toast-dismiss');
    setTimeout(function() { if (toast.parentNode) toast.parentNode.removeChild(toast); }, 300);
}

// --- Button Loading Helper ---
function setButtonLoading(btn, loading) {
    if (!btn) return;
    if (loading) {
        btn.dataset.originalText = btn.textContent;
        btn.classList.add('btn-loading');
        btn.disabled = true;
    } else {
        btn.classList.remove('btn-loading');
        btn.disabled = false;
        if (btn.dataset.originalText) {
            btn.textContent = btn.dataset.originalText;
        }
    }
}

// --- CSRF Token Helper ---
function getCSRFToken() {
    var meta = document.querySelector('meta[name="csrf-token"]');
    return meta ? meta.getAttribute('content') : '';
}

// --- Fetch JSON Wrapper ---
async function fetchJSON(url, opts) {
    try {
        opts = opts || {};
        // Auto-inject CSRF token on mutation methods
        var method = (opts.method || 'GET').toUpperCase();
        if (method === 'POST' || method === 'PUT' || method === 'DELETE' || method === 'PATCH') {
            opts.headers = opts.headers || {};
            if (!opts.headers['X-CSRF-Token']) {
                opts.headers['X-CSRF-Token'] = getCSRFToken();
            }
        }
        var resp = await fetch(url, opts);
        if (!resp.ok) {
            var err;
            try { err = await resp.json(); } catch(e) { err = {}; }
            var msg = err.error || resp.statusText || 'Request failed';
            showToast(msg, 'error');
            throw new Error(msg);
        }
        return await resp.json();
    } catch (e) {
        if (e.message && !e.message.match(/Request failed|Failed/)) {
            showToast('Network error: ' + e.message, 'error');
        }
        throw e;
    }
}

// Theme: init from localStorage or system preference, default dark
function initTheme() {
    var saved = localStorage.getItem('wave-theme');
    if (saved) {
        document.documentElement.setAttribute('data-theme', saved);
    }
    updateToggleIcon();
}

function toggleTheme() {
    var current = document.documentElement.getAttribute('data-theme');
    var isDark;
    if (current === 'light') {
        isDark = true;
    } else if (current === 'dark') {
        isDark = false;
    } else {
        // No explicit theme — check system preference
        isDark = !window.matchMedia('(prefers-color-scheme: light)').matches;
    }

    var next = isDark ? 'dark' : 'light';
    document.documentElement.setAttribute('data-theme', next);
    localStorage.setItem('wave-theme', next);
    updateToggleIcon();
}

function updateToggleIcon() {
    var btn = document.getElementById('theme-toggle');
    if (!btn) return;
    var theme = document.documentElement.getAttribute('data-theme');
    if (!theme) {
        // Use system preference to determine current icon
        theme = window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
    }
    // Moon for dark mode, sun for light mode
    btn.innerHTML = theme === 'light' ? '&#9728;' : '&#9790;';
    btn.title = theme === 'light' ? 'Switch to dark mode' : 'Switch to light mode';
}

// Initialize theme on load
initTheme();

// Bind toggle button
(function() {
    var btn = document.getElementById('theme-toggle');
    if (btn) btn.addEventListener('click', toggleTheme);
})();

// Toggle start pipeline form
function toggleStartForm() {
    var dialog = document.getElementById('start-dialog');
    if (!dialog) return;
    if (dialog.open) {
        dialog.close();
    } else {
        dialog.showModal();
        // Populate adapter dropdown on first open
        populateAdapterDropdown();
    }
}

// Populate adapter dropdown from API
var adaptersPopulated = false;
function populateAdapterDropdown() {
    if (adaptersPopulated) return;
    var sel = document.getElementById('adapter-select');
    if (!sel) return;
    fetchJSON('/api/adapters')
        .then(function(data) {
            if (data.adapters && data.adapters.length > 0) {
                // Keep the "Default" option, add others
                for (var i = 0; i < data.adapters.length; i++) {
                    var opt = document.createElement('option');
                    opt.value = data.adapters[i];
                    opt.textContent = data.adapters[i];
                    sel.appendChild(opt);
                }
                adaptersPopulated = true;
            }
        })
        .catch(function() { /* ignore */ });
}

// Collect advanced options from quickstart dialogs.
// The prefix parameter matches the element ID prefix (e.g. "qs" for "qs-model").
function collectAdvancedOptions(prefix) {
    var opts = {};
    var modelEl = document.getElementById(prefix + '-model');
    if (modelEl && modelEl.value) opts.model = modelEl.value;
    var fromStepEl = document.getElementById(prefix + '-from-step');
    if (fromStepEl && fromStepEl.value) opts.from_step = fromStepEl.value;
    var dryRunEl = document.getElementById(prefix + '-dry-run');
    if (dryRunEl && dryRunEl.checked) opts.dry_run = true;
    // Collect selected steps
    var stepsCbs = document.querySelectorAll('input[name="' + prefix + '-steps"]:checked');
    if (stepsCbs.length > 0) {
        var steps = [];
        for (var i = 0; i < stepsCbs.length; i++) steps.push(stepsCbs[i].value);
        opts.steps = steps.join(',');
    }
    // Collect excluded steps
    var excludeCbs = document.querySelectorAll('input[name="' + prefix + '-exclude"]:checked');
    if (excludeCbs.length > 0) {
        var exclude = [];
        for (var j = 0; j < excludeCbs.length; j++) exclude.push(excludeCbs[j].value);
        opts.exclude = exclude.join(',');
    }
    return opts;
}

// Start a pipeline via API
async function startPipeline(e) {
    e.preventDefault();
    const pipeline = document.getElementById('pipeline-select').value;
    const input = document.getElementById('pipeline-input').value;
    const adapter = document.getElementById('adapter-select').value;
    const model = document.getElementById('model-input').value.trim();
    const timeout = parseInt(document.getElementById('timeout-input').value, 10) || 0;
    const dryRun = document.getElementById('dry-run-checkbox').checked;
    const btn = e.submitter || e.target.querySelector('[type="submit"]');

    if (!pipeline) {
        showToast('Please select a pipeline', 'error');
        return false;
    }

    var body = {input: input};
    if (adapter) body.adapter = adapter;
    if (model) body.model = model;
    if (timeout > 0) body.timeout = timeout;
    if (dryRun) body.dry_run = true;
    // Step selection
    if (typeof getSelectedSteps === 'function') {
        var stepSel = getSelectedSteps();
        if (stepSel.steps) body.steps = stepSel.steps;
        if (stepSel.exclude) body.exclude = stepSel.exclude;
    }

    setButtonLoading(btn, true);
    try {
        var data = await fetchJSON('/api/pipelines/' + encodeURIComponent(pipeline) + '/start', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(body)
        });
        showToast('Pipeline started: ' + (data.run_id || pipeline), 'success', 3000);
        var dialog = document.getElementById('start-dialog');
        if (dialog) dialog.close();
        setTimeout(function() { window.location.reload(); }, 500);
    } catch (err) {
        // fetchJSON already showed toast
    } finally {
        setButtonLoading(btn, false);
    }
    return false;
}

// Cancel a running pipeline
async function cancelRun(runID, btn) {
    if (!confirm('Cancel this pipeline run?')) return;

    setButtonLoading(btn, true);
    try {
        await fetchJSON('/api/runs/' + encodeURIComponent(runID) + '/cancel', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({force: false})
        });
        window.location.reload();
    } catch (err) {
        // fetchJSON already showed toast
    } finally {
        setButtonLoading(btn, false);
    }
}

// Force-cancel a stale run (process no longer running)
async function forceCancel(runID, btn) {
    if (!confirm('Force-cancel this run? Use this for stale runs whose process is no longer running.')) return;
    setButtonLoading(btn, true);
    try {
        await fetchJSON('/api/runs/' + encodeURIComponent(runID) + '/cancel', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({force: true})
        });
        window.location.reload();
    } catch (err) {
        // fetchJSON already showed toast
    } finally {
        setButtonLoading(btn, false);
    }
}

// Retry a failed pipeline
async function retryRun(runID, btn) {
    setButtonLoading(btn, true);
    try {
        const data = await fetchJSON('/api/runs/' + encodeURIComponent(runID) + '/retry', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'}
        });
        showToast('Retry started, redirecting...', 'success', 3000);
        setTimeout(function() { window.location.href = '/runs/' + data.run_id; }, 500);
    } catch (err) {
        // fetchJSON already showed toast
    } finally {
        setButtonLoading(btn, false);
    }
}

// Filter runs by status, pipeline, date, and search text
function filterRuns() {
    var params = new URLSearchParams();
    var status = document.getElementById('status-filter');
    var pipeline = document.getElementById('pipeline-filter');
    var since = document.getElementById('since-filter');
    var search = document.getElementById('search-filter');
    if (status && status.value) params.set('status', status.value);
    if (pipeline && pipeline.value) params.set('pipeline', pipeline.value);
    if (since && since.value) params.set('since', since.value);
    if (search && search.value.trim()) params.set('search', search.value.trim());
    window.location.search = params.toString();
}

// Clear all filters
function clearFilters() {
    window.location.href = '/runs';
}

// Resume from a specific step
async function resumeFromStep(runID, stepID, btn) {
    setButtonLoading(btn, true);
    try {
        const data = await fetchJSON('/api/runs/' + encodeURIComponent(runID) + '/resume', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({from_step: stepID})
        });
        showToast('Resuming from ' + stepID + '...', 'success', 3000);
        setTimeout(function() { window.location.href = '/runs/' + data.run_id; }, 500);
    } catch (err) {
        // fetchJSON already showed toast
    } finally {
        setButtonLoading(btn, false);
    }
}

function showResumeDialog() {
    var dialog = document.getElementById('resume-dialog');
    if (dialog) dialog.showModal();
}

function showForkDialog() {
    var dialog = document.getElementById('fork-dialog');
    if (dialog) {
        var dropdown = document.getElementById('fork-step-dropdown');
        var confirmBtn = document.getElementById('fork-confirm-btn');
        if (dropdown) { dropdown.value = ''; dropdown.onchange = function() { confirmBtn.disabled = !dropdown.value; }; }
        if (confirmBtn) confirmBtn.disabled = true;
        dialog.showModal();
    }
}

async function forkFromStep(runID, btn) {
    var dropdown = document.getElementById('fork-step-dropdown');
    if (!dropdown || !dropdown.value) return;
    setButtonLoading(btn, true);
    try {
        var data = await fetchJSON('/api/runs/' + encodeURIComponent(runID) + '/fork', {
            method: 'POST', headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({from_step: dropdown.value})
        });
        showToast('Forked from ' + dropdown.value + ', redirecting...', 'success', 3000);
        setTimeout(function() { window.location.href = '/runs/' + data.run_id; }, 500);
    } catch (err) { /* fetchJSON showed toast */ } finally { setButtonLoading(btn, false); }
}

function showRewindDialog() {
    var dialog = document.getElementById('rewind-dialog');
    if (dialog) {
        var dropdown = document.getElementById('rewind-step-dropdown');
        var confirmBtn = document.getElementById('rewind-confirm-btn');
        if (dropdown) { dropdown.value = ''; dropdown.onchange = function() { confirmBtn.disabled = !dropdown.value; }; }
        if (confirmBtn) confirmBtn.disabled = true;
        dialog.showModal();
    }
}

async function rewindToStep(runID, btn) {
    var dropdown = document.getElementById('rewind-step-dropdown');
    if (!dropdown || !dropdown.value) return;
    if (!confirm('This will permanently delete state for all steps after "' + dropdown.value + '". Continue?')) return;
    setButtonLoading(btn, true);
    try {
        await fetchJSON('/api/runs/' + encodeURIComponent(runID) + '/rewind', {
            method: 'POST', headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({to_step: dropdown.value})
        });
        showToast('Rewound to ' + dropdown.value + ', reloading...', 'success', 3000);
        setTimeout(function() { window.location.reload(); }, 500);
    } catch (err) { /* fetchJSON showed toast */ } finally { setButtonLoading(btn, false); }
}

// Auto-refresh run list every 10 seconds if there are running pipelines
(function() {
    var rows = document.querySelectorAll('.run-row[data-status="running"]');
    if (rows.length > 0) {
        setTimeout(function() {
            window.location.reload();
        }, 10000);
    }
})();

// Relative time formatting
function relativeTime(isoString) {
    if (!isoString) return '-';
    var date = new Date(isoString);
    var now = new Date();
    var diffMs = now - date;
    var diffS = Math.floor(diffMs / 1000);
    if (diffS < 60) return diffS + 's ago';
    var diffM = Math.floor(diffS / 60);
    if (diffM < 60) return diffM + 'm ago';
    var diffH = Math.floor(diffM / 60);
    if (diffH < 24) return diffH + 'h ago';
    var diffD = Math.floor(diffH / 24);
    return diffD + 'd ago';
}

// Update all <time> elements with relative timestamps
(function() {
    var times = document.querySelectorAll('.run-row time[datetime]');
    for (var i = 0; i < times.length; i++) {
        var dt = times[i].getAttribute('datetime');
        if (dt) {
            times[i].textContent = relativeTime(dt);
        }
    }
})();

// Client-side table sorting
function sortTable(column) {
    var table = document.getElementById('runs-table');
    if (!table) return;
    var thead = table.querySelector('thead');
    var tbody = table.querySelector('tbody');
    if (!thead || !tbody) return;

    var th = thead.querySelector('[data-sort="' + column + '"]');
    if (!th) return;

    // Determine sort direction
    var currentDir = th.getAttribute('data-dir');
    var dir = currentDir === 'asc' ? 'desc' : 'asc';

    // Reset all headers
    var allThs = thead.querySelectorAll('.sortable');
    for (var i = 0; i < allThs.length; i++) {
        allThs[i].classList.remove('sort-active');
        allThs[i].removeAttribute('data-dir');
        allThs[i].setAttribute('aria-sort', 'none');
        var ind = allThs[i].querySelector('.sort-indicator');
        if (ind) ind.textContent = '';
    }

    // Set active header
    th.classList.add('sort-active');
    th.setAttribute('data-dir', dir);
    th.setAttribute('aria-sort', dir === 'asc' ? 'ascending' : 'descending');
    var indicator = th.querySelector('.sort-indicator');
    if (indicator) indicator.textContent = dir === 'asc' ? '▲' : '▼';

    // Get column index
    var colIndex = 0;
    var cols = thead.querySelectorAll('th');
    for (var j = 0; j < cols.length; j++) {
        if (cols[j] === th) { colIndex = j; break; }
    }

    // Sort rows
    var rows = Array.prototype.slice.call(tbody.querySelectorAll('tr'));
    rows.sort(function(a, b) {
        var aVal, bVal;
        var aCells = a.querySelectorAll('td');
        var bCells = b.querySelectorAll('td');
        if (colIndex >= aCells.length || colIndex >= bCells.length) return 0;

        if (column === 'started') {
            var aTime = aCells[colIndex].querySelector('time');
            var bTime = bCells[colIndex].querySelector('time');
            aVal = aTime ? aTime.getAttribute('datetime') || '' : '';
            bVal = bTime ? bTime.getAttribute('datetime') || '' : '';
        } else if (column === 'duration') {
            aVal = parseDuration(aCells[colIndex].getAttribute('data-duration') || aCells[colIndex].textContent.trim());
            bVal = parseDuration(bCells[colIndex].getAttribute('data-duration') || bCells[colIndex].textContent.trim());
            var cmp = aVal - bVal;
            return dir === 'asc' ? cmp : -cmp;
        } else {
            aVal = aCells[colIndex].textContent.trim().toLowerCase();
            bVal = bCells[colIndex].textContent.trim().toLowerCase();
        }

        if (aVal < bVal) return dir === 'asc' ? -1 : 1;
        if (aVal > bVal) return dir === 'asc' ? 1 : -1;
        return 0;
    });

    for (var k = 0; k < rows.length; k++) {
        tbody.appendChild(rows[k]);
    }

    // Persist sort state in URL
    var params = new URLSearchParams(window.location.search);
    params.set('sort', column);
    params.set('dir', dir);
    history.replaceState(null, '', '?' + params.toString());
}

// Parse duration string like "2m30s", "45s", "1h 30m" into seconds
function parseDuration(s) {
    if (!s || s === '-') return 0;
    var total = 0;
    var h = s.match(/(\d+)h/);
    if (h) total += parseInt(h[1], 10) * 3600;
    var m = s.match(/(\d+)m/);
    if (m) total += parseInt(m[1], 10) * 60;
    var sec = s.match(/(\d+)s/);
    if (sec) total += parseInt(sec[1], 10);
    return total;
}

// Restore sort state from URL on page load
(function() {
    var params = new URLSearchParams(window.location.search);
    var sort = params.get('sort');
    var dir = params.get('dir');
    if (sort) {
        // Set direction to opposite so sortTable toggles to desired direction
        var table = document.getElementById('runs-table');
        if (table) {
            var th = table.querySelector('[data-sort="' + sort + '"]');
            if (th) {
                th.setAttribute('data-dir', dir === 'asc' ? 'desc' : 'asc');
                sortTable(sort);
            }
        }
    }
})();

// Row click handler — navigate to run detail on row click
(function() {
    var rows = document.querySelectorAll('.run-row[data-href]');
    for (var i = 0; i < rows.length; i++) {
        rows[i].addEventListener('click', function(e) {
            // Don't navigate if clicking a link within the row
            if (e.target.tagName === 'A' || e.target.closest('a')) return;
            var href = this.getAttribute('data-href');
            if (href) window.location.href = href;
        });
    }
})();

// Mobile navigation toggle
function toggleNav() {
    toggleSidebar();
}

// Sidebar toggle (mobile overlay)
function toggleSidebar() {
    var sidebar = document.getElementById('sidebar');
    var btn = document.querySelector('.sidebar-toggle');
    if (sidebar) {
        var open = sidebar.classList.toggle('open');
        if (btn) btn.setAttribute('aria-expanded', open ? 'true' : 'false');
    }
}

function closeSidebar() {
    var sidebar = document.getElementById('sidebar');
    var btn = document.querySelector('.sidebar-toggle');
    if (sidebar) sidebar.classList.remove('open');
    if (btn) btn.setAttribute('aria-expanded', 'false');
}

// Sidebar nav group collapse/expand with localStorage persistence
function toggleNavGroup(group) {
    var el = document.querySelector('.nav-group[data-group="' + group + '"]');
    if (!el) return;
    var collapsed = el.classList.toggle('collapsed');
    var btn = el.querySelector('.nav-group-toggle');
    if (btn) btn.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
    // Persist state
    var state = {};
    try { state = JSON.parse(localStorage.getItem('wave-nav-groups') || '{}'); } catch(e) {}
    state[group] = collapsed;
    localStorage.setItem('wave-nav-groups', JSON.stringify(state));
}

// Restore collapsed nav groups from localStorage
(function() {
    try {
        var state = JSON.parse(localStorage.getItem('wave-nav-groups') || '{}');
        for (var group in state) {
            if (state[group]) {
                var el = document.querySelector('.nav-group[data-group="' + group + '"]');
                if (el) {
                    el.classList.add('collapsed');
                    var btn = el.querySelector('.nav-group-toggle');
                    if (btn) btn.setAttribute('aria-expanded', 'false');
                }
            }
        }
    } catch(e) {}
})();

// Live elapsed timer for running steps
(function() {
    function formatElapsed(ms) {
        var s = Math.floor(ms / 1000);
        if (s < 60) return s + 's';
        var m = Math.floor(s / 60);
        s = s % 60;
        if (m < 60) return s === 0 ? m + 'm' : m + 'm ' + s + 's';
        var h = Math.floor(m / 60);
        m = m % 60;
        return m === 0 ? h + 'h' : h + 'h ' + m + 'm';
    }

    function updateTimers() {
        var timers = document.querySelectorAll('[data-live-timer="true"]');
        var now = Date.now();
        for (var i = 0; i < timers.length; i++) {
            var card = timers[i].closest('.step-card');
            if (!card) continue;
            var startedAt = card.getAttribute('data-started-at');
            if (!startedAt) continue;
            var elapsed = now - new Date(startedAt).getTime();
            if (elapsed >= 0) {
                timers[i].textContent = formatElapsed(elapsed);
            }
        }
    }

    // Update every second
    if (document.querySelectorAll('[data-live-timer="true"]').length > 0) {
        setInterval(updateTimers, 1000);
        updateTimers();
    }
})();

// --- Gate Interaction ---

// Track the currently selected gate choice per panel
var gateSelections = {};

function selectGateChoice(btn) {
    var panel = btn.closest('.gate-interaction-panel');
    if (!panel) return;
    var stepID = panel.dataset.stepId;
    var key = btn.dataset.choiceKey;

    // Deselect all buttons in this panel
    var buttons = panel.querySelectorAll('.gate-choice-btn');
    for (var i = 0; i < buttons.length; i++) {
        buttons[i].classList.remove('gate-choice-selected');
    }

    // Select this button
    btn.classList.add('gate-choice-selected');
    gateSelections[stepID] = key;

    // Enable submit button and show selected label
    var submitBtn = document.getElementById('gate-submit-' + stepID);
    if (submitBtn) submitBtn.disabled = false;
    var selectedLabel = document.getElementById('gate-selected-' + stepID);
    if (selectedLabel) selectedLabel.textContent = 'Selected: ' + btn.textContent.trim();
}

async function submitGateDecision(runID, stepID, btn) {
    var key = gateSelections[stepID];
    if (!key) {
        showToast('Please select a choice first', 'error');
        return;
    }

    // Check if the selected choice targets _fail (pipeline abort) and confirm
    var panel = document.getElementById('gate-panel-' + stepID);
    if (panel) {
        var selectedBtn = panel.querySelector('.gate-choice-btn[data-choice-key="' + key + '"]');
        if (selectedBtn && selectedBtn.dataset.choiceTarget === '_fail') {
            if (!confirm('This choice will abort the pipeline. Are you sure?')) {
                return;
            }
        }
    }

    // Disable immediately to prevent double-submit
    btn.disabled = true;

    var freeformInput = document.getElementById('gate-text-' + stepID);
    var text = freeformInput ? freeformInput.value : '';

    setButtonLoading(btn, true);
    try {
        await approveGate(runID, stepID, key, text);
        showToast('Gate decision submitted', 'success', 3000);

        // Clean up selection state
        delete gateSelections[stepID];

        // Disable the panel after submission
        if (panel) {
            panel.classList.add('gate-panel-submitted');
            var buttons = panel.querySelectorAll('button, textarea');
            for (var i = 0; i < buttons.length; i++) {
                buttons[i].disabled = true;
            }
        }
    } catch (err) {
        // Re-enable submit button on failure so the user can retry
        btn.disabled = false;
    } finally {
        setButtonLoading(btn, false);
    }
}

async function approveGate(runID, stepID, choiceKey, freeformText) {
    var body = { choice: choiceKey };
    if (freeformText) {
        body.text = freeformText;
    }
    return fetchJSON('/api/runs/' + encodeURIComponent(runID) + '/gates/' + encodeURIComponent(stepID) + '/approve', {
        method: 'POST',
        headers: {'Content-Type': 'application/json', 'X-Wave-Request': '1'},
        body: JSON.stringify(body)
    });
}

// Gate keyboard shortcuts: when a gate panel is visible, pressing a choice key selects it
(function() {
    document.addEventListener('keydown', function(e) {
        // Ignore if user is typing in an input/textarea (except gate freeform)
        var tag = (e.target.tagName || '').toLowerCase();
        if (tag === 'input' || (tag === 'textarea' && !e.target.classList.contains('gate-freeform-input'))) return;

        var panels = document.querySelectorAll('.gate-interaction-panel:not(.gate-panel-submitted)');
        for (var i = 0; i < panels.length; i++) {
            var buttons = panels[i].querySelectorAll('.gate-choice-btn');
            for (var j = 0; j < buttons.length; j++) {
                if (buttons[j].dataset.choiceKey === e.key) {
                    e.preventDefault();
                    selectGateChoice(buttons[j]);
                    return;
                }
            }
        }
    });
})();
