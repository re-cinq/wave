// Wave Dashboard - Main Application JS

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
    const form = document.getElementById('start-form');
    if (form) {
        form.style.display = form.style.display === 'none' ? 'block' : 'none';
    }
}

// Start a pipeline via API
async function startPipeline(e) {
    e.preventDefault();
    const pipeline = document.getElementById('pipeline-select').value;
    const input = document.getElementById('pipeline-input').value;

    if (!pipeline) {
        alert('Please select a pipeline');
        return false;
    }

    try {
        const resp = await fetch('/api/pipelines/' + encodeURIComponent(pipeline) + '/start', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({input: input})
        });

        if (!resp.ok) {
            const err = await resp.json();
            alert('Failed to start pipeline: ' + (err.error || resp.statusText));
            return false;
        }

        toggleStartForm();
        window.location.reload();
    } catch (err) {
        alert('Error: ' + err.message);
    }
    return false;
}

// Cancel a running pipeline
async function cancelRun(runID) {
    if (!confirm('Cancel this pipeline run?')) return;

    try {
        const resp = await fetch('/api/runs/' + encodeURIComponent(runID) + '/cancel', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({force: false})
        });

        if (resp.ok) {
            window.location.reload();
        } else {
            const err = await resp.json();
            alert('Failed to cancel: ' + (err.error || resp.statusText));
        }
    } catch (err) {
        alert('Error: ' + err.message);
    }
}

// Retry a failed pipeline
async function retryRun(runID) {
    try {
        const resp = await fetch('/api/runs/' + encodeURIComponent(runID) + '/retry', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'}
        });

        if (resp.ok) {
            const data = await resp.json();
            window.location.href = '/runs/' + data.run_id;
        } else {
            const err = await resp.json();
            alert('Failed to retry: ' + (err.error || resp.statusText));
        }
    } catch (err) {
        alert('Error: ' + err.message);
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
async function resumeFromStep(runID, stepID) {
    try {
        const resp = await fetch('/api/runs/' + encodeURIComponent(runID) + '/resume', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({from_step: stepID})
        });

        if (resp.ok) {
            const data = await resp.json();
            window.location.href = '/runs/' + data.run_id;
        } else {
            const err = await resp.json();
            alert('Failed to resume: ' + (err.error || resp.statusText));
        }
    } catch (err) {
        alert('Error: ' + err.message);
    }
}

function showResumeDialog() {
    var dialog = document.getElementById('resume-dialog');
    if (dialog) dialog.style.display = 'flex';
}

// Keyboard handlers for interactive elements
document.addEventListener('keydown', function(e) {
    // Escape closes the start form or resume dialog if open
    if (e.key === 'Escape') {
        var form = document.getElementById('start-form');
        if (form && form.style.display !== 'none') {
            toggleStartForm();
        }
        var dialog = document.getElementById('resume-dialog');
        if (dialog && dialog.style.display !== 'none') {
            dialog.style.display = 'none';
        }
    }
});

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
        var ind = allThs[i].querySelector('.sort-indicator');
        if (ind) ind.textContent = '';
    }

    // Set active header
    th.classList.add('sort-active');
    th.setAttribute('data-dir', dir);
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
