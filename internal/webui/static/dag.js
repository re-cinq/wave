// Wave Dashboard - DAG Interaction JS

document.addEventListener('DOMContentLoaded', function() {
    // Add hover tooltips to DAG nodes
    var nodes = document.querySelectorAll('.dag-node');
    nodes.forEach(function(node) {
        node.addEventListener('mouseenter', function(e) {
            var id = this.getAttribute('data-id');
            var status = this.getAttribute('data-status');
            showTooltip(e, id + ' (' + status + ')');
        });
        node.addEventListener('mouseleave', hideTooltip);
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
    tooltip.textContent = text;
    tooltip.style.display = 'block';
    tooltip.style.left = (e.pageX + 10) + 'px';
    tooltip.style.top = (e.pageY - 30) + 'px';
}

function hideTooltip() {
    if (tooltip) {
        tooltip.style.display = 'none';
    }
}
