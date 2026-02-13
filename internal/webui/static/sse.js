// Wave Dashboard - Server-Sent Events client

var sseConnection = null;

function connectSSE(runID) {
    if (sseConnection) {
        sseConnection.close();
    }

    var url = '/api/runs/' + encodeURIComponent(runID) + '/events';
    sseConnection = new EventSource(url);

    sseConnection.onopen = function() {
        console.log('SSE connected for run:', runID);
    };

    sseConnection.onerror = function() {
        console.log('SSE connection error, will auto-reconnect...');
    };

    // Listen for all event types
    var eventTypes = ['started', 'running', 'completed', 'failed', 'step_progress', 'stream_activity', 'eta_updated'];

    eventTypes.forEach(function(type) {
        sseConnection.addEventListener(type, function(e) {
            handleSSEEvent(type, JSON.parse(e.data));
        });
    });
}

function handleSSEEvent(type, data) {
    // Update status badge if run status changed
    if (type === 'completed' || type === 'failed') {
        var badges = document.querySelectorAll('.badge.status-running');
        badges.forEach(function(badge) {
            badge.className = 'badge status-' + type;
            badge.textContent = type;
        });
        // Reload to get final state
        setTimeout(function() { window.location.reload(); }, 1000);
    }

    // Update step progress
    if (type === 'step_progress' && data.step_id) {
        var stepCard = document.querySelector('.step-card .step-id');
        // Find matching step card and update progress
        var cards = document.querySelectorAll('.step-card');
        cards.forEach(function(card) {
            var idEl = card.querySelector('.step-id');
            if (idEl && idEl.textContent === data.step_id) {
                var progressFill = card.querySelector('.progress-fill');
                if (progressFill) {
                    progressFill.style.width = data.progress + '%';
                    progressFill.textContent = data.progress + '%';
                }
                var actionEl = card.querySelector('.step-action');
                if (actionEl && data.current_action) {
                    actionEl.textContent = data.current_action;
                }
            }
        });
    }
}

// Disconnect when leaving page
window.addEventListener('beforeunload', function() {
    if (sseConnection) {
        sseConnection.close();
    }
});
