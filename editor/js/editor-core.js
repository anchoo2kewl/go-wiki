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
      initDrawEmbeds(container);
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

    initDrawEmbeds(container);

    // Enhance with Prism if available
    if (window.Prism) Prism.highlightAllUnder(container);
  }

  // ---------------------------------------------------------------------------
  // Image upload (visible zone + drag fallback on textarea wrap)
  // ---------------------------------------------------------------------------
  function uploadFiles(files, textarea, uploadEndpoint) {
    for (var i = 0; i < files.length; i++) {
      (function(file) {
        if (!file.type.startsWith('image/')) return;
        var fd = new FormData();
        fd.append('file', file);
        fetch(uploadEndpoint, { method: 'POST', body: fd })
          .then(function(res) { return res.json(); })
          .then(function(data) {
            if (data && data.url) {
              var name = file.name.replace(/\.[^.]+$/, '').replace(/[-_]/g, ' ');
              insertAtCursor(textarea, '![' + name + '](' + data.url + ')\n');
            }
          })
          .catch(function(err) {
            console.error('Image upload failed:', err);
          });
      })(files[i]);
    }
  }

  function setupUploadZone(textarea) {
    var uploadEndpoint = cfg.uploadEndpoint;
    if (!uploadEndpoint) return;

    var zone = textarea.closest('.gw-editor-container')
      ? textarea.closest('.gw-editor-container').querySelector('.gw-upload-zone')
      : (textarea.parentElement && textarea.parentElement.nextElementSibling && textarea.parentElement.nextElementSibling.classList.contains('gw-upload-zone')
        ? textarea.parentElement.nextElementSibling
        : document.getElementById('gw-upload-zone'));
    var fileInput = zone && zone.querySelector('input[type="file"]');
    var wrap = textarea.closest('.gw-editor-textarea-wrap') || textarea.closest('.gw-fullscreen-textarea-wrap') || textarea.parentElement;
    var dragCounter = 0;

    // Click zone → trigger file input
    if (zone && fileInput) {
      zone.addEventListener('click', function() { fileInput.click(); });
      fileInput.addEventListener('change', function() {
        if (fileInput.files && fileInput.files.length > 0) {
          uploadFiles(fileInput.files, textarea, uploadEndpoint);
          fileInput.value = '';
        }
      });
    }

    // Drag events on the upload zone
    if (zone) {
      zone.addEventListener('dragenter', function(e) {
        e.preventDefault();
        dragCounter++;
        if (e.dataTransfer && e.dataTransfer.types.indexOf('Files') !== -1) {
          zone.classList.add('gw-drag-over');
        }
      });
      zone.addEventListener('dragover', function(e) {
        e.preventDefault();
        if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy';
      });
      zone.addEventListener('dragleave', function(e) {
        e.preventDefault();
        dragCounter--;
        if (dragCounter <= 0) { dragCounter = 0; zone.classList.remove('gw-drag-over'); }
      });
      zone.addEventListener('drop', function(e) {
        e.preventDefault();
        dragCounter = 0;
        zone.classList.remove('gw-drag-over');
        var files = e.dataTransfer && e.dataTransfer.files;
        if (files && files.length > 0) uploadFiles(files, textarea, uploadEndpoint);
      });
    }

    // Fallback: drag onto textarea wrap still works
    var wrapDragCounter = 0;
    wrap.addEventListener('dragenter', function(e) {
      e.preventDefault();
      wrapDragCounter++;
      if (e.dataTransfer && e.dataTransfer.types.indexOf('Files') !== -1 && zone) {
        zone.classList.add('gw-drag-over');
      }
    });
    wrap.addEventListener('dragover', function(e) {
      e.preventDefault();
      if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy';
    });
    wrap.addEventListener('dragleave', function(e) {
      e.preventDefault();
      wrapDragCounter--;
      if (wrapDragCounter <= 0) { wrapDragCounter = 0; if (zone) zone.classList.remove('gw-drag-over'); }
    });
    wrap.addEventListener('drop', function(e) {
      e.preventDefault();
      wrapDragCounter = 0;
      if (zone) zone.classList.remove('gw-drag-over');
      var files = e.dataTransfer && e.dataTransfer.files;
      if (files && files.length > 0) uploadFiles(files, textarea, uploadEndpoint);
    });
  }

  // ---------------------------------------------------------------------------
  // Double-click draw shortcode → open editor
  // ---------------------------------------------------------------------------
  function setupDrawShortcodeClick(textarea) {
    var drawBase = cfg.drawBasePath;
    if (!drawBase) return;

    textarea.addEventListener('dblclick', function() {
      var pos = textarea.selectionStart;
      var val = textarea.value;
      // Find the shortcode surrounding the cursor
      var re = /\[draw:([a-zA-Z0-9_-]+)(?::edit)?\]/g;
      var match;
      while ((match = re.exec(val)) !== null) {
        if (pos >= match.index && pos <= match.index + match[0].length) {
          var drawId = match[1];
          window.open(drawBase + '/' + drawId + '/edit', '_blank');
          return;
        }
      }
    });
  }

  // ---------------------------------------------------------------------------
  // Draw embed init — makes [draw:id:edit] previews work after innerHTML set
  // Browsers ignore <script> tags injected via innerHTML, so embed.js never
  // runs.  This scans for .godraw-embed divs and creates iframes directly.
  // ---------------------------------------------------------------------------
  function initDrawEmbeds(container) {
    if (!container) return;
    var embeds = container.querySelectorAll('.godraw-embed:not(.godraw-preview-init)');
    for (var i = 0; i < embeds.length; i++) {
      var div = embeds[i];
      var src = div.getAttribute('data-src');
      var w = div.getAttribute('data-width') || '100%';
      var h = div.getAttribute('data-height') || '400px';
      if (!src) continue;
      var iframe = document.createElement('iframe');
      iframe.src = src;
      iframe.style.width = w;
      iframe.style.height = h;
      iframe.style.border = 'none';
      iframe.style.borderRadius = '8px';
      iframe.setAttribute('loading', 'lazy');
      div.innerHTML = '';
      div.appendChild(iframe);
      div.classList.add('godraw-preview-init');
    }
  }

  // ---------------------------------------------------------------------------
  // Image browser modal
  // ---------------------------------------------------------------------------
  function setupImageBrowser(editor) {
    var listEndpoint = cfg.imageListEndpoint;
    if (!listEndpoint) return;

    var modal = document.getElementById('gw-image-modal');
    var grid = document.getElementById('gw-image-grid');
    var closeBtn = document.getElementById('gw-image-modal-close');
    var backdrop = modal && modal.querySelector('.gw-modal-backdrop');
    if (!modal || !grid) return;

    function closeModal() {
      modal.classList.add('gw-hidden');
    }
    if (closeBtn) closeBtn.addEventListener('click', closeModal);
    if (backdrop) backdrop.addEventListener('click', closeModal);

    function openModal() {
      modal.classList.remove('gw-hidden');
      grid.innerHTML = '<div class="gw-img-loading">Loading images...</div>';

      fetch(listEndpoint)
        .then(function(res) { return res.json(); })
        .then(function(data) {
          var images = data.images || [];
          if (images.length === 0) {
            grid.innerHTML = '<div class="gw-img-empty">No images uploaded yet</div>';
            return;
          }
          grid.innerHTML = '';
          images.forEach(function(img) {
            var card = document.createElement('div');
            card.className = 'gw-img-card';
            card.innerHTML = '<img src="' + img.url + '" alt="' + (img.filename || '') + '" loading="lazy"/>' +
              '<span class="gw-img-name">' + (img.filename || '') + '</span>' +
              '<button type="button" class="gw-img-delete" title="Delete image">&times;</button>';

            // Click card → insert into editor
            card.addEventListener('click', function(e) {
              if (e.target.classList.contains('gw-img-delete')) return;
              var name = (img.filename || '').replace(/\.[^.]+$/, '').replace(/[-_]/g, ' ');
              insertAtCursor(editor, '![' + name + '](' + img.url + ')\n');
              closeModal();
            });

            // Delete button
            var delBtn = card.querySelector('.gw-img-delete');
            if (delBtn) {
              delBtn.addEventListener('click', function(e) {
                e.stopPropagation();
                if (!confirm('Delete this image?')) return;
                fetch(listEndpoint.replace('/list', '') + '?path=' + encodeURIComponent(img.url), { method: 'DELETE' })
                  .then(function(res) {
                    if (res.ok) {
                      card.remove();
                    } else {
                      alert('Failed to delete image');
                    }
                  })
                  .catch(function() { alert('Failed to delete image'); });
              });
            }
            grid.appendChild(card);
          });
        })
        .catch(function() {
          grid.innerHTML = '<div class="gw-img-empty">Failed to load images</div>';
        });
    }

    return openModal;
  }

  // ---------------------------------------------------------------------------
  // Draw browser modal
  // ---------------------------------------------------------------------------
  function setupDrawBrowser(editor) {
    var drawBase = cfg.drawBasePath;
    if (!drawBase) return;

    var modal = document.getElementById('gw-draw-modal');
    var grid = document.getElementById('gw-draw-grid');
    var closeBtn = document.getElementById('gw-draw-modal-close');
    var newBtn = document.getElementById('gw-draw-modal-new');
    var backdrop = modal && modal.querySelector('.gw-modal-backdrop');
    if (!modal || !grid) return;

    function closeModal() {
      modal.classList.add('gw-hidden');
    }
    if (closeBtn) closeBtn.addEventListener('click', closeModal);
    if (backdrop) backdrop.addEventListener('click', closeModal);

    function formatDate(iso) {
      try {
        var d = new Date(iso);
        return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
      } catch(e) { return iso; }
    }

    function getUsedIds() {
      var content = editor.value || '';
      var used = {};
      var re = /\[draw:([a-zA-Z0-9_-]+)/g;
      var m;
      while ((m = re.exec(content)) !== null) used[m[1]] = true;
      return used;
    }

    function loadDrawings() {
      // Remove any existing cleanup bar
      var oldCleanup = modal.querySelector('.gw-draw-cleanup');
      if (oldCleanup) oldCleanup.remove();
      grid.innerHTML = '<div class="gw-draw-loading">Loading drawings...</div>';
      fetch(drawBase + '/api/list')
        .then(function(res) { return res.json(); })
        .then(function(data) {
          var drawings = data.drawings || [];
          if (drawings.length === 0) {
            grid.innerHTML = '<div class="gw-draw-empty">No drawings yet. Create one above!</div>';
            return;
          }
          grid.innerHTML = '';
          var usedIds = getUsedIds();
          var unusedIds = [];

          drawings.forEach(function(drw) {
            var isUsed = !!usedIds[drw.id];
            if (!isUsed) unusedIds.push(drw);

            var card = document.createElement('div');
            card.className = 'gw-draw-card' + (isUsed ? ' gw-draw-card-used' : '');

            var titleRow = document.createElement('div');
            titleRow.className = 'gw-draw-card-title-row';

            var title = document.createElement('div');
            title.className = 'gw-draw-card-title';
            title.textContent = drw.title || 'Untitled';
            title.contentEditable = 'true';
            title.spellcheck = false;

            title.addEventListener('blur', function() {
              var newTitle = title.textContent.trim();
              if (!newTitle || newTitle === drw.title) {
                title.textContent = drw.title || 'Untitled';
                return;
              }
              fetch(drawBase + '/api/' + drw.id + '/rename', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ title: newTitle })
              }).then(function() { drw.title = newTitle; })
                .catch(function() { title.textContent = drw.title || 'Untitled'; });
            });

            title.addEventListener('keydown', function(e) {
              if (e.key === 'Enter') { e.preventDefault(); title.blur(); }
              if (e.key === 'Escape') { title.textContent = drw.title || 'Untitled'; title.blur(); }
            });

            titleRow.appendChild(title);

            if (isUsed) {
              var badge = document.createElement('span');
              badge.className = 'gw-draw-badge-used';
              badge.textContent = 'in use';
              titleRow.appendChild(badge);
            }

            var meta = document.createElement('div');
            meta.className = 'gw-draw-card-meta';
            meta.textContent = formatDate(drw.updated_at);

            var actions = document.createElement('div');
            actions.className = 'gw-draw-card-actions';

            var insertBtn = document.createElement('button');
            insertBtn.type = 'button';
            insertBtn.className = 'gw-draw-action-btn gw-draw-action-insert';
            insertBtn.textContent = 'Insert';
            insertBtn.addEventListener('click', function() {
              insertAtCursor(editor, '\n[draw:' + drw.id + ':edit]\n');
              closeModal();
            });

            var editBtn = document.createElement('button');
            editBtn.type = 'button';
            editBtn.className = 'gw-draw-action-btn gw-draw-action-edit';
            editBtn.textContent = 'Edit';
            editBtn.addEventListener('click', function() {
              window.open(drawBase + '/' + drw.id + '/edit', '_blank');
            });

            var delBtn = document.createElement('button');
            delBtn.type = 'button';
            delBtn.className = 'gw-draw-action-btn gw-draw-action-delete';
            delBtn.textContent = 'Delete';
            delBtn.addEventListener('click', function() {
              if (!confirm('Delete "' + (drw.title || 'Untitled') + '"?')) return;
              fetch(drawBase + '/api/' + drw.id + '/delete', { method: 'POST' })
                .then(function(res) {
                  if (res.ok) card.remove();
                  else alert('Failed to delete drawing');
                })
                .catch(function() { alert('Failed to delete drawing'); });
            });

            actions.appendChild(insertBtn);
            actions.appendChild(editBtn);
            actions.appendChild(delBtn);
            card.appendChild(titleRow);
            card.appendChild(meta);
            card.appendChild(actions);
            grid.appendChild(card);
          });

          // Cleanup bar for unused drawings
          if (unusedIds.length > 0) {
            var cleanupBar = document.createElement('div');
            cleanupBar.className = 'gw-draw-cleanup';
            cleanupBar.innerHTML = '<span>' + unusedIds.length + ' unused drawing' + (unusedIds.length > 1 ? 's' : '') + '</span>';
            var cleanupBtn = document.createElement('button');
            cleanupBtn.type = 'button';
            cleanupBtn.className = 'gw-draw-action-btn gw-draw-action-delete';
            cleanupBtn.textContent = 'Delete all unused';
            cleanupBtn.addEventListener('click', function() {
              if (!confirm('Delete ' + unusedIds.length + ' unused drawing(s)? This cannot be undone.')) return;
              cleanupBtn.disabled = true;
              cleanupBtn.textContent = 'Deleting...';
              Promise.all(unusedIds.map(function(drw) {
                return fetch(drawBase + '/api/' + drw.id + '/delete', { method: 'POST' });
              })).then(function() {
                loadDrawings();
              }).catch(function() {
                alert('Some deletions failed');
                loadDrawings();
              });
            });
            cleanupBar.appendChild(cleanupBtn);
            grid.parentNode.insertBefore(cleanupBar, grid);
          }
        })
        .catch(function() {
          grid.innerHTML = '<div class="gw-draw-empty">Failed to load drawings</div>';
        });
    }

    if (newBtn) {
      newBtn.addEventListener('click', function() {
        newBtn.disabled = true;
        fetch(drawBase + '/api/new', { method: 'POST' })
          .then(function(res) { return res.json(); })
          .then(function(data) {
            if (data && data.id) {
              var editUrl = data.edit_url || (drawBase + '/' + data.id + '/edit');
              window.open(editUrl, '_blank');
              loadDrawings();
            }
          })
          .catch(function(err) { alert('Failed to create drawing: ' + err.message); })
          .finally(function() { newBtn.disabled = false; });
      });
    }

    function openModal() {
      modal.classList.remove('gw-hidden');
      loadDrawings();
    }

    return openModal;
  }

  // Bind toolbar buttons on DOMContentLoaded
  document.addEventListener('DOMContentLoaded', function() {
    var editor = getEditor();
    if (!editor || editor.tagName !== 'TEXTAREA') return;

    // Undo/redo for the main editor textarea
    var undoMgr = createUndoManager(editor, 10);

    // Set up visible upload zone + drag fallback
    setupUploadZone(editor);

    // Set up double-click to open draw shortcodes
    setupDrawShortcodeClick(editor);

    // Set up image browser
    var openImageBrowser = setupImageBrowser(editor);

    // Set up draw browser
    var openDrawBrowser = setupDrawBrowser(editor);

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
      'gw-images':      function() {
        if (openImageBrowser) openImageBrowser();
      },
      'gw-more':        function() {
        insertAtCursor(editor, '\n<more-->\n');
      },
      'gw-draw':        function() {
        if (openDrawBrowser) openDrawBrowser();
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
    var editorWrap = document.getElementById('gw-editor-textarea-wrap') || editor;
    var uploadZone = document.getElementById('gw-upload-zone');
    var previewEl = document.getElementById('gw-preview');

    if (tabEdit) {
      tabEdit.addEventListener('click', function() {
        tabEdit.classList.add('active');
        if (tabPrev) tabPrev.classList.remove('active');
        editorWrap.classList.remove('gw-hidden');
        if (uploadZone) uploadZone.classList.remove('gw-hidden');
        if (previewEl) previewEl.classList.add('gw-hidden');
      });
    }

    if (tabPrev) {
      tabPrev.addEventListener('click', function() {
        tabPrev.classList.add('active');
        if (tabEdit) tabEdit.classList.remove('active');
        editorWrap.classList.add('gw-hidden');
        if (uploadZone) uploadZone.classList.add('gw-hidden');
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
    createUndoManager: createUndoManager,
    setupUploadZone: setupUploadZone,
    setupDrawShortcodeClick: setupDrawShortcodeClick,
    initDrawEmbeds: initDrawEmbeds
  };
})();
