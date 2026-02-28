// go-wiki editor core — toolbar handlers, preview, undo/redo, helpers
// All configuration comes from window.__goWikiConfig
(function() {
  'use strict';

  var cfg = window.__goWikiConfig || {};
  var textareaId = cfg.textareaId || 'gw-editor';
  var previewId = cfg.previewId || 'gw-preview-content';

  function getEditor() {
    return document.getElementById(textareaId);
  }

  // ---------------------------------------------------------------------------
  // Undo / Redo manager — keeps up to maxSize snapshots per textarea.
  // Attach with createUndoManager(textarea, 10). The manager is stored on the
  // element as textarea._undoMgr so insertAtCursor/wrapSelection can
  // automatically checkpoint before mutating the value.
  // ---------------------------------------------------------------------------
  function createUndoManager(textarea, maxSize) {
    maxSize = maxSize || 10;
    var undoStack = [];
    var redoStack = [];
    var inputTimer = null;

    function snap() {
      return { v: textarea.value, s: textarea.selectionStart, e: textarea.selectionEnd };
    }
    function apply(st) {
      textarea.value = st.v;
      textarea.selectionStart = st.s;
      textarea.selectionEnd = st.e;
      textarea.focus();
    }
    function push(stack, st) {
      if (stack.length && stack[stack.length - 1].v === st.v) return;
      stack.push(st);
      if (stack.length > maxSize) stack.shift();
    }

    // Save current state before a programmatic change.
    function checkpoint() {
      if (inputTimer) { clearTimeout(inputTimer); inputTimer = null; }
      push(undoStack, snap());
      redoStack.length = 0;
    }
    function undo() {
      if (undoStack.length === 0) return false;
      if (inputTimer) { clearTimeout(inputTimer); inputTimer = null; }
      push(redoStack, snap());
      apply(undoStack.pop());
      return true;
    }
    function redo() {
      if (redoStack.length === 0) return false;
      push(undoStack, snap());
      apply(redoStack.pop());
      return true;
    }
    function reset() {
      undoStack.length = 0;
      redoStack.length = 0;
      if (inputTimer) { clearTimeout(inputTimer); inputTimer = null; }
    }

    // Debounced typing → automatic checkpoint after 800 ms of silence.
    textarea.addEventListener('input', function() {
      if (inputTimer) clearTimeout(inputTimer);
      inputTimer = setTimeout(function() {
        push(undoStack, snap());
        redoStack.length = 0;
        inputTimer = null;
      }, 800);
    });

    // Keyboard: Cmd/Ctrl+Z = undo, Cmd/Ctrl+Shift+Z = redo, Ctrl+Y = redo
    textarea.addEventListener('keydown', function(e) {
      var isMod = e.metaKey || e.ctrlKey;
      if (isMod && e.key.toLowerCase() === 'z') {
        e.preventDefault();
        if (e.shiftKey) redo(); else undo();
      }
      if (e.ctrlKey && !e.metaKey && e.key.toLowerCase() === 'y') {
        e.preventDefault();
        redo();
      }
    });

    var mgr = { checkpoint: checkpoint, undo: undo, redo: redo, reset: reset };
    textarea._undoMgr = mgr;
    return mgr;
  }

  // Insert text at the current cursor position in a textarea
  function insertAtCursor(textarea, text) {
    if (!textarea) return;
    if (textarea._undoMgr) textarea._undoMgr.checkpoint();
    var start = textarea.selectionStart;
    var end = textarea.selectionEnd;
    var value = textarea.value;
    textarea.value = value.substring(0, start) + text + value.substring(end);
    textarea.selectionStart = textarea.selectionEnd = start + text.length;
    textarea.focus();
  }

  // Wrap the selected text with before/after strings
  function wrapSelection(textarea, before, after) {
    if (!textarea) return;
    if (textarea._undoMgr) textarea._undoMgr.checkpoint();
    var start = textarea.selectionStart;
    var end = textarea.selectionEnd;
    var selectedText = textarea.value.substring(start, end);
    var replacement = before + selectedText + after;
    textarea.value = textarea.value.substring(0, start) + replacement + textarea.value.substring(end);
    textarea.selectionStart = start + before.length;
    textarea.selectionEnd = start + before.length + selectedText.length;
    textarea.focus();
  }

  // Client-side fallback for code fence rendering
  function convertFences(html) {
    return html.replace(/```(\w+)?\s*([\s\S]*?)```/g, function(m, lang, code) {
      var l = (lang || '').trim();
      var c = code.trim().replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
      return '<pre><code class="language-' + l + '">' + c + '</code></pre>';
    });
  }

  function cutAtMore(html) {
    var markers = ['<more-->', '&lt;more--&gt;'];
    for (var i = 0; i < markers.length; i++) {
      var idx = html.indexOf(markers[i]);
      if (idx !== -1) return html.slice(0, idx);
    }
    return html;
  }

  function removeMore(html) {
    return html.replaceAll('<more-->','').replaceAll('&lt;more--&gt;','');
  }

  // Fetch rendered preview from the server
  async function renderPreview(content, fullContent) {
    if (fullContent) {
      content = removeMore(content);
    } else {
      content = cutAtMore(content);
    }

    var container = document.getElementById(previewId);
    if (!container) return;

    var endpoint = cfg.previewEndpoint;
    if (!endpoint) {
      container.innerHTML = convertFences(content);
      return;
    }

    try {
      var body = new URLSearchParams();
      body.set('content', content);
      var res = await fetch(endpoint, {
        method: 'POST',
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: body.toString()
      });
      var j = res.ok ? await res.json() : { html: convertFences(content) };
      container.innerHTML = j.html || convertFences(content);
    } catch(e) {
      container.innerHTML = convertFences(content);
    }

    // Enhance with Prism if available
    if (window.Prism) Prism.highlightAllUnder(container);
  }

  // Bind toolbar buttons on DOMContentLoaded
  document.addEventListener('DOMContentLoaded', function() {
    var editor = getEditor();
    if (!editor || editor.tagName !== 'TEXTAREA') return;

    // Undo/redo for the main editor textarea
    var undoMgr = createUndoManager(editor, 10);

    // Toolbar button handlers
    var actions = {
      'gw-undo':        function() { undoMgr.undo(); },
      'gw-redo':        function() { undoMgr.redo(); },
      'gw-bold':        function() { wrapSelection(editor, '**', '**'); },
      'gw-italic':      function() { wrapSelection(editor, '_', '_'); },
      'gw-h2':          function() { insertAtCursor(editor, '\n## '); },
      'gw-h3':          function() { insertAtCursor(editor, '\n### '); },
      'gw-ul':          function() { insertAtCursor(editor, '\n- '); },
      'gw-ol':          function() { insertAtCursor(editor, '\n1. '); },
      'gw-blockquote':  function() { insertAtCursor(editor, '\n> '); },
      'gw-hr':          function() { insertAtCursor(editor, '\n---\n'); },
      'gw-code':        function() { wrapSelection(editor, '`', '`'); },
      'gw-link':        function() {
        var url = prompt('Enter URL:');
        var text = prompt('Enter link text:');
        if (url && text) insertAtCursor(editor, '[' + text + '](' + url + ')');
      },
      'gw-draw':        function() {
        var drawBase = cfg.drawBasePath;
        if (!drawBase) return;
        var btn = document.getElementById('gw-draw');
        if (btn) btn.disabled = true;
        fetch(drawBase + '/api/new', { method: 'POST' })
          .then(function(res) { return res.json(); })
          .then(function(data) {
            if (data && data.id) {
              insertAtCursor(editor, '\n[draw:' + data.id + ':edit]\n');
            }
          })
          .catch(function(err) {
            alert('Failed to create drawing: ' + err.message);
          })
          .finally(function() {
            if (btn) btn.disabled = false;
          });
      }
    };

    Object.keys(actions).forEach(function(id) {
      var btn = document.getElementById(id);
      if (btn) btn.addEventListener('click', actions[id]);
    });

    // Edit/Preview tabs
    var tabEdit = document.getElementById('gw-tab-edit');
    var tabPrev = document.getElementById('gw-tab-preview');
    var previewFull = document.getElementById('gw-preview-full');
    var editorEl = editor;
    var previewEl = document.getElementById('gw-preview');

    if (tabEdit) {
      tabEdit.addEventListener('click', function() {
        tabEdit.classList.add('active');
        if (tabPrev) tabPrev.classList.remove('active');
        editorEl.classList.remove('gw-hidden');
        if (previewEl) previewEl.classList.add('gw-hidden');
      });
    }

    if (tabPrev) {
      tabPrev.addEventListener('click', function() {
        tabPrev.classList.add('active');
        if (tabEdit) tabEdit.classList.remove('active');
        editorEl.classList.add('gw-hidden');
        if (previewEl) previewEl.classList.remove('gw-hidden');
        renderPreview(editor.value, previewFull && previewFull.checked);
      });
    }

    if (previewFull) {
      previewFull.addEventListener('change', function() {
        if (previewEl && !previewEl.classList.contains('gw-hidden')) {
          renderPreview(editor.value, previewFull.checked);
        }
      });
    }
  });

  // Expose for use by fullscreen module
  window.__goWiki = {
    insertAtCursor: insertAtCursor,
    wrapSelection: wrapSelection,
    convertFences: convertFences,
    cutAtMore: cutAtMore,
    removeMore: removeMore,
    renderPreview: renderPreview,
    createUndoManager: createUndoManager
  };
})();
