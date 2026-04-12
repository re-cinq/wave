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

// formatDuration formats milliseconds into human-friendly duration strings
// matching the Go formatDurationValue output (Xs, Xm Ys, Xh Ym).
function formatDuration(ms) {
    if (ms == null || ms === undefined) return '';
    var totalSec = Math.round(ms / 1000);
    if (totalSec < 60) return totalSec + 's';
    var totalMin = Math.floor(totalSec / 60);
    var sec = totalSec % 60;
    if (totalMin < 60) {
        return sec === 0 ? totalMin + 'm' : totalMin + 'm ' + sec + 's';
    }
    var hours = Math.floor(totalMin / 60);
    var min = totalMin % 60;
    return min === 0 ? hours + 'h' : hours + 'h ' + min + 'm';
}

// formatStartTime formats an ISO timestamp to a short time string (HH:MM:SS).
function formatStartTime(isoString) {
    if (!isoString) return '';
    try {
        var d = new Date(isoString);
        return d.toLocaleTimeString([], {hour: '2-digit', minute: '2-digit', second: '2-digit'});
    } catch (e) {
        return '';
    }
}

// Track expanded step IDs for persistence across SSE updates
var expandedSteps = new Set();

function toggleStepCard(stepID) {
    var card = document.getElementById('step-' + stepID);
    if (!card) return;
    var isExpanded = card.getAttribute('data-expanded') === 'true';
    var nowExpanded = !isExpanded;
    card.setAttribute('data-expanded', nowExpanded ? 'true' : 'false');
    // Update aria-expanded on the header button
    var header = card.querySelector('.step-header');
    if (header) header.setAttribute('aria-expanded', nowExpanded ? 'true' : 'false');
    if (isExpanded) {
        expandedSteps.delete(stepID);
    } else {
        expandedSteps.add(stepID);
    }
}

function toggleErrorExpand(stepID) {
    var banner = document.getElementById('error-' + stepID);
    if (!banner) return;
    var toggle = banner.nextElementSibling;
    if (banner.classList.contains('collapsed')) {
        banner.classList.remove('collapsed');
        if (toggle) toggle.textContent = 'Show less';
    } else {
        banner.classList.add('collapsed');
        if (toggle) toggle.textContent = 'Show more';
    }
}

// Initialize collapse states on page load
function initStepCollapseStates() {
    var cards = document.querySelectorAll('.step-card');
    cards.forEach(function(card) {
        var expanded = card.getAttribute('data-expanded') === 'true';
        var stepID = card.id.replace('step-', '');
        if (expanded) {
            expandedSteps.add(stepID);
        }
    });
}

var sseConnection = null;
var pollTimer = null;
var currentRunID = null;
var sseHasConnected = false;

function showConnectionBanner(visible) {
    var banner = document.getElementById('connection-banner');
    if (!banner) return;
    banner.hidden = !visible;
}

function connectSSE(runID) {
    currentRunID = runID;

    if (sseConnection) {
        sseConnection.close();
    }

    var url = '/api/runs/' + encodeURIComponent(runID) + '/events';
    sseConnection = new EventSource(url);

    sseConnection.onopen = function() {
        console.log('SSE connected for run:', runID);
        sseHasConnected = true;
        showConnectionBanner(false);
        if (window.logViewer) {
            window.logViewer.onReconnect();
        }
    };

    sseConnection.onerror = function() {
        console.log('SSE error, falling back to polling...');
        // Only show "connection lost" if we had a working connection before
        // AND the run is still in a running state (not completed/failed)
        if (sseHasConnected && !document.querySelector('.badge.status-completed, .badge.status-failed')) {
            showConnectionBanner(true);
        }
        if (window.logViewer) {
            window.logViewer.onDisconnect();
        }
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
    var stepsList = document.querySelector('.steps-timeline, .steps-list');
    if (stepsList && data.steps && data.steps.length > 0) {
        stepsList.innerHTML = '';
        data.steps.forEach(function(step) {
            stepsList.appendChild(createStepCard(step));
        });
        if (window.logViewer) window.logViewer.reattach();
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
            var stepHtml = ev.step_id ? '<span class="event-step">' + escapeHTML(ev.step_id) + '</span>' : '';
            item.innerHTML =
                '<span class="event-time">' + t + '</span>' +
                '<span class="badge status-' + escapeHTML(ev.state) + '">' + escapeHTML(ev.state) + '</span>' +
                stepHtml +
                '<span class="event-message">' + escapeHTML(ev.message || '') + '</span>';
            timeline.appendChild(item);
        });
    }
}

function createStepCard(step) {
    var card = document.createElement('div');
    card.className = 'step-card status-' + step.state;
    card.id = 'step-' + step.step_id;

    // Preserve expand state or default based on status
    var isExpanded = expandedSteps.has(step.step_id) ||
        step.state === 'running' || step.state === 'failed';
    card.setAttribute('data-expanded', isExpanded ? 'true' : 'false');
    if (isExpanded) expandedSteps.add(step.step_id);

    var statusIconHtml = '';
    if (step.state === 'running') {
        statusIconHtml = '<span class="step-status-icon step-status-running" aria-label="Running"></span>';
    } else if (step.state === 'completed') {
        statusIconHtml = '<span class="step-status-icon step-status-completed" aria-label="Completed">&#10003;</span>';
    } else if (step.state === 'failed') {
        statusIconHtml = '<span class="step-status-icon step-status-failed" aria-label="Failed">&#10007;</span>';
    } else {
        statusIconHtml = '<span class="step-status-icon step-status-pending" aria-label="Pending">&#9679;</span>';
    }
    var startTimeHtml = step.started_at ? '<span class="step-start-time">' + formatStartTime(step.started_at) + '</span>' : '';
    var durationHtml = '<span class="step-duration"' + (step.state === 'running' ? ' data-live-timer="true"' : '') + '>' + (step.duration || '') + '</span>';

    var safeStepId = escapeHTML(step.step_id);
    var headerHtml =
        '<div class="step-header" data-step-id="' + safeStepId + '" onclick="toggleStepCard(this.getAttribute(\'data-step-id\'))">' +
        '<span class="step-chevron" aria-hidden="true"></span>' +
        statusIconHtml +
        '<span class="step-id">' + safeStepId + '</span>' +
        '<span class="badge status-' + escapeHTML(step.state) + '">' + escapeHTML(step.state) + '</span>' +
        startTimeHtml +
        durationHtml +
        '<span class="step-persona">' + escapeHTML(step.persona || '') + '</span>' +
        (step.model ? '<span class="badge badge-model">' + escapeHTML(step.model) + '</span>' : '') +
        (step.adapter ? '<span class="badge badge-adapter">' + escapeHTML(step.adapter) + '</span>' : '') +
        '<button class="btn-icon" data-step-id="' + safeStepId + '" onclick="event.stopPropagation(); if(window.logViewer) window.logViewer.downloadLog(this.getAttribute(\'data-step-id\'))" title="Download log">Save</button>' +
        '<button class="btn-icon" data-step-id="' + safeStepId + '" onclick="event.stopPropagation(); if(window.logViewer) window.logViewer.copyLog(this.getAttribute(\'data-step-id\'))" title="Copy log">Copy</button>' +
        '</div>';

    var bodyParts = [];
    if (step.current_action) {
        bodyParts.push('<div class="step-action">' + escapeHTML(step.current_action) + '</div>');
    }
    var metaParts = [];
    if (step.duration) metaParts.push('<span>Duration: ' + step.duration + '</span>');
    metaParts.push('<span class="token-count" data-step-id="' + safeStepId + '">Tokens: ' + formatTokens(step.tokens_used) + '</span>');
    bodyParts.push('<div class="step-meta">' + metaParts.join(' ') + '</div>');
    if (step.error) {
        var isLong = step.error.length > 200;
        bodyParts.push('<div class="step-error-banner' + (isLong ? ' collapsed' : '') + '" id="error-' + safeStepId + '">' + escapeHTML(step.error) + '</div>');
        if (isLong) {
            bodyParts.push('<span class="step-error-toggle" data-step-id="' + safeStepId + '" onclick="toggleErrorExpand(this.getAttribute(\'data-step-id\'))">Show more</span>');
        }
    }
    if (step.artifacts && step.artifacts.length > 0) {
        var artItems = step.artifacts.map(function(art) {
            return '<li>' +
                '<a href="#" class="artifact-link"' +
                ' data-run-id="' + escapeHTML(step.run_id) + '"' +
                ' data-step-id="' + safeStepId + '"' +
                ' data-artifact-name="' + escapeHTML(art.name) + '"' +
                ' onclick="toggleArtifact(this); return false;"' +
                ' aria-expanded="false">' + escapeHTML(art.name) + '</a>' +
                '<span class="artifact-size">(' + (art.size_bytes || 0) + ' bytes)</span>' +
                '<div class="artifact-inline" hidden></div>' +
                '</li>';
        }).join('');
        bodyParts.push('<div class="step-artifacts"><strong>Artifacts:</strong><ul>' + artItems + '</ul></div>');
    }

    var logHtml = '<div class="step-log" id="log-' + safeStepId + '" data-step-id="' + safeStepId + '">' +
        '<div class="step-log-content"></div>' +
        '</div>';
    card.innerHTML = headerHtml + '<div class="step-body">' + bodyParts.join('') + logHtml + '</div>';
    // Apply collapsed class for non-running/non-failed steps
    if (step.state !== 'running' && step.state !== 'failed') {
        card.classList.add('step-collapsed');
    }
    return card;
}

function handleSSEEvent(type, data) {
    // Route stream_activity events to the log viewer
    if (type === 'stream_activity' && window.logViewer) {
        window.logViewer.addLine(data.step_id, data);
    }

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
        if (window.logViewer) window.logViewer.onStepStateChange(data.step_id, 'running');
    }
    if (data.step_id && type === 'completed') {
        updateStepCardState(data.step_id, 'completed');
        if (window.logViewer) window.logViewer.onStepStateChange(data.step_id, 'completed');
    }
    if (data.step_id && type === 'failed') {
        updateStepCardState(data.step_id, 'failed');
        if (window.logViewer) window.logViewer.onStepStateChange(data.step_id, 'failed');
    }

    // Update step card tokens in real-time
    if (data.step_id && data.tokens_used !== undefined) {
        updateStepCardTokens(data.step_id, data.tokens_used);
    }

    // Append meaningful events to the timeline (skip empty heartbeats)
    if (data.message && type !== 'step_progress' && type !== 'stream_activity') {
        appendEventToTimeline(data, type);
    }

    // Refresh diff viewer on step completion during running pipelines
    if (type === 'completed' && data.step_id && window.diffViewer) {
        window.diffViewer.refreshIfNeeded();
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
        var icon = node.querySelector('.dag-status-icon');
        if (icon) {
            icon.setAttribute('class', 'dag-status-icon ' + cssStatus);
        }
    }

    // Refresh detail overlay if one is open for this node
    var overlay = document.querySelector('.dag-detail-overlay[data-for="' + stepID + '"]');
    if (overlay) {
        var badge = overlay.querySelector('.badge');
        if (badge) {
            badge.className = 'badge status-' + cssStatus;
            badge.textContent = cssStatus;
        }
        var durationText = node ? (node.getAttribute('data-duration') || '') : '';
        var rows = overlay.querySelectorAll('.detail-row');
        for (var i = 0; i < rows.length; i++) {
            if (rows[i].textContent.indexOf('Duration:') === 0) {
                if (durationText) {
                    rows[i].textContent = 'Duration: ' + durationText;
                }
                break;
            }
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
    // V2 layout: step wrapper is .ws with id="w-{stepID}"
    var wsEl = document.getElementById('w-' + stepID);
    if (wsEl) {
        // Update state class (st-running -> st-completed etc.)
        wsEl.className = wsEl.className.replace(/\bst-\w+/g, '') + ' st-' + newState;
        // Remove spinner on non-running states
        if (newState !== 'running') {
            var spinner = wsEl.querySelector('.spinner');
            if (spinner) spinner.remove();
        }
        // Add spinner if transitioning to running
        if (newState === 'running' && !wsEl.querySelector('.spinner')) {
            var nameEl = wsEl.querySelector('.ws-name');
            if (nameEl) {
                var sp = document.createElement('span');
                sp.className = 'spinner spinner-sm';
                sp.style.verticalAlign = 'middle';
                sp.style.marginLeft = '0.35rem';
                nameEl.appendChild(sp);
            }
        }
        return;
    }

    // V1 fallback: step wrapper is .step-card with .step-id text match
    var cards = document.querySelectorAll('.step-card');
    cards.forEach(function(card) {
        var idEl = card.querySelector('.step-id');
        if (idEl && idEl.textContent.trim() === stepID) {
            card.classList.remove('status-pending','status-running','status-completed','status-failed','status-cancelled');
            card.classList.add('status-' + newState);
            var badge = card.querySelector('.badge');
            if (badge) {
                badge.classList.remove('status-pending','status-running','status-completed','status-failed','status-cancelled');
                badge.classList.add('status-' + newState);
                badge.textContent = newState;
            }
            var icon = card.querySelector('.step-status-icon');
            if (icon) {
                icon.className = 'step-status-icon step-status-' + newState;
                if (newState === 'completed') icon.innerHTML = '&#10003;';
                else if (newState === 'failed') icon.innerHTML = '&#10007;';
                else if (newState === 'running') icon.innerHTML = '';
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
    var stepHtml = data.step_id ? '<span class="event-step">' + escapeHTML(data.step_id) + '</span>' : '';

    item.innerHTML =
        '<span class="event-time">' + time + '</span>' +
        '<span class="badge status-' + escapeHTML(type) + '">' + escapeHTML(type) + '</span>' +
        stepHtml +
        '<span class="event-message">' + escapeHTML(data.message || '') + '</span>';

    timeline.appendChild(item);
}

// Disconnect when leaving page
window.addEventListener('beforeunload', function() {
    stopPolling();
    if (sseConnection) {
        sseConnection.close();
    }
});
