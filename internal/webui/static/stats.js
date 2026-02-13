// stats.js — Statistics page interactions for Wave dashboard
(function() {
'use strict';

document.addEventListener('DOMContentLoaded', function() {
    var rangeSelect = document.getElementById('time-range');
    if (rangeSelect) {
        rangeSelect.addEventListener('change', function() {
            var params = new URLSearchParams(window.location.search);
            params.set('range', this.value);
            window.location.href = '/statistics?' + params.toString();
        });
    }
});
})();
