// Wave Dashboard - DAG Interaction JS

document.addEventListener('DOMContentLoaded', function() {
    var nodes = document.querySelectorAll('.dag-node');
    nodes.forEach(function(node) {
        node.addEventListener('mouseenter', function(e) {
            var id = this.getAttribute('data-id');
            var status = this.getAttribute('data-status');
            var duration = this.getAttribute('data-duration');
            var tokens = this.getAttribute('data-tokens');

            var lines = [id, 'Status: ' + status];
            if (duration) {
                lines.push('Duration: ' + duration);
            }
            if (tokens && tokens !== '0') {
                lines.push('Tokens: ' + tokens);
            }
            showTooltip(e, lines.join('\n'));
        });
        node.addEventListener('mouseleave', hideTooltip);
        node.addEventListener('click', function() {
            var id = this.getAttribute('data-id');
            scrollToStep(id);
        });
        node.addEventListener('keydown', function(e) {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                var id = this.getAttribute('data-id');
                scrollToStep(id);
            }
        });
        node.style.cursor = 'pointer';
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
