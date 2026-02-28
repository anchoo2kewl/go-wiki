// go-wiki keyboard shortcuts
(function() {
  'use strict';

  var cfg = window.__goWikiConfig || {};

  document.addEventListener('DOMContentLoaded', function() {
    var editor = document.getElementById(cfg.textareaId || 'gw-editor');
    var fsTextarea = document.getElementById('gw-fullscreen-textarea');

    // Tab key inserts 2 spaces in both textareas
    function handleTab(textarea) {
      textarea.addEventListener('keydown', function(e) {
        if (e.key === 'Tab') {
          e.preventDefault();
          var start = textarea.selectionStart;
          var end = textarea.selectionEnd;
          textarea.value = textarea.value.substring(0, start) + '  ' + textarea.value.substring(end);
          textarea.selectionStart = textarea.selectionEnd = start + 2;
        }
      });
    }

    if (editor) handleTab(editor);
    if (fsTextarea) handleTab(fsTextarea);

    // Ctrl+Shift+S/M/L image resize (when an img is focused in preview)
    document.addEventListener('keydown', function(e) {
      if (!e.ctrlKey || !e.shiftKey) return;
      var sel = window.getSelection();
      if (!sel || sel.rangeCount === 0) return;
      var node = sel.anchorNode && sel.anchorNode.parentElement;
      if (!node) return;
      var img = node.tagName === 'IMG' ? node : (node.querySelector && node.querySelector('img'));
      if (!img) return;
      var key = e.key.toLowerCase();
      if (key === 's') { img.style.width = '33%'; e.preventDefault(); }
      if (key === 'm') { img.style.width = '66%'; e.preventDefault(); }
      if (key === 'l') { img.style.width = '100%'; e.preventDefault(); }
    });
  });
})();
