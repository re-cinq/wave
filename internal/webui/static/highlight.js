// highlight.js — Regex-based syntax highlighter for Wave dashboard
// Supports: YAML, JSON, Go, SQL, Shell, JavaScript, CSS, HTML, Markdown
// All content is HTML-escaped before tokenization to prevent XSS
(function() {
'use strict';

function escapeHtml(text) {
    return text
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

var languages = {
    yaml: [
        [/^(\s*#.*)$/gm, 'tok-comment'],
        [/^(\s*[\w.-]+)(\s*:)/gm, function(m, key, colon) { return '<span class="tok-key">' + key + '</span>' + colon; }],
        [/:\s*(&quot;[^&]*&quot;|'[^']*')/g, function(m, str) { return ': <span class="tok-str">' + str + '</span>'; }],
        [/\b(true|false|null|yes|no)\b/g, 'tok-bool'],
        [/\b(\d+\.?\d*)\b/g, 'tok-num']
    ],
    json: [
        [/(&quot;[^&]*&quot;)\s*:/g, function(m, key) { return '<span class="tok-key">' + key + '</span>:'; }],
        [/:\s*(&quot;[^&]*&quot;)/g, function(m, str) { return ': <span class="tok-str">' + str + '</span>'; }],
        [/\b(true|false|null)\b/g, 'tok-bool'],
        [/\b(-?\d+\.?\d*([eE][+-]?\d+)?)\b/g, 'tok-num']
    ],
    go: [
        [/\/\/.*$/gm, 'tok-comment'],
        [/&quot;[^&]*&quot;/g, 'tok-str'],
        [/`[^`]*`/g, 'tok-str'],
        [/\b(package|import|func|return|if|else|for|range|switch|case|default|var|const|type|struct|interface|map|chan|go|defer|select|break|continue|fallthrough|nil|true|false|error|string|int|int64|bool|byte|float64|append|len|make|new|panic|recover)\b/g, 'tok-kw'],
        [/\b(\d+\.?\d*)\b/g, 'tok-num']
    ],
    sql: [
        [/--.*$/gm, 'tok-comment'],
        [/'[^']*'/g, 'tok-str'],
        [/\b(SELECT|FROM|WHERE|AND|OR|NOT|INSERT|INTO|VALUES|UPDATE|SET|DELETE|CREATE|TABLE|ALTER|DROP|INDEX|JOIN|LEFT|RIGHT|INNER|OUTER|ON|GROUP|BY|ORDER|ASC|DESC|LIMIT|OFFSET|HAVING|UNION|AS|IN|IS|NULL|LIKE|BETWEEN|EXISTS|CASE|WHEN|THEN|ELSE|END|COUNT|SUM|AVG|MIN|MAX|DISTINCT|PRIMARY|KEY|FOREIGN|REFERENCES|INTEGER|TEXT|REAL|BOOLEAN|DEFAULT|NOT|CONSTRAINT|UNIQUE|CHECK|IF)\b/gi, 'tok-kw'],
        [/\b(\d+\.?\d*)\b/g, 'tok-num']
    ],
    shell: [
        [/#.*$/gm, 'tok-comment'],
        [/&quot;[^&]*&quot;/g, 'tok-str'],
        [/'[^']*'/g, 'tok-str'],
        [/\b(if|then|else|elif|fi|for|while|do|done|case|esac|function|return|exit|echo|export|source|cd|ls|cat|grep|sed|awk|find|xargs|pipe|sudo|chmod|chown|mkdir|rm|cp|mv|ln|test)\b/g, 'tok-kw'],
        [/\$[\w]+/g, 'tok-kw']
    ],
    javascript: [
        [/\/\/.*$/gm, 'tok-comment'],
        [/\/\*[\s\S]*?\*\//gm, 'tok-comment'],
        [/&quot;[^&]*&quot;/g, 'tok-str'],
        [/'[^']*'/g, 'tok-str'],
        [/\b(var|let|const|function|return|if|else|for|while|do|switch|case|default|break|continue|new|this|class|extends|import|export|from|async|await|try|catch|finally|throw|typeof|instanceof|in|of|null|undefined|true|false|void|delete|yield)\b/g, 'tok-kw'],
        [/\b(\d+\.?\d*)\b/g, 'tok-num']
    ],
    css: [
        [/\/\*[\s\S]*?\*\//gm, 'tok-comment'],
        [/([.#]?[\w-]+)\s*\{/g, function(m, sel) { return '<span class="tok-key">' + sel + '</span> {'; }],
        [/([\w-]+)\s*:/g, function(m, prop) { return '<span class="tok-kw">' + prop + '</span>:'; }],
        [/#[0-9a-fA-F]{3,8}\b/g, 'tok-num'],
        [/\b(\d+\.?\d*(px|em|rem|%|vh|vw|s|ms)?)\b/g, 'tok-num']
    ],
    html: [
        [/&lt;!--[\s\S]*?--&gt;/gm, 'tok-comment'],
        [/(&lt;\/?)([\w-]+)/g, function(m, bracket, tag) { return bracket + '<span class="tok-key">' + tag + '</span>'; }],
        [/\s([\w-]+)=(&quot;[^&]*&quot;)/g, function(m, attr, val) { return ' <span class="tok-kw">' + attr + '</span>=<span class="tok-str">' + val + '</span>'; }]
    ],
    markdown: [
        [/^(#{1,6}\s+.*)$/gm, 'tok-key'],
        [/\*\*[^*]+\*\*/g, 'tok-str'],
        [/`[^`]+`/g, 'tok-str'],
        [/\[([^\]]+)\]\([^)]+\)/g, 'tok-kw']
    ]
};

function highlight(code, language) {
    if (!code) return '';

    var escaped = escapeHtml(code);
    var lang = (language || '').toLowerCase();

    // Normalize language aliases
    var aliases = {
        'yml': 'yaml', 'sh': 'shell', 'bash': 'shell',
        'js': 'javascript', 'ts': 'javascript',
        'golang': 'go', 'htm': 'html', 'md': 'markdown'
    };
    if (aliases[lang]) lang = aliases[lang];

    var rules = languages[lang];
    if (!rules) return escaped; // plain text fallback

    for (var i = 0; i < rules.length; i++) {
        var rule = rules[i];
        var pattern = rule[0];
        var replacement = rule[1];

        if (typeof replacement === 'string') {
            escaped = escaped.replace(pattern, '<span class="' + replacement + '">$&</span>');
        } else {
            escaped = escaped.replace(pattern, replacement);
        }
    }

    return escaped;
}

window.highlight = highlight;
})();
