// Wave Dashboard - Server-Sent Events client with polling fallback

// formatTokens formats a token count to abbreviated human-readable form,
// mirroring the Go FormatTokenCount logic for k/M/B thresholds.
function formatTokens(n) {
    if (n == null || n === undefined) return '0';
    if (n < 1000) return String(n);
    if (n < 1000000) return (n / 1000).toFixed(1) + 'k';
    if (n < 1000000000) return (n / 1000000).toFixed(1) + 'M';
    return (n / 1000000000).toFixed(1) + 'B';
}

var sseConnection = null;
var pollTimer = null;
var currentRunID = null;

function connectSSE(runID) {
    currentRunID = runID;

    if (sseConnection) {
        sseConnection.close();
    }

    var url = '/api/runs/' + encodeURIComponent(runID) + '/events';
    sseConnection = new EventSource(url);

    sseConnection.onopen = function() {
        console.log('SSE connected for run:', runID);
    };

    sseConnection.onerror = function() {
        console.log('SSE error, falling back to polling...');
        // If SSE fails, start polling as fallback
        startPolling(runID);
    };

    // Listen for all event types
    var eventTypes = ['started', 'running', 'completed', 'failed', 'step_progress', 'stream_activity', 'eta_updated'];

    eventTypes.forEach(function(type) {
        sseConnection.addEventListener(type, function(e) {
            handleSSEEvent(type, JSON.parse(e.data));
        });
    });

    // Also start polling as a belt-and-suspenders approach —
    // the page fully refreshes every 3s to pick up new steps/DAG state
    startPolling(runID);
}

function startPolling(runID) {
    if (pollTimer) return; // already polling
    pollTimer = setInterval(function() {
        fetch('/api/runs/' + encodeURIComponent(runID))
            .then(function(r) { return r.json(); })
            .then(function(data) {
                updatePageFromAPI(data);
                // Stop polling when run is terminal
                if (data.run.status === 'completed' || data.run.status === 'failed' || data.run.status === 'cancelled') {
                    stopPolling();
                    if (sseConnection) sseConnection.close();
                    // One final full reload for clean state
                    window.location.reload();
                }
            })
            .catch(function() {}); // silently ignore fetch errors
    }, 3000);
}

function stopPolling() {
    if (pollTimer) {
        clearInterval(pollTimer);
        pollTimer = null;
    }
}

function updatePageFromAPI(data) {
    // Update run status badge
    var headerBadge = document.querySelector('.page-header .badge');
    if (headerBadge && data.run.status) {
        headerBadge.className = 'badge status-' + data.run.status;
        headerBadge.textContent = data.run.status;
    }

    // Update step cards from API data
    var stepsList = document.querySelector('.steps-list');
    if (stepsList && data.steps && data.steps.length > 0) {
        stepsList.innerHTML = '';
        data.steps.forEach(function(step) {
            stepsList.appendChild(createStepCard(step));
        });
    }

    // Update DAG nodes from step data
    if (data.steps) {
        data.steps.forEach(function(step) {
            updateDAGNodeStatus(step.step_id, step.state);
        });
    }

    // Update events timeline
    if (data.events && data.events.length > 0) {
        var timeline = document.querySelector('.events-timeline');
        if (!timeline) {
            var eventsCard = document.createElement('div');
            eventsCard.className = 'card';
            eventsCard.innerHTML = '<h2>Events</h2><div class="events-timeline"></div>';
            document.querySelector('.container').appendChild(eventsCard);
            timeline = eventsCard.querySelector('.events-timeline');
        }
        timeline.innerHTML = '';
        data.events.forEach(function(ev) {
            if (!ev.message && (ev.state === 'step_progress' || ev.state === 'stream_activity')) return;
            var item = document.createElement('div');
            item.className = 'event-item';
            var t = new Date(ev.timestamp).toLocaleString();
            var stepHtml = ev.step_id ? '<span class="event-step">' + ev.step_id + '</span>' : '';
            item.innerHTML =
                '<span class="event-time">' + t + '</span>' +
                '<span class="badge status-' + ev.state + '">' + ev.state + '</span>' +
                stepHtml +
                '<span class="event-message">' + (ev.message || '') + '</span>';
            timeline.appendChild(item);
        });
    }
}

function createStepCard(step) {
    var card = document.createElement('div');
    card.className = 'step-card status-' + step.state;

    var headerHtml =
        '<div class="step-header">' +
        '<span class="step-id">' + step.step_id + '</span>' +
        '<span class="badge status-' + step.state + '">' + step.state + '</span>' +
        '<span class="step-persona">' + (step.persona || '') + '</span>' +
        '</div>';

    var bodyParts = [];
    if (step.current_action) {
        bodyParts.push('<div class="step-action">' + step.current_action + '</div>');
    }
    if (step.progress > 0) {
        bodyParts.push(
            '<div class="progress-bar">' +
            '<div class="progress-fill" style="width: ' + step.progress + '%">' + step.progress + '%</div>' +
            '</div>'
        );
    }
    var metaParts = [];
    if (step.duration) metaParts.push('<span>Duration: ' + step.duration + '</span>');
    if (step.state === 'completed' || step.state === 'failed' || step.state === 'running') {
        metaParts.push('<span class="token-count" data-step-id="' + step.step_id + '">Tokens: ' + formatTokens(step.tokens_used) + '</span>');
    }
    if (metaParts.length) {
        bodyParts.push('<div class="step-meta">' + metaParts.join(' ') + '</div>');
    }
    if (step.error) {
        bodyParts.push('<div class="step-error">' + step.error + '</div>');
    }

    card.innerHTML = headerHtml + '<div class="step-body">' + bodyParts.join('') + '</div>';
    return card;
}

function handleSSEEvent(type, data) {
    // Update status badge if run status changed
    if (type === 'completed' || type === 'failed') {
        var badges = document.querySelectorAll('.badge.status-running');
        badges.forEach(function(badge) {
            badge.className = 'badge status-' + type;
            badge.textContent = type;
        });
        stopPolling();
        if (sseConnection) sseConnection.close();
        // Reload to get final state
        setTimeout(function() { window.location.reload(); }, 500);
    }

    // Update DAG node status in real-time
    if (data.step_id) {
        var stateForDAG = type;
        if (type === 'running' || type === 'started') stateForDAG = 'running';
        updateDAGNodeStatus(data.step_id, stateForDAG);
    }

    // Update step card status
    if (data.step_id && (type === 'started' || type === 'running')) {
        updateStepCardState(data.step_id, 'running');
    }
    if (data.step_id && type === 'completed') {
        updateStepCardState(data.step_id, 'completed');
    }
    if (data.step_id && type === 'failed') {
        updateStepCardState(data.step_id, 'failed');
    }

    // Update step card tokens in real-time
    if (data.step_id && data.tokens_used !== undefined) {
        updateStepCardTokens(data.step_id, data.tokens_used);
    }

    // Append meaningful events to the timeline (skip empty heartbeats)
    if (data.message && type !== 'step_progress' && type !== 'stream_activity') {
        appendEventToTimeline(data, type);
    }
}

function updateDAGNodeStatus(stepID, status) {
    var statusMap = {
        'started': 'running',
        'running': 'running',
        'completed': 'completed',
        'failed': 'failed'
    };
    var cssStatus = statusMap[status] || status;

    var node = document.querySelector('.dag-node[data-id="' + stepID + '"]');
    if (node) {
        node.setAttribute('data-status', cssStatus);
        var rect = node.querySelector('rect');
        if (rect) {
            rect.setAttribute('class', 'dag-node-rect ' + cssStatus);
        }
    }
}

function updateStepCardTokens(stepID, tokensUsed) {
    var cards = document.querySelectorAll('.step-card');
    cards.forEach(function(card) {
        var idEl = card.querySelector('.step-id');
        if (idEl && idEl.textContent === stepID) {
            var tokenSpan = card.querySelector('.token-count');
            if (tokenSpan) {
                tokenSpan.textContent = 'Tokens: ' + formatTokens(tokensUsed);
            } else {
                // Step transitioned from pending to running — create token span
                var meta = card.querySelector('.step-meta');
                if (!meta) {
                    var body = card.querySelector('.step-body');
                    if (body) {
                        meta = document.createElement('div');
                        meta.className = 'step-meta';
                        body.appendChild(meta);
                    }
                }
                if (meta) {
                    var span = document.createElement('span');
                    span.className = 'token-count';
                    span.setAttribute('data-step-id', stepID);
                    span.textContent = 'Tokens: ' + formatTokens(tokensUsed);
                    meta.appendChild(span);
                }
            }
        }
    });
}

function updateStepCardState(stepID, newState) {
    var cards = document.querySelectorAll('.step-card');
    cards.forEach(function(card) {
        var idEl = card.querySelector('.step-id');
        if (idEl && idEl.textContent === stepID) {
            // Update card border color
            card.className = 'step-card status-' + newState;
            // Update badge
            var badge = card.querySelector('.badge');
            if (badge) {
                badge.className = 'badge status-' + newState;
                badge.textContent = newState;
            }
        }
    });
}

function appendEventToTimeline(data, type) {
    var timeline = document.querySelector('.events-timeline');
    if (!timeline) {
        var card = document.createElement('div');
        card.className = 'card';
        card.innerHTML = '<h2>Events</h2><div class="events-timeline"></div>';
        document.querySelector('.container').appendChild(card);
        timeline = card.querySelector('.events-timeline');
    }

    var item = document.createElement('div');
    item.className = 'event-item';

    var time = new Date(data.timestamp).toLocaleString();
    var stepHtml = data.step_id ? '<span class="event-step">' + data.step_id + '</span>' : '';

    item.innerHTML =
        '<span class="event-time">' + time + '</span>' +
        '<span class="badge status-' + type + '">' + type + '</span>' +
        stepHtml +
        '<span class="event-message">' + (data.message || '') + '</span>';

    timeline.appendChild(item);
}

// Disconnect when leaving page
window.addEventListener('beforeunload', function() {
    stopPolling();
    if (sseConnection) {
        sseConnection.close();
    }
});
