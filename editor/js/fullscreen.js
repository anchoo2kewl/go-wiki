// go-wiki fullscreen split-view editor
(function() {
  'use strict';

  var cfg = window.__goWikiConfig || {};
  var gw = window.__goWiki || {};

  var fsOverlay, fsTextarea, fsPreview, fsStatus, fsTitleEl, fsFullCheck, fsExitBtn, fsBtn, fsDivider, mainEditor;
  var fsActive = false;
  var fsUndoMgr = null;
  var renderTimer = null;
  var renderAbort = null;
  var DEBOUNCE_MS = 350;

  function init() {
    fsOverlay   = document.getElementById('gw-fullscreen-editor');
    fsTextarea  = document.getElementById('gw-fullscreen-textarea');
    fsPreview   = document.getElementById('gw-fullscreen-preview-content');
    fsStatus    = document.getElementById('gw-fullscreen-preview-status');
    fsTitleEl   = document.getElementById('gw-fullscreen-post-title');
    fsFullCheck = document.getElementById('gw-fullscreen-preview-full');
    fsExitBtn   = document.getElementById('gw-fullscreen-exit');
    fsBtn       = document.getElementById('gw-fullscreen-btn');
    fsDivider   = document.getElementById('gw-fullscreen-divider');
    mainEditor  = document.getElementById(cfg.textareaId || 'gw-editor');

    if (!fsOverlay || !fsTextarea || !mainEditor) return;

    // Undo/redo for the fullscreen textarea
    if (gw.createUndoManager) {
      fsUndoMgr = gw.createUndoManager(fsTextarea, 10);
    }

    // Set up upload zone for fullscreen textarea
    if (gw.setupUploadZone) {
      gw.setupUploadZone(fsTextarea);
    }

    // Set up double-click to open draw shortcodes in fullscreen
    if (gw.setupDrawShortcodeClick) {
      gw.setupDrawShortcodeClick(fsTextarea);
    }

    fsBtn && fsBtn.addEventListener('click', enterFullscreen);
    fsExitBtn && fsExitBtn.addEventListener('click', exitFullscreen);

    document.addEventListener('keydown', function(e) {
      if (e.key === 'Escape' && fsActive) {
        exitFullscreen();
        e.preventDefault();
        e.stopPropagation();
      }
      if (e.key === 'F11' && !document.querySelector('.fixed:not(.hidden):not(.gw-hidden)')) {
        e.preventDefault();
        if (fsActive) exitFullscreen(); else enterFullscreen();
      }
    });

    fsTextarea.addEventListener('input', scheduleRender);
    fsTextarea.addEventListener('input', function() {
      mainEditor.value = fsTextarea.value;
    });

    if (fsFullCheck) {
      fsFullCheck.addEventListener('change', function() {
        if (fsActive) scheduleRender();
      });
    }

    // Fullscreen toolbar buttons
    var fsToolbarBtns = fsOverlay.querySelectorAll('[data-fs]');
    fsToolbarBtns.forEach(function(btn) {
      btn.addEventListener('click', function() {
        var cmd = btn.getAttribute('data-fs');
        handleFsToolbar(cmd);
      });
    });

    // Resizable divider
    if (fsDivider) {
      var isDragging = false;
      fsDivider.addEventListener('mousedown', function(e) {
        isDragging = true;
        fsDivider.classList.add('dragging');
        document.body.style.cursor = 'col-resize';
        document.body.style.userSelect = 'none';
        e.preventDefault();
      });
      document.addEventListener('mousemove', function(e) {
        if (!isDragging) return;
        var split = document.querySelector('.gw-fullscreen-split');
        var rect = split.getBoundingClientRect();
        var pct = ((e.clientX - rect.left) / rect.width) * 100;
        var clamped = Math.min(Math.max(pct, 20), 80);
        var editorPane = document.querySelector('.gw-fullscreen-pane-editor');
        var previewPane = document.querySelector('.gw-fullscreen-pane-preview');
        editorPane.style.flex = 'none';
        editorPane.style.width = clamped + '%';
        previewPane.style.flex = 'none';
        previewPane.style.width = (100 - clamped) + '%';
      });
      document.addEventListener('mouseup', function() {
        if (!isDragging) return;
        isDragging = false;
        fsDivider.classList.remove('dragging');
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
      });
    }
  }

  function handleFsToolbar(cmd) {
    if (!fsTextarea) return;
    switch (cmd) {
      case 'bold':    gw.wrapSelection(fsTextarea, '**', '**'); break;
      case 'italic':  gw.wrapSelection(fsTextarea, '_', '_'); break;
      case 'h2':      gw.insertAtCursor(fsTextarea, '\n## '); break;
      case 'h3':      gw.insertAtCursor(fsTextarea, '\n### '); break;
      case 'ul':      gw.insertAtCursor(fsTextarea, '\n- '); break;
      case 'ol':      gw.insertAtCursor(fsTextarea, '\n1. '); break;
      case 'quote':   gw.insertAtCursor(fsTextarea, '\n> '); break;
      case 'hr':      gw.insertAtCursor(fsTextarea, '\n---\n'); break;
      case 'code':    gw.wrapSelection(fsTextarea, '`', '`'); break;
      case 'link':
        var url = prompt('Enter URL:');
        var text = prompt('Enter link text:');
        if (url && text) gw.insertAtCursor(fsTextarea, '[' + text + '](' + url + ')');
        break;
      case 'images':
        // Open the image modal from the main editor setup
        var imgBtn = document.getElementById('gw-images');
        if (imgBtn) imgBtn.click();
        break;
      case 'more':
        gw.insertAtCursor(fsTextarea, '\n<more-->\n');
        break;
      case 'draw':
        // Open the draw browser modal from the main editor setup
        var drawBtn = document.getElementById('gw-draw');
        if (drawBtn) drawBtn.click();
        break;
      case 'edit-img':
        if (gw.setupEditImage) {
          var openEditImg = gw.setupEditImage(fsTextarea);
          openEditImg();
        }
        break;
    }
    // Sync to main editor
    mainEditor.value = fsTextarea.value;
    scheduleRender();
  }

  function enterFullscreen() {
    if (mainEditor._undoMgr) mainEditor._undoMgr.checkpoint();
    fsTextarea.value = mainEditor.value;
    if (fsUndoMgr) fsUndoMgr.reset();
    if (fsTitleEl) {
      var titleInput = document.querySelector('input[name="title"]');
      fsTitleEl.textContent = (titleInput && titleInput.value) ? titleInput.value : 'Editing';
    }
    fsOverlay.classList.remove('gw-hidden');
    document.body.style.overflow = 'hidden';
    fsActive = true;
    fsTextarea.focus();
    scheduleRender();
  }

  function exitFullscreen() {
    if (!fsActive) return;
    mainEditor.value = fsTextarea.value;
    fsOverlay.classList.add('gw-hidden');
    document.body.style.overflow = '';
    fsActive = false;
    if (renderTimer) clearTimeout(renderTimer);
    if (renderAbort) renderAbort.abort();
  }

  function scheduleRender() {
    if (renderTimer) clearTimeout(renderTimer);
    if (fsStatus) {
      fsStatus.textContent = 'Typing...';
      fsStatus.classList.remove('rendering');
    }
    renderTimer = setTimeout(doRender, DEBOUNCE_MS);
  }

  async function doRender() {
    var content = fsTextarea.value;
    if (!content.trim()) {
      fsPreview.innerHTML = '<div class="gw-preview-empty"><svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg><span>Start typing to see a live preview</span></div>';
      if (gw.initDrawEmbeds) gw.initDrawEmbeds(fsPreview);
      if (fsStatus) { fsStatus.textContent = 'Live'; fsStatus.classList.remove('rendering'); }
      return;
    }

    content = (fsFullCheck && fsFullCheck.checked) ? gw.removeMore(content) : gw.cutAtMore(content);

    if (renderAbort) renderAbort.abort();
    renderAbort = new AbortController();

    if (fsStatus) { fsStatus.textContent = 'Rendering...'; fsStatus.classList.add('rendering'); }

    var endpoint = cfg.previewEndpoint;
    try {
      if (!endpoint) throw new Error('no endpoint');
      var body = new URLSearchParams();
      body.set('content', content);
      var res = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: body.toString(),
        signal: renderAbort.signal
      });
      var j = res.ok ? await res.json() : { html: gw.convertFences(content) };
      fsPreview.innerHTML = j.html || gw.convertFences(content);
    } catch (e) {
      if (e.name === 'AbortError') return;
      fsPreview.innerHTML = gw.convertFences(content);
    }

    if (gw.initDrawEmbeds) gw.initDrawEmbeds(fsPreview);
    if (gw.initImageEditOverlays) gw.initImageEditOverlays(fsPreview);
    if (window.Prism) Prism.highlightAllUnder(fsPreview);
    if (fsStatus) { fsStatus.textContent = 'Live'; fsStatus.classList.remove('rendering'); }
  }

  document.addEventListener('DOMContentLoaded', init);
})();
