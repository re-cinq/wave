// introspect.js — Step drill-down and event timeline for Wave dashboard
(function() {
'use strict';

document.addEventListener('DOMContentLoaded', function() {
    // Step drill-down toggle
    var stepHeaders = document.querySelectorAll('.step-header[data-toggle]');
    for (var i = 0; i < stepHeaders.length; i++) {
        stepHeaders[i].addEventListener('click', function() {
            var target = this.getAttribute('data-toggle');
            var panel = document.getElementById(target);
            if (panel) {
                var isHidden = panel.style.display === 'none' || panel.style.display === '';
                panel.style.display = isHidden ? 'block' : 'none';
                this.classList.toggle('expanded', isHidden);
            }
        });
    }

    // Event timeline scroll-to-step
    var eventSteps = document.querySelectorAll('.event-step[data-step]');
    for (var j = 0; j < eventSteps.length; j++) {
        eventSteps[j].addEventListener('click', function() {
            var stepId = this.getAttribute('data-step');
            var stepCard = document.getElementById('step-' + stepId);
            if (stepCard) {
                stepCard.scrollIntoView({ behavior: 'smooth', block: 'center' });
                stepCard.classList.add('highlight');
                setTimeout(function() { stepCard.classList.remove('highlight'); }, 2000);
            }
        });
    }

    // Raw/rendered toggle for markdown viewer
    var toggleBtns = document.querySelectorAll('.md-toggle');
    for (var k = 0; k < toggleBtns.length; k++) {
        toggleBtns[k].addEventListener('click', function() {
            var viewer = this.closest('.md-viewer');
            if (!viewer) return;
            var rendered = viewer.querySelector('.md-rendered');
            var raw = viewer.querySelector('.md-raw');
            if (!rendered || !raw) return;
            var showRaw = this.getAttribute('data-view') === 'raw';
            rendered.style.display = showRaw ? 'none' : 'block';
            raw.style.display = showRaw ? 'block' : 'none';
            // Update active button
            var btns = viewer.querySelectorAll('.md-toggle');
            for (var b = 0; b < btns.length; b++) {
                btns[b].classList.toggle('active', btns[b] === this);
            }
        });
    }

    // Raw/formatted toggle for code viewer
    var codeBtns = document.querySelectorAll('.code-toggle');
    for (var l = 0; l < codeBtns.length; l++) {
        codeBtns[l].addEventListener('click', function() {
            var viewer = this.closest('.code-viewer');
            if (!viewer) return;
            var highlighted = viewer.querySelector('.code-highlighted');
            var raw = viewer.querySelector('.code-raw');
            if (!highlighted || !raw) return;
            var showRaw = this.getAttribute('data-view') === 'raw';
            highlighted.style.display = showRaw ? 'none' : 'block';
            raw.style.display = showRaw ? 'block' : 'none';
            var btns = viewer.querySelectorAll('.code-toggle');
            for (var b = 0; b < btns.length; b++) {
                btns[b].classList.toggle('active', btns[b] === this);
            }
        });
    }

    // Init markdown rendering for elements with data-markdown attribute
    var mdElements = document.querySelectorAll('[data-markdown]');
    for (var m = 0; m < mdElements.length; m++) {
        var el = mdElements[m];
        var rawText = el.textContent;
        var rendered = document.createElement('div');
        rendered.className = 'md-rendered';
        rendered.innerHTML = window.renderMarkdown ? window.renderMarkdown(rawText) : rawText;
        el.parentNode.insertBefore(rendered, el);
    }

    // Init syntax highlighting for elements with data-language attribute
    var codeElements = document.querySelectorAll('[data-language]');
    for (var n = 0; n < codeElements.length; n++) {
        var codeEl = codeElements[n];
        var lang = codeEl.getAttribute('data-language');
        var codeText = codeEl.textContent;
        if (window.highlight) {
            codeEl.innerHTML = window.highlight(codeText, lang);
        }
    }
});
})();
