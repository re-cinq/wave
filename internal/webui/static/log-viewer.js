// Wave Dashboard - Log Viewer for pipeline step output
// Manages log output from pipeline steps via SSE stream_activity events.

function LogViewer() {
    this.sections = new Map();
    this.seenEventIDs = new Set();
    this.search = {
        query: '',
        matches: [],
        currentIndex: -1,
        totalCount: 0,
        debounceTimer: null,
        filterMode: false
    };
    this.connection = {
        status: 'connected',
        retryCount: 0,
        lastEventId: null
    };
    this.pageAutoScroll = true;
    this.batchBuffer = [];
    this.batchTimer = null;
}

// ---------------------------------------------------------------------------
// Initialization
// ---------------------------------------------------------------------------

LogViewer.prototype.init = function(runStatus) {
    var self = this;
    var cards = document.querySelectorAll('.step-card');

    cards.forEach(function(card) {
        var cardId = card.getAttribute('id') || '';
        var stepId = cardId.replace(/^step-/, '');
        if (!stepId) return;

        var badgeEl = card.querySelector('.badge');
        var status = badgeEl ? badgeEl.textContent.trim() : 'pending';
        var nameEl = card.querySelector('.step-id');
        var stepName = nameEl ? nameEl.textContent.trim() : stepId;

        var section = self.createSection(stepId, stepName, status, card);
        self.sections.set(stepId, section);

        if (status === 'running' || status === 'failed') {
            section.expanded = true;
        } else {
            section.expanded = false;
        }
        self._applyCollapsed(section);
        self._attachScrollListener(section);
    });

    // If run is terminal, fetch historical events
    if (runStatus === 'completed' || runStatus === 'failed' || runStatus === 'cancelled') {
        var runID = self._getRunIDFromURL();
        if (runID) {
            fetch('/api/runs/' + encodeURIComponent(runID))
                .then(function(r) { return r.json(); })
                .then(function(data) {
                    if (data.events && data.events.length > 0) {
                        data.events.forEach(function(ev) {
                            if (ev.state === 'stream_activity' && ev.step_id) {
                                self.addLine(ev.step_id, ev);
                            }
                        });
                    }
                })
                .catch(function() {});
        }
    }

    // Wire collapse toggle handlers
    document.querySelectorAll('.step-header').forEach(function(header) {
        header.style.cursor = 'pointer';
        header.addEventListener('click', function() {
            var card = header.closest('.step-card');
            if (!card) return;
            var sid = (card.getAttribute('id') || '').replace(/^step-/, '');
            if (sid) self.toggleSection(sid);
        });
    });

    // Keyboard shortcuts for search
    document.addEventListener('keydown', function(e) {
        if (e.ctrlKey && e.key === 'g') {
            e.preventDefault();
            if (e.shiftKey) {
                self.prevMatch();
            } else {
                self.nextMatch();
            }
        }
    });

    // Wire search DOM elements
    var searchInput = document.getElementById('log-search-input');
    if (searchInput) {
        searchInput.addEventListener('input', function() {
            self.searchQuery(searchInput.value);
        });
    }
    var searchNext = document.getElementById('log-search-next');
    if (searchNext) {
        searchNext.addEventListener('click', function() { self.nextMatch(); });
    }
    var searchPrev = document.getElementById('log-search-prev');
    if (searchPrev) {
        searchPrev.addEventListener('click', function() { self.prevMatch(); });
    }
    var searchClear = document.getElementById('log-search-clear');
    if (searchClear) {
        searchClear.addEventListener('click', function() { self.clearSearch(); });
    }

    // Wire filter toggle button
    var filterToggle = document.getElementById('log-filter-toggle');
    if (filterToggle) {
        filterToggle.addEventListener('click', function() { self.toggleFilterMode(); });
    }

    // Page-level scroll listener: disable auto-scroll to active step on manual scroll
    var scrollTimer = null;
    window.addEventListener('scroll', function() {
        if (scrollTimer) clearTimeout(scrollTimer);
        scrollTimer = setTimeout(function() {
            self.pageAutoScroll = false;
        }, 100);
    }, { passive: true });
};

// ---------------------------------------------------------------------------
// LogSection factory
// ---------------------------------------------------------------------------

LogViewer.prototype.createSection = function(stepId, stepName, status, element) {
    return {
        stepId: stepId,
        stepName: stepName,
        status: status,
        expanded: false,
        autoScroll: true,
        lines: [],
        lineCount: 0,
        element: element,
        lastMessage: null,
        lastRepeatElement: null,
        repeatCount: 0
    };
};

// ---------------------------------------------------------------------------
// addLine — create a LogLine from SSE event data and buffer it
// ---------------------------------------------------------------------------

LogViewer.prototype.addLine = function(stepId, eventData) {
    var section = this.sections.get(stepId);
    if (!section) return;

    // De-duplicate: skip if event ID already seen
    var eventID = eventData.id;
    if (eventID) {
        if (this.seenEventIDs.has(eventID)) return;
        this.seenEventIDs.add(eventID);
        // Cap the dedup set at 10000 entries
        if (this.seenEventIDs.size > 10000) {
            var iter = this.seenEventIDs.values();
            this.seenEventIDs.delete(iter.next().value);
        }
    }

    var rawContent = eventData.message || '';

    // Repeat collapsing: detect consecutive identical stream_activity messages
    if (eventData.state === 'stream_activity' && rawContent === section.lastMessage && rawContent !== '') {
        section.repeatCount++;
        // Update existing repeat element or create one
        if (section.lastRepeatElement) {
            var badge = section.lastRepeatElement.querySelector('.log-repeat-badge');
            if (badge) {
                badge.textContent = '\u00d7' + (section.repeatCount + 1);
            }
        } else {
            // Convert last line to have a repeat badge
            var lastLine = section.lines[section.lines.length - 1];
            if (lastLine && lastLine.element) {
                var badge = document.createElement('span');
                badge.className = 'log-repeat-badge';
                badge.textContent = '\u00d7' + (section.repeatCount + 1);
                lastLine.element.appendChild(badge);
                lastLine.element.classList.add('log-repeated');
                section.lastRepeatElement = lastLine.element;
            }
        }
        return;
    }

    // Reset repeat tracking for new unique message
    section.lastMessage = rawContent;
    section.lastRepeatElement = null;
    section.repeatCount = 0;

    section.lineCount++;
    var lineNumber = section.lineCount;
    var htmlContent = this.ansiToHtml(rawContent);

    var line = {
        lineNumber: lineNumber,
        timestamp: eventData.timestamp || '',
        stepId: stepId,
        toolName: eventData.tool_name || '',
        toolTarget: eventData.tool_target || '',
        message: eventData.message || '',
        rawContent: rawContent,
        htmlContent: htmlContent,
        element: null
    };

    section.lines.push(line);
    this.batchBuffer.push(line);

    // Check new line against active search
    if (this.search.query) {
        var lowerQuery = this.search.query.toLowerCase();
        var lowerRaw = rawContent.toLowerCase();
        var charStart = lowerRaw.indexOf(lowerQuery);
        while (charStart !== -1) {
            this.search.matches.push({
                stepId: stepId,
                lineIndex: section.lines.length - 1,
                charStart: charStart,
                charEnd: charStart + this.search.query.length
            });
            this.search.totalCount++;
            charStart = lowerRaw.indexOf(lowerQuery, charStart + 1);
        }
        this._updateSearchCount();
    }

    // Schedule flush
    if (this.batchTimer === null) {
        var self = this;
        this.batchTimer = requestAnimationFrame(function() {
            self.flushBatch();
        });
    }
};

// ---------------------------------------------------------------------------
// flushBatch — batch DOM insertions using DocumentFragment
// ---------------------------------------------------------------------------

LogViewer.prototype.flushBatch = function() {
    var self = this;
    self.batchTimer = null;

    var maxPerFrame = 100;
    var toProcess = self.batchBuffer.splice(0, maxPerFrame);
    if (toProcess.length === 0) return;

    // Group lines by stepId for batched insertion
    var groups = {};
    toProcess.forEach(function(line) {
        if (!groups[line.stepId]) groups[line.stepId] = [];
        groups[line.stepId].push(line);
    });

    var sectionIds = Object.keys(groups);
    for (var i = 0; i < sectionIds.length; i++) {
        var sid = sectionIds[i];
        var section = self.sections.get(sid);
        if (!section) continue;

        var logBody = self._getLogBody(section);
        if (!logBody) continue;

        var frag = document.createDocumentFragment();
        var lines = groups[sid];

        for (var j = 0; j < lines.length; j++) {
            var line = lines[j];
            var div = document.createElement('div');
            div.className = 'log-line';
            div.style.contain = 'content';

            var gutter = document.createElement('span');
            gutter.className = 'log-gutter';
            gutter.textContent = line.lineNumber;

            var timeSpan = document.createElement('span');
            timeSpan.className = 'log-time';
            timeSpan.textContent = self._formatTime(line.timestamp);

            var toolSpan = document.createElement('span');
            toolSpan.className = 'log-tool';
            toolSpan.textContent = line.toolName;

            var contentSpan = document.createElement('span');
            contentSpan.className = 'log-content';
            contentSpan.innerHTML = line.htmlContent;

            div.appendChild(gutter);
            div.appendChild(timeSpan);
            div.appendChild(toolSpan);
            div.appendChild(contentSpan);

            line.element = div;
            frag.appendChild(div);
        }

        logBody.appendChild(frag);

        // Auto-scroll to bottom if enabled for this section
        if (section.autoScroll) {
            logBody.scrollTop = logBody.scrollHeight;
        }

        // Apply search highlights to new lines if search is active
        if (self.search.query) {
            for (var k = 0; k < lines.length; k++) {
                self._highlightLineIfMatch(lines[k]);
            }
        }
    }

    // If buffer still has items, schedule another frame
    if (self.batchBuffer.length > 0) {
        self.batchTimer = requestAnimationFrame(function() {
            self.flushBatch();
        });
    }
};

// ---------------------------------------------------------------------------
// toggleSection — expand/collapse a step's log section
// ---------------------------------------------------------------------------

LogViewer.prototype.toggleSection = function(stepId) {
    var section = this.sections.get(stepId);
    if (!section) return;

    section.expanded = !section.expanded;
    this._applyCollapsed(section);
};

LogViewer.prototype._applyCollapsed = function(section) {
    var card = section.element;
    if (!card) return;

    if (section.expanded) {
        card.classList.remove('step-collapsed');
        card.setAttribute('data-expanded', 'true');
    } else {
        card.classList.add('step-collapsed');
        card.setAttribute('data-expanded', 'false');
    }

    // Auto-scroll to bottom when expanding if autoScroll is enabled
    if (section.expanded && section.autoScroll) {
        var logBody = card.querySelector('.step-log-content');
        if (logBody) {
            logBody.scrollTop = logBody.scrollHeight;
        }
    }
};


// ---------------------------------------------------------------------------
// Auto-scroll helpers
// ---------------------------------------------------------------------------

LogViewer.prototype._isNearBottom = function(element) {
    return element.scrollTop + element.clientHeight >= element.scrollHeight - 50;
};

LogViewer.prototype._attachScrollListener = function(section) {
    var self = this;
    var logBody = section.element ? section.element.querySelector('.step-log-content') : null;
    if (!logBody) return;

    var jumpBtn = section.element.querySelector('.jump-to-bottom');

    logBody.addEventListener('scroll', function() {
        var nearBottom = self._isNearBottom(logBody);
        section.autoScroll = nearBottom;
        if (jumpBtn) {
            if (nearBottom) {
                jumpBtn.classList.remove('visible');
            } else {
                jumpBtn.classList.add('visible');
            }
        }
    }, { passive: true });

    if (jumpBtn) {
        jumpBtn.addEventListener('click', function() {
            logBody.scrollTop = logBody.scrollHeight;
            section.autoScroll = true;
            jumpBtn.classList.remove('visible');
        });
    }
};

// ---------------------------------------------------------------------------
// ANSI to HTML parser
// ---------------------------------------------------------------------------

LogViewer.prototype.ansiToHtml = function(rawText) {
    if (!rawText) return '';

    var result = '';
    var lastIndex = 0;
    var openSpans = 0;

    // Active style state
    var bold = false;
    var italic = false;
    var underline = false;
    var strikethrough = false;
    var fgColor = null;   // null, class name, or inline color string
    var bgColor = null;
    var fgInline = false; // true if fgColor is inline style (256-color)
    var bgInline = false;

    var ansiRegex = /\x1b\[([0-9;]*)m/g;
    var match;

    while ((match = ansiRegex.exec(rawText)) !== null) {
        // Emit text before this escape
        if (match.index > lastIndex) {
            var text = rawText.substring(lastIndex, match.index);
            result += htmlEscape(text);
        }
        lastIndex = match.index + match[0].length;

        var codes = match[1] ? match[1].split(';').map(Number) : [0];
        var ci = 0;
        while (ci < codes.length) {
            var code = codes[ci];

            if (code === 0) {
                // Reset — close all open spans
                while (openSpans > 0) { result += '</span>'; openSpans--; }
                bold = false; italic = false; underline = false; strikethrough = false;
                fgColor = null; bgColor = null; fgInline = false; bgInline = false;
            } else if (code === 1) {
                bold = true;
            } else if (code === 3) {
                italic = true;
            } else if (code === 4) {
                underline = true;
            } else if (code === 9) {
                strikethrough = true;
            } else if (code >= 30 && code <= 37) {
                fgColor = 'ansi-fg-' + (code - 30); fgInline = false;
            } else if (code >= 40 && code <= 47) {
                bgColor = 'ansi-bg-' + (code - 40); bgInline = false;
            } else if (code >= 90 && code <= 97) {
                fgColor = 'ansi-fg-' + (code - 90 + 8); fgInline = false;
            } else if (code >= 100 && code <= 107) {
                bgColor = 'ansi-bg-' + (code - 100 + 8); bgInline = false;
            } else if (code === 38 && codes[ci + 1] === 5 && ci + 2 < codes.length) {
                // 256-color foreground
                fgColor = ansi256ToHex(codes[ci + 2]);
                fgInline = true;
                ci += 2;
            } else if (code === 48 && codes[ci + 1] === 5 && ci + 2 < codes.length) {
                // 256-color background
                bgColor = ansi256ToHex(codes[ci + 2]);
                bgInline = true;
                ci += 2;
            } else if (code === 39) {
                fgColor = null; fgInline = false;
            } else if (code === 49) {
                bgColor = null; bgInline = false;
            }
            ci++;
        }

        // Close previous spans and open new ones with current state
        while (openSpans > 0) { result += '</span>'; openSpans--; }

        var styles = [];
        var classes = [];
        if (bold) styles.push('font-weight:bold');
        if (italic) styles.push('font-style:italic');
        if (underline) styles.push('text-decoration:underline');
        if (strikethrough) styles.push('text-decoration:line-through');
        if (fgColor) {
            if (fgInline) { styles.push('color:' + fgColor); } else { classes.push(fgColor); }
        }
        if (bgColor) {
            if (bgInline) { styles.push('background-color:' + bgColor); } else { classes.push(bgColor); }
        }

        if (styles.length > 0 || classes.length > 0) {
            var attrs = '';
            if (classes.length > 0) attrs += ' class="' + classes.join(' ') + '"';
            if (styles.length > 0) attrs += ' style="' + styles.join(';') + '"';
            result += '<span' + attrs + '>';
            openSpans++;
        }
    }

    // Emit remaining text
    if (lastIndex < rawText.length) {
        result += htmlEscape(rawText.substring(lastIndex));
    }

    // Close any unclosed spans
    while (openSpans > 0) { result += '</span>'; openSpans--; }

    return result;
};

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

LogViewer.prototype.searchQuery = function(query) {
    var self = this;
    if (self.search.debounceTimer) {
        clearTimeout(self.search.debounceTimer);
    }
    self.search.debounceTimer = setTimeout(function() {
        self._executeSearch(query);
    }, 300);
};

LogViewer.prototype._executeSearch = function(query) {
    var self = this;
    // Clear previous highlights
    self._clearHighlights();

    self.search.query = query;
    self.search.matches = [];
    self.search.currentIndex = -1;
    self.search.totalCount = 0;

    if (!query) {
        self._updateSearchCount();
        return;
    }

    var lowerQuery = query.toLowerCase();

    self.sections.forEach(function(section) {
        for (var i = 0; i < section.lines.length; i++) {
            var line = section.lines[i];
            var lowerRaw = line.rawContent.toLowerCase();
            var charStart = lowerRaw.indexOf(lowerQuery);
            while (charStart !== -1) {
                self.search.matches.push({
                    stepId: section.stepId,
                    lineIndex: i,
                    charStart: charStart,
                    charEnd: charStart + query.length
                });
                self.search.totalCount++;
                charStart = lowerRaw.indexOf(lowerQuery, charStart + 1);
            }
        }
    });

    self._updateSearchCount();

    if (self.search.matches.length > 0) {
        self.search.currentIndex = 0;
        self._applyVisibleHighlights();
        self._scrollToCurrentMatch();
    }

    // Apply filter if filter mode is active
    if (self.search.filterMode) {
        self._applyFilter();
    }
};

LogViewer.prototype.nextMatch = function() {
    if (this.search.matches.length === 0) return;
    this.search.currentIndex = (this.search.currentIndex + 1) % this.search.matches.length;
    this._applyVisibleHighlights();
    this._scrollToCurrentMatch();
};

LogViewer.prototype.prevMatch = function() {
    if (this.search.matches.length === 0) return;
    this.search.currentIndex = (this.search.currentIndex - 1 + this.search.matches.length) % this.search.matches.length;
    this._applyVisibleHighlights();
    this._scrollToCurrentMatch();
};

LogViewer.prototype.clearSearch = function() {
    this._clearHighlights();
    this.search.query = '';
    this.search.matches = [];
    this.search.currentIndex = -1;
    this.search.totalCount = 0;
    if (this.search.debounceTimer) {
        clearTimeout(this.search.debounceTimer);
        this.search.debounceTimer = null;
    }
    var input = document.getElementById('log-search-input');
    if (input) input.value = '';
    this._updateSearchCount();
    this._applyFilter();
};

// ---------------------------------------------------------------------------
// Filter mode — hide non-matching lines
// ---------------------------------------------------------------------------

LogViewer.prototype.toggleFilterMode = function() {
    this.search.filterMode = !this.search.filterMode;
    var btn = document.getElementById('log-filter-toggle');
    if (btn) {
        if (this.search.filterMode) {
            btn.classList.add('log-filter-active');
        } else {
            btn.classList.remove('log-filter-active');
        }
    }
    this._applyFilter();
};

LogViewer.prototype._applyFilter = function() {
    var self = this;
    var query = self.search.query;
    var filterMode = self.search.filterMode;
    var visibleCount = 0;
    var totalCount = 0;

    self.sections.forEach(function(section) {
        for (var i = 0; i < section.lines.length; i++) {
            var line = section.lines[i];
            totalCount++;
            if (!line.element) continue;

            if (filterMode && query) {
                var lowerRaw = line.rawContent.toLowerCase();
                var lowerQuery = query.toLowerCase();
                if (lowerRaw.indexOf(lowerQuery) === -1) {
                    line.element.style.display = 'none';
                } else {
                    line.element.style.display = '';
                    visibleCount++;
                }
            } else {
                line.element.style.display = '';
                visibleCount++;
            }
        }
    });

    // Update filter count display
    var countEl = document.getElementById('log-filter-count');
    if (countEl) {
        if (filterMode && query) {
            countEl.textContent = visibleCount + '/' + totalCount;
        } else {
            countEl.textContent = '';
        }
    }
};

LogViewer.prototype._clearHighlights = function() {
    var marks = document.querySelectorAll('mark.search-match');
    for (var i = 0; i < marks.length; i++) {
        var mark = marks[i];
        var parent = mark.parentNode;
        if (parent) {
            parent.replaceChild(document.createTextNode(mark.textContent), mark);
            parent.normalize();
        }
    }
};

LogViewer.prototype._applyVisibleHighlights = function() {
    var self = this;
    if (!self.search.query || self.search.matches.length === 0) return;

    // Clear existing highlights first
    self._clearHighlights();

    // Only highlight lines within +/- 50 of current match to optimize for large logs
    var currentMatch = self.search.matches[self.search.currentIndex];
    if (!currentMatch) return;

    var currentSection = self.sections.get(currentMatch.stepId);
    if (!currentSection) return;
    var centerLine = currentMatch.lineIndex;

    // Highlight matches near the visible area
    for (var m = 0; m < self.search.matches.length; m++) {
        var match = self.search.matches[m];
        var section = self.sections.get(match.stepId);
        if (!section) continue;

        // Only highlight within +/- 50 lines of current match in same section,
        // or all matches in other sections that are visible
        if (match.stepId === currentMatch.stepId) {
            if (Math.abs(match.lineIndex - centerLine) > 50) continue;
        }

        var line = section.lines[match.lineIndex];
        if (!line || !line.element) continue;

        var contentSpan = line.element.querySelector('.log-content');
        if (!contentSpan) continue;

        var isCurrent = (m === self.search.currentIndex);
        self._highlightMatch(contentSpan, line.rawContent, self.search.query, isCurrent);
    }
};

LogViewer.prototype._highlightMatch = function(contentSpan, rawContent, query, isCurrent) {
    // Re-render content with highlight marks
    var lowerRaw = rawContent.toLowerCase();
    var lowerQuery = query.toLowerCase();
    var html = '';
    var lastIdx = 0;
    var idx = lowerRaw.indexOf(lowerQuery);

    while (idx !== -1) {
        // Text before match — use ansiToHtml for proper rendering
        html += this.ansiToHtml(rawContent.substring(lastIdx, idx));

        var matchText = rawContent.substring(idx, idx + query.length);
        var cls = 'search-match' + (isCurrent ? ' search-current' : '');
        html += '<mark class="' + cls + '">' + htmlEscape(matchText) + '</mark>';

        // Only mark the first occurrence as current
        isCurrent = false;

        lastIdx = idx + query.length;
        idx = lowerRaw.indexOf(lowerQuery, lastIdx);
    }

    html += this.ansiToHtml(rawContent.substring(lastIdx));
    contentSpan.innerHTML = html;
};

LogViewer.prototype._scrollToCurrentMatch = function() {
    if (this.search.currentIndex < 0 || this.search.currentIndex >= this.search.matches.length) return;

    var match = this.search.matches[this.search.currentIndex];
    var section = this.sections.get(match.stepId);
    if (!section) return;

    // Expand the section if collapsed
    if (!section.expanded) {
        section.expanded = true;
        this._applyCollapsed(section);
    }

    // Disable auto-scroll to prevent fighting with search navigation
    section.autoScroll = false;

    var line = section.lines[match.lineIndex];
    if (line && line.element) {
        line.element.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
};

LogViewer.prototype._highlightLineIfMatch = function(line) {
    if (!this.search.query) return;
    var lowerQuery = this.search.query.toLowerCase();
    var lowerRaw = line.rawContent.toLowerCase();
    if (lowerRaw.indexOf(lowerQuery) === -1) return;

    if (!line.element) return;
    var contentSpan = line.element.querySelector('.log-content');
    if (!contentSpan) return;

    this._highlightMatch(contentSpan, line.rawContent, this.search.query, false);
};

LogViewer.prototype._updateSearchCount = function() {
    var el = document.getElementById('log-search-count');
    if (!el) return;
    if (!this.search.query) {
        el.textContent = '';
    } else if (this.search.totalCount === 0) {
        el.textContent = 'No matches';
    } else {
        el.textContent = (this.search.currentIndex + 1) + ' / ' + this.search.totalCount;
    }
};

// ---------------------------------------------------------------------------
// Download and Copy
// ---------------------------------------------------------------------------

LogViewer.prototype.downloadLog = function(stepId) {
    var section = this.sections.get(stepId);
    if (!section) return;

    var text = '';
    for (var i = 0; i < section.lines.length; i++) {
        text += section.lines[i].rawContent + '\n';
    }

    var blob = new Blob([text], { type: 'text/plain' });
    var url = URL.createObjectURL(blob);
    var a = document.createElement('a');
    a.href = url;
    a.download = stepId + '.log';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
};

LogViewer.prototype.copyLog = function(stepId) {
    var section = this.sections.get(stepId);
    if (!section) return;

    var card = section.element;
    var btn = card ? card.querySelector('.btn-icon[title="Copy log"]') : null;

    if (section.lines.length === 0) {
        if (btn) {
            var orig = btn.textContent;
            btn.textContent = 'No logs';
            setTimeout(function() { btn.textContent = orig; }, 1500);
        }
        return;
    }

    var text = '';
    for (var i = 0; i < section.lines.length; i++) {
        text += section.lines[i].rawContent + '\n';
    }

    if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(function() {
            showCopiedFeedback(btn);
        }).catch(function() {
            fallbackCopy(text, btn);
        });
    } else {
        fallbackCopy(text, btn);
    }
};

// ---------------------------------------------------------------------------
// ConnectionState management
// ---------------------------------------------------------------------------

LogViewer.prototype.onDisconnect = function() {
    this.connection.retryCount++;
    this.connection.status = 'reconnecting';

    var banner = document.getElementById('connection-banner');
    if (banner) {
        banner.hidden = false;
        banner.classList.remove('disconnected');
    }

    if (this.connection.retryCount >= 3) {
        this.connection.status = 'disconnected';
        if (banner) banner.classList.add('disconnected');
    }
};

LogViewer.prototype.onReconnect = function() {
    this.connection.status = 'connected';
    this.connection.retryCount = 0;

    var banner = document.getElementById('connection-banner');
    if (banner) {
        banner.hidden = true;
        banner.classList.remove('disconnected');
    }
};

LogViewer.prototype.reconnect = function() {
    if (typeof connectSSE === 'function' && currentRunID) {
        connectSSE(currentRunID);
    }
};

// ---------------------------------------------------------------------------
// Step state change handler
// ---------------------------------------------------------------------------

LogViewer.prototype.onStepStateChange = function(stepId, newState) {
    var section = this.sections.get(stepId);
    if (!section) return;

    section.status = newState;

    // Remove step-active from all cards, apply to current running step
    var allCards = document.querySelectorAll('.step-card');
    for (var i = 0; i < allCards.length; i++) {
        allCards[i].classList.remove('step-active');
    }

    if (newState === 'running' || newState === 'failed') {
        section.expanded = true;
        this._applyCollapsed(section);

        if (newState === 'running' && section.element) {
            section.element.classList.add('step-active');

            // Auto-scroll to active step card (page-level)
            if (this.pageAutoScroll) {
                section.element.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
            }
        }
    } else if (newState === 'completed') {
        section.expanded = false;
        this._applyCollapsed(section);

        // Insert empty-output notice if no lines were logged
        if (section.lineCount === 0) {
            var logBody = this._getLogBody(section);
            if (logBody) {
                var emptyDiv = document.createElement('div');
                emptyDiv.className = 'log-empty';
                emptyDiv.textContent = 'No output';
                logBody.appendChild(emptyDiv);
            }
        }
    }
};

// ---------------------------------------------------------------------------
// reattach — re-bind after polling rebuilds step cards
// ---------------------------------------------------------------------------

LogViewer.prototype.reattach = function() {
    var self = this;
    var cards = document.querySelectorAll('.step-card');

    cards.forEach(function(card) {
        var cardId = card.getAttribute('id') || '';
        var stepId = cardId.replace(/^step-/, '');
        if (!stepId) return;

        var section = self.sections.get(stepId);
        if (section) {
            // Re-bind element
            section.element = card;

            // Re-apply collapsed state
            self._applyCollapsed(section);

            // Re-create log body content from accumulated lines
            var logBody = self._getLogBody(section);
            if (logBody) {
                // Clear existing log lines
                var existingLines = logBody.querySelectorAll('.log-line');
                for (var i = 0; i < existingLines.length; i++) {
                    existingLines[i].remove();
                }

                // Re-append all accumulated lines
                var frag = document.createDocumentFragment();
                for (var j = 0; j < section.lines.length; j++) {
                    var line = section.lines[j];
                    var div = document.createElement('div');
                    div.className = 'log-line';
                    div.style.contain = 'content';

                    var gutter = document.createElement('span');
                    gutter.className = 'log-gutter';
                    gutter.textContent = line.lineNumber;

                    var timeSpan = document.createElement('span');
                    timeSpan.className = 'log-time';
                    timeSpan.textContent = self._formatTime(line.timestamp);

                    var toolSpan = document.createElement('span');
                    toolSpan.className = 'log-tool';
                    toolSpan.textContent = line.toolName;

                    var contentSpan = document.createElement('span');
                    contentSpan.className = 'log-content';
                    contentSpan.innerHTML = line.htmlContent;

                    div.appendChild(gutter);
                    div.appendChild(timeSpan);
                    div.appendChild(toolSpan);
                    div.appendChild(contentSpan);

                    line.element = div;
                    frag.appendChild(div);
                }
                logBody.appendChild(frag);

                // Auto-scroll to bottom if enabled
                if (section.autoScroll) {
                    logBody.scrollTop = logBody.scrollHeight;
                }
            }

            // Re-bind scroll listener for rebuilt DOM
            self._attachScrollListener(section);

        } else {
            // New section not seen before
            var badgeEl = card.querySelector('.badge');
            var status = badgeEl ? badgeEl.textContent.trim() : 'pending';
            var nameEl = card.querySelector('.step-id');
            var stepName = nameEl ? nameEl.textContent.trim() : stepId;
            var newSection = self.createSection(stepId, stepName, status, card);
            self.sections.set(stepId, newSection);
            self._applyCollapsed(newSection);
            self._attachScrollListener(newSection);
        }
    });

    // Re-wire header click handlers
    document.querySelectorAll('.step-header').forEach(function(header) {
        header.style.cursor = 'pointer';
        // Remove old listeners by cloning
        var newHeader = header.cloneNode(true);
        header.parentNode.replaceChild(newHeader, header);
        newHeader.addEventListener('click', function() {
            var card = newHeader.closest('.step-card');
            if (!card) return;
            var sid = (card.getAttribute('id') || '').replace(/^step-/, '');
            if (sid) self.toggleSection(sid);
        });
    });
};

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

LogViewer.prototype._getLogBody = function(section) {
    if (!section.element) return null;
    var logBody = section.element.querySelector('.step-log-content');
    if (!logBody) {
        // Create log container if it does not exist
        var logContainer = document.createElement('div');
        logContainer.className = 'step-log';
        logContainer.id = 'log-' + section.stepId;
        logContainer.setAttribute('data-step-id', section.stepId);
        logBody = document.createElement('div');
        logBody.className = 'step-log-content';
        logContainer.appendChild(logBody);

        // Add Jump to Bottom button
        var jumpBtn = document.createElement('button');
        jumpBtn.className = 'jump-to-bottom';
        jumpBtn.setAttribute('aria-label', 'Jump to bottom');
        jumpBtn.innerHTML = '&#8595; Bottom';
        logContainer.appendChild(jumpBtn);

        var stepBody = section.element.querySelector('.step-body');
        if (stepBody) {
            stepBody.appendChild(logContainer);
        } else {
            section.element.appendChild(logContainer);
        }

        // Attach scroll listener for the new container
        this._attachScrollListener(section);
    }
    return logBody;
};

LogViewer.prototype._formatTime = function(isoTimestamp) {
    if (!isoTimestamp) return '';
    try {
        var d = new Date(isoTimestamp);
        if (isNaN(d.getTime())) return '';
        var hh = String(d.getHours()).padStart(2, '0');
        var mm = String(d.getMinutes()).padStart(2, '0');
        var ss = String(d.getSeconds()).padStart(2, '0');
        var ms = String(d.getMilliseconds()).padStart(3, '0');
        return hh + ':' + mm + ':' + ss + '.' + ms;
    } catch (e) {
        return '';
    }
};

LogViewer.prototype._getRunIDFromURL = function() {
    // URL format: /runs/{runID}
    var parts = window.location.pathname.split('/');
    for (var i = 0; i < parts.length; i++) {
        if (parts[i] === 'runs' && i + 1 < parts.length) {
            return parts[i + 1];
        }
    }
    return null;
};

// ---------------------------------------------------------------------------
// Standalone utility functions
// ---------------------------------------------------------------------------

function htmlEscape(text) {
    return text
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function ansi256ToHex(n) {
    if (n < 0 || n > 255) return '#ffffff';

    // Standard 16 colors — use approximate hex values
    var standard16 = [
        '#000000', '#aa0000', '#00aa00', '#aa5500', '#0000aa', '#aa00aa', '#00aaaa', '#aaaaaa',
        '#555555', '#ff5555', '#55ff55', '#ffff55', '#5555ff', '#ff55ff', '#55ffff', '#ffffff'
    ];
    if (n < 16) return standard16[n];

    // 216-color cube (indices 16-231)
    if (n < 232) {
        var idx = n - 16;
        var r = Math.floor(idx / 36);
        var g = Math.floor((idx % 36) / 6);
        var b = idx % 6;
        var levels = [0, 95, 135, 175, 215, 255];
        return '#' + toHex2(levels[r]) + toHex2(levels[g]) + toHex2(levels[b]);
    }

    // Grayscale (indices 232-255)
    var gray = 8 + (n - 232) * 10;
    return '#' + toHex2(gray) + toHex2(gray) + toHex2(gray);
}

function toHex2(num) {
    var h = num.toString(16);
    return h.length < 2 ? '0' + h : h;
}

function fallbackCopy(text, btn) {
    var textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    try {
        document.execCommand('copy');
        showCopiedFeedback(btn);
    } catch (e) {
        // silently fail
    }
    document.body.removeChild(textarea);
}

function showCopiedFeedback(btn) {
    if (!btn) return;
    var original = btn.textContent;
    btn.textContent = 'Copied!';
    setTimeout(function() {
        btn.textContent = original;
    }, 1500);
}

// ---------------------------------------------------------------------------
// Instantiate singleton
// ---------------------------------------------------------------------------

window.logViewer = new LogViewer();
