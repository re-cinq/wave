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

// Filter runs by status, pipeline, and since date
function filterRuns() {
    var status = document.getElementById('status-filter');
    var pipeline = document.getElementById('pipeline-filter');
    var since = document.getElementById('since-filter');
    var params = new URLSearchParams();

    if (status && status.value) params.set('status', status.value);
    if (pipeline && pipeline.value) params.set('pipeline', pipeline.value);
    if (since && since.value) params.set('since', since.value);

    // Remove cursor when filters change
    window.location.search = params.toString();
}

// Clear all filters
function clearFilters() {
    window.location.href = '/runs';
}

// Client-side column sorting
function sortTable(column) {
    var table = document.getElementById('runs-table');
    if (!table) return;

    var headers = table.querySelectorAll('th.sortable');
    var clickedHeader = table.querySelector('th[data-sort="' + column + '"]');
    if (!clickedHeader) return;

    // Determine sort direction
    var isActive = clickedHeader.classList.contains('sort-active');
    var isAsc = clickedHeader.classList.contains('sort-asc');
    var direction = isActive && isAsc ? 'desc' : isActive && !isAsc ? 'asc' : 'asc';

    // Update header classes
    headers.forEach(function(h) {
        h.classList.remove('sort-active', 'sort-asc', 'sort-desc');
    });
    clickedHeader.classList.add('sort-active', 'sort-' + direction);

    // Sort rows
    var tbody = table.querySelector('tbody');
    var rows = Array.from(tbody.querySelectorAll('tr'));

    var colIndex;
    switch (column) {
        case 'status': colIndex = 0; break;
        case 'pipeline': colIndex = 1; break;
        case 'started': colIndex = 3; break;
        case 'duration': colIndex = 4; break;
        default: return;
    }

    rows.sort(function(a, b) {
        var aVal, bVal;
        if (column === 'duration') {
            aVal = parseInt(a.cells[colIndex].getAttribute('data-sort-value') || '0', 10);
            bVal = parseInt(b.cells[colIndex].getAttribute('data-sort-value') || '0', 10);
        } else if (column === 'started') {
            var aTime = a.cells[colIndex].querySelector('time');
            var bTime = b.cells[colIndex].querySelector('time');
            aVal = aTime ? aTime.getAttribute('datetime') : '';
            bVal = bTime ? bTime.getAttribute('datetime') : '';
        } else {
            aVal = (a.cells[colIndex].textContent || '').trim().toLowerCase();
            bVal = (b.cells[colIndex].textContent || '').trim().toLowerCase();
        }

        if (aVal < bVal) return direction === 'asc' ? -1 : 1;
        if (aVal > bVal) return direction === 'asc' ? 1 : -1;
        return 0;
    });

    rows.forEach(function(row) {
        tbody.appendChild(row);
    });
}

// Relative time formatting
function relativeTime(isoString) {
    if (!isoString) return '';
    var date = new Date(isoString);
    var now = new Date();
    var diff = Math.floor((now - date) / 1000);

    if (diff < 5) return 'just now';
    if (diff < 60) return diff + 's ago';
    if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
    if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
    if (diff < 604800) return Math.floor(diff / 86400) + 'd ago';
    return date.toLocaleDateString();
}

// Update all <time> elements with relative timestamps
function updateRelativeTimes() {
    var times = document.querySelectorAll('time[datetime]');
    times.forEach(function(el) {
        var iso = el.getAttribute('datetime');
        if (iso) {
            el.textContent = relativeTime(iso);
        }
    });
}

// Run on load and periodically
updateRelativeTimes();
setInterval(updateRelativeTimes, 30000);

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

// Row click handler — navigate to run detail on row click, except on links
document.addEventListener('click', function(e) {
    var row = e.target.closest('.run-row[data-href]');
    if (!row) return;
    // Don't navigate if clicking on a link
    if (e.target.closest('a')) return;
    window.location.href = row.getAttribute('data-href');
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
