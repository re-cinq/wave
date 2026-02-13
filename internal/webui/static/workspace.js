// workspace.js — Workspace file tree browser for Wave dashboard
(function() {
'use strict';

function fetchTree(runId, stepId, path) {
    var url = '/api/runs/' + encodeURIComponent(runId) + '/workspace/' + encodeURIComponent(stepId) + '/tree';
    if (path) url += '?path=' + encodeURIComponent(path);
    return fetch(url).then(function(r) { return r.json(); });
}

function fetchFile(runId, stepId, path) {
    var url = '/api/runs/' + encodeURIComponent(runId) + '/workspace/' + encodeURIComponent(stepId) + '/file?path=' + encodeURIComponent(path);
    return fetch(url).then(function(r) { return r.json(); });
}

function detectLanguage(name) {
    var ext = name.split('.').pop().toLowerCase();
    var map = {
        'go': 'go', 'yaml': 'yaml', 'yml': 'yaml', 'json': 'json',
        'js': 'javascript', 'ts': 'javascript', 'md': 'markdown',
        'sql': 'sql', 'sh': 'shell', 'bash': 'shell',
        'css': 'css', 'html': 'html', 'htm': 'html'
    };
    return map[ext] || '';
}

function renderTree(container, entries, runId, stepId, basePath) {
    var ul = document.createElement('ul');
    ul.className = 'ws-tree';

    for (var i = 0; i < entries.length; i++) {
        var entry = entries[i];
        var li = document.createElement('li');
        li.className = entry.is_dir ? 'ws-dir' : 'ws-file';

        var span = document.createElement('span');
        span.className = 'ws-name';
        span.textContent = (entry.is_dir ? '\u25B6 ' : '\u25CB ') + entry.name;

        var entryPath = basePath ? basePath + '/' + entry.name : entry.name;

        if (entry.is_dir) {
            (function(ep, s) {
                var expanded = false;
                var childContainer = document.createElement('div');
                childContainer.style.display = 'none';
                s.addEventListener('click', function() {
                    if (!expanded) {
                        expanded = true;
                        s.textContent = '\u25BC ' + entry.name;
                        fetchTree(runId, stepId, ep).then(function(data) {
                            if (data.entries && data.entries.length > 0) {
                                renderTree(childContainer, data.entries, runId, stepId, ep);
                            } else {
                                childContainer.textContent = '(empty)';
                            }
                            childContainer.style.display = 'block';
                        });
                    } else {
                        var hidden = childContainer.style.display === 'none';
                        childContainer.style.display = hidden ? 'block' : 'none';
                        s.textContent = (hidden ? '\u25BC ' : '\u25B6 ') + entry.name;
                    }
                });
                li.appendChild(s);
                li.appendChild(childContainer);
            })(entryPath, span);
        } else {
            (function(ep, name) {
                span.addEventListener('click', function() {
                    var viewer = document.querySelector('.ws-file-viewer');
                    if (!viewer) return;
                    viewer.innerHTML = '<p>Loading...</p>';
                    fetchFile(runId, stepId, ep).then(function(data) {
                        if (data.error) {
                            viewer.innerHTML = '<p class="ws-error">' + data.error + '</p>';
                            return;
                        }
                        var lang = detectLanguage(name);
                        var content = data.content;
                        if (window.highlight && lang) {
                            content = window.highlight(data.content, lang);
                        }
                        viewer.innerHTML = '<div class="ws-file-header"><strong>' + name + '</strong> <span class="ws-file-size">' + formatSize(data.size) + '</span>' + (data.truncated ? ' <span class="badge status-warning">truncated</span>' : '') + '</div><pre class="ws-file-content"><code>' + content + '</code></pre>';
                    });
                });
                li.appendChild(span);
            })(entryPath, entry.name);
        }

        ul.appendChild(li);
    }
    container.appendChild(ul);
}

function formatSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
}

function initWorkspaceBrowser(container, runId, stepId) {
    fetchTree(runId, stepId, '').then(function(data) {
        if (data.error) {
            container.innerHTML = '<p class="ws-error">' + data.error + '</p>';
            return;
        }
        var treePane = document.createElement('div');
        treePane.className = 'ws-tree-pane';
        var filePane = document.createElement('div');
        filePane.className = 'ws-file-viewer';
        filePane.innerHTML = '<p class="ws-placeholder">Select a file to view its content</p>';
        container.innerHTML = '';
        container.appendChild(treePane);
        container.appendChild(filePane);
        if (data.entries && data.entries.length > 0) {
            renderTree(treePane, data.entries, runId, stepId, '');
        } else {
            treePane.innerHTML = '<p>(empty workspace)</p>';
        }
    });
}

window.initWorkspaceBrowser = initWorkspaceBrowser;
})();
