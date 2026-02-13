// markdown.js — Minimal markdown parser for Wave dashboard
// Supports: headings, lists, code blocks, inline code, bold/italic, links, tables
// All content is HTML-escaped before processing to prevent XSS
(function() {
'use strict';

function escapeHtml(text) {
    return text
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

function renderInline(text) {
    // Order matters: process code spans first to protect their content
    var result = '';
    var parts = text.split(/(`[^`]+`)/g);
    for (var i = 0; i < parts.length; i++) {
        if (i % 2 === 1) {
            // Code span — already escaped, just wrap
            result += '<code>' + parts[i].slice(1, -1) + '</code>';
        } else {
            var s = parts[i];
            // Bold
            s = s.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
            // Italic
            s = s.replace(/\*([^*]+)\*/g, '<em>$1</em>');
            // Links [text](url) — only allow http/https/mailto URLs
            s = s.replace(/\[([^\]]+)\]\(([^)]+)\)/g, function(m, t, u) {
                if (/^(https?:|mailto:)/.test(u)) {
                    return '<a href="' + u + '">' + t + '</a>';
                }
                return t;
            });
            result += s;
        }
    }
    return result;
}

function renderMarkdown(text) {
    if (!text) return '';

    // Escape HTML first for XSS safety
    var escaped = escapeHtml(text);
    var lines = escaped.split('\n');
    var html = [];
    var i = 0;

    while (i < lines.length) {
        var line = lines[i];

        // Fenced code block
        if (/^```/.test(line)) {
            var lang = line.slice(3).trim();
            var code = [];
            i++;
            while (i < lines.length && !/^```/.test(lines[i])) {
                code.push(lines[i]);
                i++;
            }
            i++; // skip closing ```
            html.push('<pre><code' + (lang ? ' class="language-' + lang + '"' : '') + '>' + code.join('\n') + '</code></pre>');
            continue;
        }

        // Headings
        var headingMatch = line.match(/^(#{1,4})\s+(.+)/);
        if (headingMatch) {
            var level = headingMatch[1].length;
            html.push('<h' + level + '>' + renderInline(headingMatch[2]) + '</h' + level + '>');
            i++;
            continue;
        }

        // Horizontal rule
        if (/^(---|\*\*\*|___)/.test(line.trim())) {
            html.push('<hr>');
            i++;
            continue;
        }

        // Table
        if (line.indexOf('|') !== -1 && i + 1 < lines.length && /^\|?\s*[-:]+/.test(lines[i + 1])) {
            var tableHtml = '<table>';
            // Header row
            var headerCells = line.split('|').filter(function(c) { return c.trim() !== ''; });
            tableHtml += '<thead><tr>';
            for (var h = 0; h < headerCells.length; h++) {
                tableHtml += '<th>' + renderInline(headerCells[h].trim()) + '</th>';
            }
            tableHtml += '</tr></thead>';
            i += 2; // skip header + separator
            // Body rows
            tableHtml += '<tbody>';
            while (i < lines.length && lines[i].indexOf('|') !== -1) {
                var cells = lines[i].split('|').filter(function(c) { return c.trim() !== ''; });
                tableHtml += '<tr>';
                for (var c = 0; c < cells.length; c++) {
                    tableHtml += '<td>' + renderInline(cells[c].trim()) + '</td>';
                }
                tableHtml += '</tr>';
                i++;
            }
            tableHtml += '</tbody></table>';
            html.push(tableHtml);
            continue;
        }

        // Unordered list
        if (/^[\s]*[-*+]\s+/.test(line)) {
            var items = [];
            while (i < lines.length && /^[\s]*[-*+]\s+/.test(lines[i])) {
                items.push(renderInline(lines[i].replace(/^[\s]*[-*+]\s+/, '')));
                i++;
            }
            html.push('<ul>' + items.map(function(item) { return '<li>' + item + '</li>'; }).join('') + '</ul>');
            continue;
        }

        // Ordered list
        if (/^[\s]*\d+\.\s+/.test(line)) {
            var oitems = [];
            while (i < lines.length && /^[\s]*\d+\.\s+/.test(lines[i])) {
                oitems.push(renderInline(lines[i].replace(/^[\s]*\d+\.\s+/, '')));
                i++;
            }
            html.push('<ol>' + oitems.map(function(item) { return '<li>' + item + '</li>'; }).join('') + '</ol>');
            continue;
        }

        // Empty line
        if (line.trim() === '') {
            i++;
            continue;
        }

        // Paragraph — collect consecutive non-empty lines
        var paraLines = [];
        while (i < lines.length && lines[i].trim() !== '' && !/^#{1,4}\s/.test(lines[i]) && !/^```/.test(lines[i]) && !/^[-*+]\s/.test(lines[i]) && !/^\d+\.\s/.test(lines[i])) {
            paraLines.push(lines[i]);
            i++;
        }
        if (paraLines.length > 0) {
            html.push('<p>' + renderInline(paraLines.join(' ')) + '</p>');
        }
    }

    return html.join('\n');
}

window.renderMarkdown = renderMarkdown;
})();
