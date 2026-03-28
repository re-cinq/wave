// Wave Dashboard - DAG Interaction JS

// Thread color palette — deterministic color per thread name.
var threadColorPalette = [
    '#22d3ee', '#a78bfa', '#f472b6', '#34d399', '#fbbf24',
    '#fb923c', '#60a5fa', '#c084fc', '#f87171', '#4ade80'
];
var threadColorMap = {};

function getThreadColor(threadName) {
    if (!threadName) return '';
    if (threadColorMap[threadName]) return threadColorMap[threadName];
    var idx = Object.keys(threadColorMap).length % threadColorPalette.length;
    threadColorMap[threadName] = threadColorPalette[idx];
    return threadColorMap[threadName];
}

// Apply thread colors to SVG thread bars on load.
function applyThreadColors() {
    var bars = document.querySelectorAll('.dag-thread-bar');
    bars.forEach(function(bar) {
        var name = bar.getAttribute('data-thread-name');
        if (name) {
            bar.style.fill = getThreadColor(name);
        }
    });
}

document.addEventListener('DOMContentLoaded', function() {
    applyThreadColors();

    var nodes = document.querySelectorAll('.dag-node');
    nodes.forEach(function(node) {
        node.addEventListener('mouseenter', function(e) {
            var id = this.getAttribute('data-id');
            var status = this.getAttribute('data-status');
            var duration = this.getAttribute('data-duration');
            var tokens = this.getAttribute('data-tokens');
            var stepType = this.getAttribute('data-step-type');
            var thread = this.getAttribute('data-thread');

            var lines = [id];
            if (stepType) {
                lines[0] += ' [' + stepType + ']';
            }
            lines.push('Status: ' + status);
            if (duration) {
                lines.push('Duration: ' + duration);
            }
            if (tokens && tokens !== '0') {
                lines.push('Tokens: ' + tokens);
            }
            if (thread) {
                lines.push('Thread: ' + thread);
            }
            showTooltip(e, lines.join('\n'));
        });
        node.addEventListener('mouseleave', hideTooltip);
        node.addEventListener('click', function() {
            toggleDetailOverlay(this);
        });
        node.addEventListener('keydown', function(e) {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                toggleDetailOverlay(this);
            }
        });
        node.style.cursor = 'pointer';
    });

    // Close overlay when clicking outside of it and outside dag nodes
    document.addEventListener('click', function(e) {
        var overlay = document.querySelector('.dag-detail-overlay');
        if (!overlay) return;
        // If the click is inside the overlay or on a dag node, do nothing
        if (overlay.contains(e.target) || e.target.closest('.dag-node')) return;
        overlay.remove();
    });
});

var tooltip = null;

function showTooltip(e, text) {
    if (!tooltip) {
        tooltip = document.createElement('div');
        tooltip.className = 'dag-tooltip';
        document.body.appendChild(tooltip);
    }
    tooltip.textContent = '';
    var lines = text.split('\n');
    for (var i = 0; i < lines.length; i++) {
        if (i > 0) {
            tooltip.appendChild(document.createElement('br'));
        }
        tooltip.appendChild(document.createTextNode(lines[i]));
    }
    tooltip.style.display = 'block';
    tooltip.style.left = (e.pageX + 10) + 'px';
    tooltip.style.top = (e.pageY - 30) + 'px';
}

function hideTooltip() {
    if (tooltip) {
        tooltip.style.display = 'none';
    }
}

// Extract X,Y from an SVG transform="translate(X, Y)" attribute.
function parseTranslate(node) {
    var transform = node.getAttribute('transform') || '';
    var match = transform.match(/translate\(\s*([\d.]+)\s*,\s*([\d.]+)\s*\)/);
    if (match) {
        return { x: parseFloat(match[1]), y: parseFloat(match[2]) };
    }
    return { x: 0, y: 0 };
}

// Compute the scale factor between SVG viewBox coordinates and rendered pixels.
function getSVGScale(svgEl) {
    var viewBox = svgEl.viewBox.baseVal;
    if (!viewBox || viewBox.width === 0) return 1;
    var rect = svgEl.getBoundingClientRect();
    return rect.width / viewBox.width;
}

// Step type display labels and colors
var stepTypeConfig = {
    'gate':        { label: 'Gate',        color: '#f59e0b' },
    'command':     { label: 'Command',     color: '#6b7280' },
    'conditional': { label: 'Conditional', color: '#a78bfa' },
    'pipeline':    { label: 'Pipeline',    color: '#3b82f6' }
};

// Toggle the detail overlay for a given DAG node <g> element.
function toggleDetailOverlay(node) {
    var id = node.getAttribute('data-id');
    var existing = document.querySelector('.dag-detail-overlay');

    // If an overlay already exists for this node, remove it (toggle off)
    if (existing && existing.getAttribute('data-for') === id) {
        existing.remove();
        return;
    }

    // Remove any existing overlay for a different node
    if (existing) {
        existing.remove();
    }

    var container = node.closest('.dag-container');
    if (!container) return;

    // Ensure the container is a positioning context for absolute children
    if (getComputedStyle(container).position === 'static') {
        container.style.position = 'relative';
    }

    var svgEl = container.querySelector('.dag-svg');
    if (!svgEl) return;

    var status = node.getAttribute('data-status') || '';
    var persona = node.getAttribute('data-persona') || '';
    var duration = node.getAttribute('data-duration') || '';
    var tokens = node.getAttribute('data-tokens') || '';
    var stepType = node.getAttribute('data-step-type') || '';
    var script = node.getAttribute('data-script') || '';
    var subPipeline = node.getAttribute('data-sub-pipeline') || '';
    var gatePrompt = node.getAttribute('data-gate-prompt') || '';
    var gateChoices = node.getAttribute('data-gate-choices') || '';
    var edgeInfo = node.getAttribute('data-edge-info') || '';
    var thread = node.getAttribute('data-thread') || '';

    var pos = parseTranslate(node);
    var scale = getSVGScale(svgEl);

    // Position the overlay to the right of the node (node width 140 + 8px gap)
    var overlayX = (pos.x + 140 + 8) * scale;
    var overlayY = pos.y * scale;

    // Build the overlay element
    var overlay = document.createElement('div');
    overlay.className = 'dag-detail-overlay';
    overlay.setAttribute('data-for', id);
    overlay.style.left = overlayX + 'px';
    overlay.style.top = overlayY + 'px';

    // Header: step ID
    var header = document.createElement('div');
    header.className = 'detail-header';
    header.textContent = id;
    overlay.appendChild(header);

    // Step type badge
    if (stepType && stepTypeConfig[stepType]) {
        var typeBadge = document.createElement('span');
        typeBadge.className = 'detail-type-badge';
        typeBadge.style.color = stepTypeConfig[stepType].color;
        typeBadge.style.borderColor = stepTypeConfig[stepType].color;
        typeBadge.textContent = stepTypeConfig[stepType].label;
        header.appendChild(document.createTextNode(' '));
        header.appendChild(typeBadge);
    }

    // Status row with badge
    var statusRow = document.createElement('div');
    statusRow.className = 'detail-row';
    var badge = document.createElement('span');
    badge.className = 'badge status-' + status;
    badge.textContent = status;
    statusRow.appendChild(document.createTextNode('Status: '));
    statusRow.appendChild(badge);
    overlay.appendChild(statusRow);

    // Persona row (skip for command/conditional steps)
    if (persona && stepType !== 'command' && stepType !== 'conditional') {
        var personaRow = document.createElement('div');
        personaRow.className = 'detail-row';
        personaRow.textContent = 'Persona: ' + persona;
        overlay.appendChild(personaRow);
    }

    // Duration row
    if (duration) {
        var durationRow = document.createElement('div');
        durationRow.className = 'detail-row';
        durationRow.textContent = 'Duration: ' + duration;
        overlay.appendChild(durationRow);
    }

    // Tokens row
    if (tokens && tokens !== '0') {
        var tokensRow = document.createElement('div');
        tokensRow.className = 'detail-row';
        tokensRow.textContent = 'Tokens: ' + tokens;
        overlay.appendChild(tokensRow);
    }

    // Type-specific details
    if (stepType === 'gate') {
        if (gatePrompt) {
            var promptRow = document.createElement('div');
            promptRow.className = 'detail-row detail-type-info';
            promptRow.textContent = 'Prompt: ' + gatePrompt;
            overlay.appendChild(promptRow);
        }
        if (gateChoices) {
            var choicesRow = document.createElement('div');
            choicesRow.className = 'detail-row detail-type-info';
            choicesRow.style.color = '#f59e0b';
            choicesRow.textContent = 'Choices: ' + gateChoices;
            overlay.appendChild(choicesRow);
        }
    }

    if (stepType === 'command' && script) {
        var scriptRow = document.createElement('div');
        scriptRow.className = 'detail-row detail-type-info';
        var scriptCode = document.createElement('code');
        scriptCode.style.fontSize = '0.7rem';
        var displayScript = script.length > 60 ? script.substring(0, 60) + '...' : script;
        scriptCode.textContent = displayScript;
        scriptRow.appendChild(document.createTextNode('Script: '));
        scriptRow.appendChild(scriptCode);
        overlay.appendChild(scriptRow);
    }

    if (stepType === 'conditional' && edgeInfo) {
        var edgeRow = document.createElement('div');
        edgeRow.className = 'detail-row detail-type-info';
        edgeRow.style.color = '#a78bfa';
        edgeRow.textContent = 'Edges: ' + edgeInfo;
        overlay.appendChild(edgeRow);
    }

    if (stepType === 'pipeline' && subPipeline) {
        var pipelineRow = document.createElement('div');
        pipelineRow.className = 'detail-row detail-type-info';
        var pipelineLink = document.createElement('a');
        pipelineLink.href = '/pipelines/' + subPipeline;
        pipelineLink.textContent = subPipeline;
        pipelineLink.style.color = '#60a5fa';
        pipelineRow.appendChild(document.createTextNode('Pipeline: '));
        pipelineRow.appendChild(pipelineLink);
        overlay.appendChild(pipelineRow);
    }

    // Thread group row
    if (thread) {
        var threadRow = document.createElement('div');
        threadRow.className = 'detail-row detail-type-info';
        var threadBadge = document.createElement('span');
        threadBadge.className = 'detail-thread-badge';
        threadBadge.style.borderColor = getThreadColor(thread);
        threadBadge.style.color = getThreadColor(thread);
        threadBadge.textContent = thread;
        threadRow.appendChild(document.createTextNode('Thread: '));
        threadRow.appendChild(threadBadge);
        overlay.appendChild(threadRow);
    }

    // "Go to step" link
    var link = document.createElement('a');
    link.className = 'detail-link';
    link.href = '#';
    link.textContent = 'Go to step \u2192';
    link.addEventListener('click', function(e) {
        e.preventDefault();
        overlay.remove();
        scrollToStep(id);
    });
    overlay.appendChild(link);

    container.appendChild(overlay);
}

function scrollToStep(stepID) {
    var card = document.getElementById('step-' + stepID);
    if (!card) {
        // Fallback: search by step-id text content
        var cards = document.querySelectorAll('.step-card');
        for (var i = 0; i < cards.length; i++) {
            var idEl = cards[i].querySelector('.step-id');
            if (idEl && idEl.textContent.trim() === stepID) {
                card = cards[i];
                break;
            }
        }
    }
    if (card) {
        card.scrollIntoView({ behavior: 'smooth', block: 'center' });
        card.style.outline = '2px solid var(--wave-primary)';
        card.style.outlineOffset = '2px';
        setTimeout(function(c) {
            c.style.outline = '';
            c.style.outlineOffset = '';
        }, 2000, card);
    }
}
