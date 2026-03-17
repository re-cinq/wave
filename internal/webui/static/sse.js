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

// formatDuration formats milliseconds to human-readable duration,
// mirroring Go's formatDurationValue output format.
function formatDuration(ms) {
    if (ms == null || ms === undefined || ms < 0) return '';
    var totalSec = Math.round(ms / 1000);
    if (totalSec < 1) return '<1s';
    if (totalSec < 60) return totalSec + 's';
    if (totalSec < 3600) {
        var m = Math.floor(totalSec / 60);
        var s = totalSec % 60;
        return s === 0 ? m + 'm' : m + 'm ' + s + 's';
    }
    var h = Math.floor(totalSec / 3600);
    var m = Math.floor((totalSec % 3600) / 60);
    return m === 0 ? h + 'h' : h + 'h ' + m + 'm';
}

// formatStartTime formats an ISO timestamp string to a localized display string.
function formatStartTime(isoString) {
    if (!isoString) return '';
    var d = new Date(isoString);
    if (isNaN(d.getTime())) return '';
    var pad = function(n) { return n < 10 ? '0' + n : String(n); };
    return d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate()) +
           ' ' + pad(d.getHours()) + ':' + pad(d.getMinutes()) + ':' + pad(d.getSeconds());
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

    // Update step cards from API data, preserving collapse state
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
    // Determine collapse state: preserve user's choice via expandedSteps set,
    // otherwise default running/failed to expanded and completed/pending to collapsed
    var isExpanded;
    if (typeof expandedSteps !== 'undefined' && expandedSteps.has(step.step_id)) {
        isExpanded = true;
    } else if (typeof expandedSteps !== 'undefined' && expandedSteps.size > 0) {
        // User has interacted — respect their choices, keep others collapsed
        isExpanded = false;
    } else {
        // Default: expand running/failed, collapse others
        isExpanded = (step.state === 'running' || step.state === 'failed');
    }
    var collapsedClass = isExpanded ? '' : ' step-collapsed';
    card.className = 'step-card status-' + step.state + collapsedClass;
    card.id = 'step-' + step.step_id;
    card.setAttribute('data-step-id', step.step_id);
    card.setAttribute('data-step-state', step.state);

    var spinnerHtml = step.state === 'running' ? '<span class="step-running-spinner" aria-hidden="true"></span>' : '';
    var startTimeHtml = step.formatted_started_at
        ? '<span class="step-start-time" title="Started at ' + step.formatted_started_at + '">' + step.formatted_started_at + '</span>'
        : (step.started_at ? '<span class="step-start-time">' + formatStartTime(step.started_at) + '</span>' : '');
    var durationHtml = step.duration ? '<span class="step-duration">' + step.duration + '</span>' : '';

    var headerHtml =
        '<div class="step-header" onclick="toggleStepCard(\'' + step.step_id + '\')" role="button" aria-expanded="' + isExpanded + '" tabindex="0">' +
        '<span class="step-toggle" aria-hidden="true">&#9656;</span>' +
        '<span class="step-id">' + step.step_id + '</span>' +
        '<span class="badge status-' + step.state + '">' + spinnerHtml + step.state + '</span>' +
        '<span class="step-persona">' + (step.persona || '') + '</span>' +
        '<span class="step-header-meta">' + startTimeHtml + durationHtml + '</span>' +
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
    if (step.state === 'completed' || step.state === 'failed' || step.state === 'running') {
        metaParts.push('<span class="token-count" data-step-id="' + step.step_id + '">Tokens: ' + formatTokens(step.tokens_used) + '</span>');
    }
    if (metaParts.length) {
        bodyParts.push('<div class="step-meta">' + metaParts.join(' ') + '</div>');
    }
    if (step.error) {
        bodyParts.push(
            '<div class="step-error-banner">' +
            '<div class="step-error-header" onclick="event.stopPropagation(); this.parentElement.classList.toggle(\'expanded\')">' +
            '<strong>Error</strong><span class="step-error-toggle">&#9656;</span></div>' +
            '<div class="step-error-content">' + step.error + '</div>' +
            '</div>'
        );
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
            // Preserve collapse state class
            var wasCollapsed = card.classList.contains('step-collapsed');
            card.className = 'step-card status-' + newState;
            if (wasCollapsed) card.classList.add('step-collapsed');
            card.setAttribute('data-step-state', newState);
            // Update badge
            var badge = card.querySelector('.badge');
            if (badge) {
                badge.className = 'badge status-' + newState;
                var spinnerHtml = newState === 'running' ? '<span class="step-running-spinner" aria-hidden="true"></span>' : '';
                badge.innerHTML = spinnerHtml + newState;
            }
            // Auto-expand running/failed steps when they transition
            if (newState === 'running' || newState === 'failed') {
                card.classList.remove('step-collapsed');
                if (typeof expandedSteps !== 'undefined') expandedSteps.add(stepID);
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
