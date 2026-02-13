// Wave Dashboard - Main Application JS

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

// Filter runs by status
function filterRuns() {
    const status = document.getElementById('status-filter').value;
    const params = new URLSearchParams(window.location.search);
    if (status) {
        params.set('status', status);
    } else {
        params.delete('status');
    }
    window.location.search = params.toString();
}

// Auto-refresh run list every 10 seconds if there are running pipelines
(function() {
    const rows = document.querySelectorAll('.run-row[data-status="running"]');
    if (rows.length > 0) {
        setTimeout(function() {
            window.location.reload();
        }, 10000);
    }
})();
