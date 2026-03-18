// Wave Dashboard - Main Application JS

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

// --- Fetch JSON Wrapper ---
async function fetchJSON(url, opts) {
    try {
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
    }
}

// Start a pipeline via API
async function startPipeline(e) {
    e.preventDefault();
    const pipeline = document.getElementById('pipeline-select').value;
    const input = document.getElementById('pipeline-input').value;
    const btn = e.submitter || e.target.querySelector('[type="submit"]');

    if (!pipeline) {
        showToast('Please select a pipeline', 'error');
        return false;
    }

    setButtonLoading(btn, true);
    try {
        var data = await fetchJSON('/api/pipelines/' + encodeURIComponent(pipeline) + '/start', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({input: input})
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

// Filter runs by status, pipeline, and date
function filterRuns() {
    var params = new URLSearchParams();
    var status = document.getElementById('status-filter');
    var pipeline = document.getElementById('pipeline-filter');
    var since = document.getElementById('since-filter');
    if (status && status.value) params.set('status', status.value);
    if (pipeline && pipeline.value) params.set('pipeline', pipeline.value);
    if (since && since.value) params.set('since', since.value);
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

// Parse duration string like "2m30s", "45s" into seconds
function parseDuration(s) {
    if (!s || s === '-') return 0;
    var total = 0;
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
    var links = document.getElementById('nav-links');
    var btn = document.querySelector('.nav-toggle');
    if (links) {
        var open = links.classList.toggle('nav-open');
        if (btn) btn.setAttribute('aria-expanded', open ? 'true' : 'false');
    }
}

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
