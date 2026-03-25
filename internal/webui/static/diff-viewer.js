// Wave Dashboard - Diff Viewer
// Provides changed-file browsing and diff rendering for pipeline runs.

(function() {
    'use strict';

    var LINE_HEIGHT = 20; // px, fixed for virtualization
    var VIRTUAL_BUFFER = 50; // lines above/below viewport
    var VIRTUAL_THRESHOLD = 500; // activate virtualization above this line count
    var VIEW_MODE_KEY = 'wave-diff-view-mode';

    // --- DiffViewer class ---

    function DiffViewer(runID, container) {
        this.runID = runID;
        this.container = container;
        this.summary = null;
        this.currentFile = null;
        this.currentFileDiff = null;
        this.currentParsed = null;
        this.viewMode = localStorage.getItem(VIEW_MODE_KEY) || 'unified';
        this.rawSubMode = 'after';
        this._lastRefresh = 0;
    }

    // --- File List Loading (T009, T018, T019) ---

    DiffViewer.prototype.loadFileList = function() {
        var self = this;
        var url = '/api/runs/' + encodeURIComponent(this.runID) + '/diff';

        fetch(url)
            .then(function(resp) {
                if (!resp.ok) {
                    return resp.json().then(function(e) { throw new Error(e.error || resp.statusText); });
                }
                return resp.json();
            })
            .then(function(summary) {
                self.summary = summary;
                self.render();
            })
            .catch(function(err) {
                self.renderError(String(err));
            });
    };

    // --- Main Render ---

    DiffViewer.prototype.render = function() {
        var summary = this.summary;
        if (!summary) return;

        // Handle unavailable diff (T019)
        if (!summary.available) {
            this.container.innerHTML =
                '<h2>Changed Files</h2>' +
                '<div class="diff-unavailable">' + escapeHTML(summary.message || 'Diff unavailable') + '</div>';
            return;
        }

        // Handle empty file list (T019)
        if (!summary.files || summary.files.length === 0) {
            this.container.innerHTML =
                '<h2>Changed Files</h2>' +
                '<div class="diff-unavailable">No files changed</div>';
            return;
        }

        var html = '<h2>Changed Files</h2>';

        // Summary bar (T018)
        html += '<div class="diff-summary-bar">' +
            '<span class="diff-summary-files">' + summary.total_files + ' file' + (summary.total_files !== 1 ? 's' : '') + ' changed</span>' +
            '<span class="diff-summary-adds">+' + summary.total_additions + '</span>' +
            '<span class="diff-summary-dels">-' + summary.total_deletions + '</span>' +
            '</div>';

        // Layout: file list + diff content
        html += '<div class="diff-layout">';

        // File list panel
        html += '<div class="diff-file-list" id="diff-file-list">';
        for (var i = 0; i < summary.files.length; i++) {
            var f = summary.files[i];
            var statusChar = f.status === 'added' ? 'A' : f.status === 'deleted' ? 'D' : f.status === 'renamed' ? 'R' : 'M';
            var statusClass = 'diff-status-' + statusChar;
            var selected = this.currentFile === f.path ? ' diff-file-selected' : '';
            var counts = '';
            if (!f.binary) {
                if (f.additions > 0) counts += '<span class="diff-file-adds">+' + f.additions + '</span>';
                if (f.deletions > 0) counts += '<span class="diff-file-dels">-' + f.deletions + '</span>';
            } else {
                counts = '<span class="diff-file-binary">binary</span>';
            }
            html += '<div class="diff-file-item' + selected + '" data-path="' + escapeHTML(f.path) + '">' +
                '<span class="diff-file-status ' + statusClass + '">' + statusChar + '</span>' +
                '<span class="diff-file-path">' + escapeHTML(f.path) + '</span>' +
                counts +
                '</div>';
        }
        html += '</div>';

        // Diff content panel
        html += '<div class="diff-content" id="diff-viewer-content">';
        if (this.currentFileDiff) {
            html += this.renderDiffContent();
        } else {
            html += '<div class="diff-placeholder">Select a file to view its diff</div>';
        }
        html += '</div>';

        html += '</div>'; // .diff-layout

        this.container.innerHTML = html;

        // Bind click handlers
        var self = this;
        var items = this.container.querySelectorAll('.diff-file-item');
        for (var j = 0; j < items.length; j++) {
            items[j].addEventListener('click', function() {
                var path = this.getAttribute('data-path');
                self.loadFileDiff(path);
            });
        }

        // Bind toggle handlers if present
        this.bindToggleHandlers();

        // Setup virtualization if active
        if (this.currentParsed && this.needsVirtualization()) {
            this.setupVirtualization();
        }
    };

    // --- File Diff Loading (T009) ---

    DiffViewer.prototype.loadFileDiff = function(path) {
        var self = this;
        this.currentFile = path;
        var url = '/api/runs/' + encodeURIComponent(this.runID) + '/diff/' +
            path.split('/').map(encodeURIComponent).join('/');

        fetch(url)
            .then(function(resp) {
                if (!resp.ok) {
                    return resp.json().then(function(e) { throw new Error(e.error || resp.statusText); });
                }
                return resp.json();
            })
            .then(function(fileDiff) {
                self.currentFileDiff = fileDiff;
                self.currentParsed = fileDiff.content ? parseDiff(fileDiff.content) : null;
                self.render();
            })
            .catch(function(err) {
                self.currentFileDiff = null;
                self.currentParsed = null;
                self.render();
                var content = document.getElementById('diff-viewer-content');
                if (content) {
                    content.innerHTML = '<div class="diff-error">Failed to load diff: ' + escapeHTML(String(err)) + '</div>';
                }
            });
    };

    // --- Diff Content Rendering ---

    DiffViewer.prototype.renderDiffContent = function() {
        var fd = this.currentFileDiff;
        if (!fd) return '';

        var html = '';

        // View mode toggle (T015)
        html += '<div class="diff-view-toggle">' +
            '<button class="diff-toggle-btn' + (this.viewMode === 'unified' ? ' diff-toggle-active' : '') + '" data-mode="unified">Unified</button>' +
            '<button class="diff-toggle-btn' + (this.viewMode === 'side-by-side' ? ' diff-toggle-active' : '') + '" data-mode="side-by-side">Side-by-side</button>' +
            '<button class="diff-toggle-btn' + (this.viewMode === 'raw' ? ' diff-toggle-active' : '') + '" data-mode="raw">Raw</button>' +
            '</div>';

        // Binary file (T019)
        if (fd.binary) {
            html += '<div class="diff-binary-notice">Binary file changed</div>';
            return html;
        }

        // Truncated file (T019)
        if (fd.truncated) {
            html += '<div class="diff-truncated-notice">Truncated — file too large (' + fd.size + ' bytes)</div>';
        }

        if (!fd.content) {
            html += '<div class="diff-placeholder">No diff content</div>';
            return html;
        }

        var parsed = this.currentParsed;
        if (!parsed) return html;

        // Raw sub-mode toggle
        if (this.viewMode === 'raw') {
            html += '<div class="diff-raw-toggle">' +
                '<button class="diff-toggle-btn' + (this.rawSubMode === 'before' ? ' diff-toggle-active' : '') + '" data-raw="before">Before</button>' +
                '<button class="diff-toggle-btn' + (this.rawSubMode === 'after' ? ' diff-toggle-active' : '') + '" data-raw="after">After</button>' +
                '</div>';
        }

        var ext = getFileExtension(fd.path);

        // Render based on view mode
        if (this.viewMode === 'side-by-side') {
            html += renderSideBySide(parsed, ext);
        } else if (this.viewMode === 'raw') {
            html += renderRaw(parsed, this.rawSubMode, ext);
        } else {
            html += renderUnified(parsed, ext);
        }

        return html;
    };

    // --- Toggle Handlers (T015) ---

    DiffViewer.prototype.bindToggleHandlers = function() {
        var self = this;
        var toggleBtns = this.container.querySelectorAll('.diff-toggle-btn[data-mode]');
        for (var i = 0; i < toggleBtns.length; i++) {
            toggleBtns[i].addEventListener('click', function() {
                var mode = this.getAttribute('data-mode');
                self.viewMode = mode;
                localStorage.setItem(VIEW_MODE_KEY, mode);
                self.render();
            });
        }

        var rawBtns = this.container.querySelectorAll('.diff-toggle-btn[data-raw]');
        for (var j = 0; j < rawBtns.length; j++) {
            rawBtns[j].addEventListener('click', function() {
                self.rawSubMode = this.getAttribute('data-raw');
                self.render();
            });
        }
    };

    // --- Error Rendering (T019) ---

    DiffViewer.prototype.renderError = function(message) {
        this.container.innerHTML =
            '<h2>Changed Files</h2>' +
            '<div class="diff-error">' + escapeHTML(message) + '</div>';
    };

    // --- SSE Refresh (T020) ---

    DiffViewer.prototype.refreshIfNeeded = function() {
        var now = Date.now();
        if (now - this._lastRefresh < 2000) return; // debounce 2s
        this._lastRefresh = now;
        this.loadFileList();
    };

    // --- Virtualization (T017) ---

    DiffViewer.prototype.needsVirtualization = function() {
        if (!this.currentParsed) return false;
        var totalLines = 0;
        for (var i = 0; i < this.currentParsed.hunks.length; i++) {
            totalLines += this.currentParsed.hunks[i].lines.length + 1; // +1 for hunk header
        }
        return totalLines > VIRTUAL_THRESHOLD;
    };

    DiffViewer.prototype.setupVirtualization = function() {
        var contentEl = document.getElementById('diff-viewer-content');
        if (!contentEl) return;

        var diffBody = contentEl.querySelector('.diff-table-body');
        if (!diffBody) return;

        var allRows = diffBody.querySelectorAll('tr');
        if (allRows.length <= VIRTUAL_THRESHOLD) return;

        var totalHeight = allRows.length * LINE_HEIGHT;
        var self = this;

        // Store all rows data
        var rowsData = [];
        for (var i = 0; i < allRows.length; i++) {
            rowsData.push(allRows[i].outerHTML);
        }

        // Create virtualized container
        var scrollContainer = contentEl.querySelector('.diff-table-container');
        if (!scrollContainer) return;

        scrollContainer.style.height = '600px';
        scrollContainer.style.overflowY = 'auto';
        scrollContainer.style.position = 'relative';

        var table = scrollContainer.querySelector('table');
        if (!table) return;

        var spacer = document.createElement('div');
        spacer.style.height = totalHeight + 'px';
        spacer.style.position = 'relative';

        table.style.position = 'absolute';
        table.style.top = '0';
        table.style.left = '0';
        table.style.right = '0';

        function updateVisibleRows() {
            var scrollTop = scrollContainer.scrollTop;
            var viewportHeight = scrollContainer.clientHeight;
            var startIdx = Math.max(0, Math.floor(scrollTop / LINE_HEIGHT) - VIRTUAL_BUFFER);
            var endIdx = Math.min(rowsData.length, Math.ceil((scrollTop + viewportHeight) / LINE_HEIGHT) + VIRTUAL_BUFFER);

            table.style.top = (startIdx * LINE_HEIGHT) + 'px';
            diffBody.innerHTML = rowsData.slice(startIdx, endIdx).join('');
        }

        var rafId = null;
        scrollContainer.addEventListener('scroll', function() {
            if (rafId) cancelAnimationFrame(rafId);
            rafId = requestAnimationFrame(updateVisibleRows);
        });

        // Wrap table with spacer
        scrollContainer.insertBefore(spacer, table);
        spacer.appendChild(table);

        updateVisibleRows();
    };

    // --- Diff Parser (T010) ---

    function parseDiff(content) {
        var result = { hunks: [] };
        var lines = content.split('\n');
        var currentHunk = null;
        var oldLine = 0;
        var newLine = 0;

        for (var i = 0; i < lines.length; i++) {
            var line = lines[i];

            // Skip diff header lines
            if (line.indexOf('diff --git') === 0 ||
                line.indexOf('index ') === 0 ||
                line.indexOf('---') === 0 ||
                line.indexOf('+++') === 0 ||
                line.indexOf('new file mode') === 0 ||
                line.indexOf('deleted file mode') === 0 ||
                line.indexOf('old mode') === 0 ||
                line.indexOf('new mode') === 0 ||
                line.indexOf('similarity index') === 0 ||
                line.indexOf('rename from') === 0 ||
                line.indexOf('rename to') === 0 ||
                line.indexOf('Binary files') === 0) {
                continue;
            }

            // Hunk header
            var hunkMatch = line.match(/^@@\s+-(\d+)(?:,\d+)?\s+\+(\d+)(?:,\d+)?\s+@@(.*)/);
            if (hunkMatch) {
                currentHunk = {
                    header: line,
                    context: hunkMatch[3] || '',
                    lines: []
                };
                result.hunks.push(currentHunk);
                oldLine = parseInt(hunkMatch[1], 10);
                newLine = parseInt(hunkMatch[2], 10);
                continue;
            }

            if (!currentHunk) continue;

            if (line.charAt(0) === '+') {
                currentHunk.lines.push({
                    type: 'add',
                    content: line.substring(1),
                    oldNum: null,
                    newNum: newLine++
                });
            } else if (line.charAt(0) === '-') {
                currentHunk.lines.push({
                    type: 'del',
                    content: line.substring(1),
                    oldNum: oldLine++,
                    newNum: null
                });
            } else if (line.charAt(0) === '\\') {
                // "\ No newline at end of file" — skip
                continue;
            } else {
                // Context line (starts with space or is the line itself for empty context)
                var ctx = line.length > 0 ? line.substring(1) : '';
                currentHunk.lines.push({
                    type: 'context',
                    content: ctx,
                    oldNum: oldLine++,
                    newNum: newLine++
                });
            }
        }

        return result;
    }

    // --- Unified Diff Renderer (T010) ---

    function renderUnified(parsed, ext) {
        var html = '<div class="diff-table-container"><table class="diff-table"><tbody class="diff-table-body">';

        for (var h = 0; h < parsed.hunks.length; h++) {
            var hunk = parsed.hunks[h];
            html += '<tr class="diff-hunk-header"><td class="diff-line-num" colspan="2"></td><td class="diff-line-content">' + escapeHTML(hunk.header) + '</td></tr>';

            for (var l = 0; l < hunk.lines.length; l++) {
                var line = hunk.lines[l];
                var cls = line.type === 'add' ? 'diff-add' : line.type === 'del' ? 'diff-del' : 'diff-context';
                var oldNum = line.oldNum !== null ? line.oldNum : '';
                var newNum = line.newNum !== null ? line.newNum : '';
                var prefix = line.type === 'add' ? '+' : line.type === 'del' ? '-' : ' ';
                var highlighted = highlightSyntax(line.content, ext);

                html += '<tr class="diff-line ' + cls + '">' +
                    '<td class="diff-line-num diff-line-num-old">' + oldNum + '</td>' +
                    '<td class="diff-line-num diff-line-num-new">' + newNum + '</td>' +
                    '<td class="diff-line-content"><pre>' + prefix + highlighted + '</pre></td>' +
                    '</tr>';
            }
        }

        html += '</tbody></table></div>';
        return html;
    }

    // --- Side-by-Side Renderer (T013) ---

    function renderSideBySide(parsed, ext) {
        var html = '<div class="diff-table-container"><table class="diff-table diff-table-sbs"><tbody class="diff-table-body">';

        for (var h = 0; h < parsed.hunks.length; h++) {
            var hunk = parsed.hunks[h];
            html += '<tr class="diff-hunk-header">' +
                '<td class="diff-line-num"></td><td class="diff-line-content" colspan="1">' + escapeHTML(hunk.header) + '</td>' +
                '<td class="diff-line-num"></td><td class="diff-line-content" colspan="1"></td>' +
                '</tr>';

            // Align lines: pair deletions with additions
            var lines = hunk.lines;
            var i = 0;
            while (i < lines.length) {
                if (lines[i].type === 'context') {
                    var highlighted = highlightSyntax(lines[i].content, ext);
                    html += '<tr class="diff-line diff-context">' +
                        '<td class="diff-line-num">' + lines[i].oldNum + '</td>' +
                        '<td class="diff-line-content diff-sbs-left"><pre> ' + highlighted + '</pre></td>' +
                        '<td class="diff-line-num">' + lines[i].newNum + '</td>' +
                        '<td class="diff-line-content diff-sbs-right"><pre> ' + highlighted + '</pre></td>' +
                        '</tr>';
                    i++;
                } else if (lines[i].type === 'del') {
                    // Collect consecutive deletions and additions
                    var dels = [];
                    while (i < lines.length && lines[i].type === 'del') {
                        dels.push(lines[i]);
                        i++;
                    }
                    var adds = [];
                    while (i < lines.length && lines[i].type === 'add') {
                        adds.push(lines[i]);
                        i++;
                    }
                    var maxLen = Math.max(dels.length, adds.length);
                    for (var k = 0; k < maxLen; k++) {
                        var leftNum = k < dels.length ? dels[k].oldNum : '';
                        var leftContent = k < dels.length ? highlightSyntax(dels[k].content, ext) : '';
                        var leftCls = k < dels.length ? 'diff-del' : '';
                        var leftPrefix = k < dels.length ? '-' : ' ';

                        var rightNum = k < adds.length ? adds[k].newNum : '';
                        var rightContent = k < adds.length ? highlightSyntax(adds[k].content, ext) : '';
                        var rightCls = k < adds.length ? 'diff-add' : '';
                        var rightPrefix = k < adds.length ? '+' : ' ';

                        html += '<tr class="diff-line">' +
                            '<td class="diff-line-num">' + leftNum + '</td>' +
                            '<td class="diff-line-content diff-sbs-left ' + leftCls + '"><pre>' + leftPrefix + leftContent + '</pre></td>' +
                            '<td class="diff-line-num">' + rightNum + '</td>' +
                            '<td class="diff-line-content diff-sbs-right ' + rightCls + '"><pre>' + rightPrefix + rightContent + '</pre></td>' +
                            '</tr>';
                    }
                } else if (lines[i].type === 'add') {
                    var addHighlighted = highlightSyntax(lines[i].content, ext);
                    html += '<tr class="diff-line">' +
                        '<td class="diff-line-num"></td>' +
                        '<td class="diff-line-content diff-sbs-left"><pre></pre></td>' +
                        '<td class="diff-line-num">' + lines[i].newNum + '</td>' +
                        '<td class="diff-line-content diff-sbs-right diff-add"><pre>+' + addHighlighted + '</pre></td>' +
                        '</tr>';
                    i++;
                } else {
                    i++;
                }
            }
        }

        html += '</tbody></table></div>';
        return html;
    }

    // --- Raw Before/After Renderer (T014) ---

    function renderRaw(parsed, mode, ext) {
        // Reconstruct file content from diff
        var lines = [];
        for (var h = 0; h < parsed.hunks.length; h++) {
            var hunk = parsed.hunks[h];
            for (var l = 0; l < hunk.lines.length; l++) {
                var line = hunk.lines[l];
                if (mode === 'before') {
                    if (line.type === 'context' || line.type === 'del') {
                        lines.push({ num: line.type === 'del' ? line.oldNum : line.oldNum, content: line.content });
                    }
                } else {
                    if (line.type === 'context' || line.type === 'add') {
                        lines.push({ num: line.type === 'add' ? line.newNum : line.newNum, content: line.content });
                    }
                }
            }
        }

        var html = '<div class="diff-table-container"><table class="diff-table"><tbody class="diff-table-body">';
        for (var i = 0; i < lines.length; i++) {
            var highlighted = highlightSyntax(lines[i].content, ext);
            html += '<tr class="diff-line diff-context">' +
                '<td class="diff-line-num">' + lines[i].num + '</td>' +
                '<td class="diff-line-content"><pre> ' + highlighted + '</pre></td>' +
                '</tr>';
        }
        html += '</tbody></table></div>';
        return html;
    }

    // --- Syntax Highlighting (T016) ---

    var LANG_KEYWORDS = {
        go: /\b(func|package|import|var|const|type|struct|interface|map|chan|go|defer|return|if|else|for|range|switch|case|default|break|continue|select|fallthrough|nil|true|false|err|error)\b/g,
        js: /\b(function|var|let|const|return|if|else|for|while|do|switch|case|default|break|continue|new|this|class|extends|import|export|from|async|await|try|catch|finally|throw|typeof|instanceof|in|of|null|undefined|true|false|void|delete|yield)\b/g,
        ts: /\b(function|var|let|const|return|if|else|for|while|do|switch|case|default|break|continue|new|this|class|extends|import|export|from|async|await|try|catch|finally|throw|typeof|instanceof|in|of|null|undefined|true|false|void|delete|yield|type|interface|enum|implements|namespace|abstract|as|keyof|readonly|declare|module|require)\b/g,
        yaml: /\b(true|false|null|yes|no|on|off)\b/g,
        sql: /\b(SELECT|FROM|WHERE|INSERT|INTO|UPDATE|DELETE|CREATE|DROP|ALTER|TABLE|INDEX|JOIN|LEFT|RIGHT|INNER|OUTER|ON|AND|OR|NOT|NULL|IS|IN|LIKE|ORDER|BY|GROUP|HAVING|LIMIT|OFFSET|AS|SET|VALUES|COUNT|SUM|AVG|MAX|MIN|DISTINCT|UNION|EXISTS|BETWEEN|CASE|WHEN|THEN|ELSE|END|PRIMARY|KEY|FOREIGN|REFERENCES|UNIQUE|CONSTRAINT|CHECK|DEFAULT|CASCADE|TRIGGER|VIEW|PROCEDURE|FUNCTION|BEGIN|COMMIT|ROLLBACK)\b/gi,
        sh: /\b(if|then|else|elif|fi|for|while|do|done|case|esac|in|function|return|local|export|source|echo|exit|test|set|unset|shift|break|continue|true|false|read|eval|exec|trap)\b/g,
        css: /\b(important|inherit|initial|unset|none|auto|block|inline|flex|grid|relative|absolute|fixed|sticky|hidden|visible|scroll|solid|dashed|dotted|transparent|currentColor)\b/g
    };

    var LANG_MAP = {
        '.go': 'go', '.js': 'js', '.ts': 'ts', '.tsx': 'ts', '.jsx': 'js',
        '.yaml': 'yaml', '.yml': 'yaml', '.json': 'json', '.md': 'md',
        '.html': 'html', '.css': 'css', '.sql': 'sql',
        '.sh': 'sh', '.bash': 'sh', '.zsh': 'sh'
    };

    function getFileExtension(path) {
        if (!path) return '';
        var dot = path.lastIndexOf('.');
        if (dot === -1) return '';
        return path.substring(dot).toLowerCase();
    }

    function highlightSyntax(code, ext) {
        if (!code) return '';

        // HTML-escape first to prevent XSS
        var escaped = escapeHTML(code);
        var lang = LANG_MAP[ext];
        if (!lang) return escaped;

        // JSON: highlight keys and values
        if (lang === 'json') {
            escaped = escaped
                .replace(/(&quot;)((?:[^&]|&(?!quot;))*)(&quot;)\s*:/g, '<span class="syntax-keyword">$1$2$3</span>:')
                .replace(/:\s*(&quot;)((?:[^&]|&(?!quot;))*)(&quot;)/g, ': <span class="syntax-string">$1$2$3</span>')
                .replace(/:\s*(-?\d+(?:\.\d+)?)/g, ': <span class="syntax-number">$1</span>')
                .replace(/:\s*(true|false|null)\b/g, ': <span class="syntax-keyword">$1</span>');
            return escaped;
        }

        // Markdown: headers, bold, code
        if (lang === 'md') {
            escaped = escaped
                .replace(/^(#{1,6}\s.*)$/gm, '<span class="syntax-keyword">$1</span>')
                .replace(/`([^`]+)`/g, '<span class="syntax-string">`$1`</span>');
            return escaped;
        }

        // HTML: tags and attributes
        if (lang === 'html') {
            escaped = escaped
                .replace(/(&lt;\/?)([\w-]+)/g, '$1<span class="syntax-keyword">$2</span>')
                .replace(/([\w-]+)=(&quot;)/g, '<span class="syntax-string">$1=$2</span>');
            return escaped;
        }

        // General languages: comments, strings, numbers, keywords
        // Strings (double-quoted)
        escaped = escaped.replace(/(&quot;)((?:[^&]|&(?!quot;))*)(&quot;)/g, '<span class="syntax-string">$1$2$3</span>');

        // Strings (single-quoted) — for JS/TS/SQL/sh
        if (lang !== 'go') {
            escaped = escaped.replace(/(&#x27;|')((?:[^'\\]|\\.)*)('|&#x27;)/g, '<span class="syntax-string">$1$2$3</span>');
        }

        // Line comments
        if (lang === 'go' || lang === 'js' || lang === 'ts' || lang === 'css') {
            escaped = escaped.replace(/(\/\/.*)/g, '<span class="syntax-comment">$1</span>');
        }
        if (lang === 'sh' || lang === 'yaml') {
            escaped = escaped.replace(/(#.*)/g, '<span class="syntax-comment">$1</span>');
        }
        if (lang === 'sql') {
            escaped = escaped.replace(/(--.*)/g, '<span class="syntax-comment">$1</span>');
        }

        // Numbers
        escaped = escaped.replace(/\b(\d+(?:\.\d+)?)\b/g, '<span class="syntax-number">$1</span>');

        // Keywords
        var kwRegex = LANG_KEYWORDS[lang];
        if (kwRegex) {
            escaped = escaped.replace(kwRegex, '<span class="syntax-keyword">$1</span>');
        }

        return escaped;
    }

    // --- Expose globally ---
    window.DiffViewer = DiffViewer;

})();
