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
  // Image markup builder (shared by image browser + drag-drop)
  // ---------------------------------------------------------------------------
  function buildImageMarkup(url, alt, caption, size) {
    alt = alt || '';
    caption = (caption || '').trim();
    size = size || 'm';

    var sizeStyles = {
      s: 'max-width:50%; height:auto;',
      m: 'max-width:75%; height:auto;',
      l: 'width:100%; height:auto; max-width:100%;'
    };
    var imgStyle = sizeStyles[size] || sizeStyles.m;

    // Large + no caption → plain markdown
    if (size === 'l' && !caption) {
      return '![' + alt + '](' + url + ')\n';
    }

    // Otherwise use <figure>
    var escapedAlt = alt.replace(/"/g, '&quot;');
    return '\n<figure style="text-align:center; margin: 1.5rem 0;">\n' +
      '  <a href="' + url + '" data-lightbox="article-images" data-title="' + escapedAlt + '">\n' +
      '    <img src="' + url + '" alt="' + escapedAlt + '" style="' + imgStyle + '" />\n' +
      '  </a>\n' +
      (caption ? '  <figcaption>' + caption + '</figcaption>\n' : '') +
      '</figure>\n';
  }

  // ---------------------------------------------------------------------------
  // Cloudinary upload helper
  // ---------------------------------------------------------------------------
  function uploadToCloudinary(file, folder) {
    var signatureEndpoint = cfg.cloudinarySignatureEndpoint;
    var cloudName = cfg.cloudinaryCloudName;
    if (!signatureEndpoint || !cloudName) return Promise.reject(new Error('Cloudinary not configured'));

    return fetch(signatureEndpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ folder: folder || 'blog/images' })
    })
    .then(function(res) { return res.json(); })
    .then(function(sig) {
      var fd = new FormData();
      fd.append('file', file);
      fd.append('api_key', sig.api_key);
      fd.append('timestamp', sig.timestamp);
      fd.append('signature', sig.signature);
      if (folder) fd.append('folder', folder);
      return fetch('https://api.cloudinary.com/v1_1/' + cloudName + '/auto/upload', {
        method: 'POST',
        body: fd
      });
    })
    .then(function(res) { return res.json(); })
    .then(function(data) {
      if (data.secure_url) return data.secure_url;
      if (data.url) return data.url;
      throw new Error(data.error ? data.error.message : 'Upload failed');
    });
  }

  // Save image metadata to backend
  function saveImageMetadata(url, alt, caption) {
    var endpoint = cfg.imageMetadataEndpoint;
    if (!endpoint) return Promise.resolve();
    return fetch(endpoint, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ image_url: url, alt_text: alt || '', title: '', caption: caption || '' })
    });
  }

  function isCloudinaryConfigured() {
    return !!(cfg.cloudinarySignatureEndpoint && cfg.cloudinaryCloudName);
  }

  // ---------------------------------------------------------------------------
  // Image upload (visible zone + drag fallback on textarea wrap)
  // ---------------------------------------------------------------------------
  function uploadFiles(files, textarea, uploadEndpoint) {
    for (var i = 0; i < files.length; i++) {
      (function(file) {
        if (!file.type.startsWith('image/')) return;

        // If Cloudinary configured, show inline toast for alt text before uploading
        if (isCloudinaryConfigured()) {
          showInlineUploadToast(file, textarea);
          return;
        }

        // Fallback: local upload
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

  // Show an inline toast near the textarea for entering alt text before Cloudinary upload
  function showInlineUploadToast(file, textarea) {
    // Remove any existing toast
    var existing = document.querySelector('.gw-inline-upload-toast');
    if (existing) existing.remove();

    var wrap = textarea.closest('.gw-editor-textarea-wrap') || textarea.closest('.gw-fullscreen-textarea-wrap') || textarea.parentElement;
    wrap.style.position = 'relative';

    var toast = document.createElement('div');
    toast.className = 'gw-inline-upload-toast';

    var defaultAlt = file.name.replace(/\.[^.]+$/, '').replace(/[-_]/g, ' ');

    toast.innerHTML =
      '<div class="gw-inline-upload-toast-row">' +
        '<img class="gw-inline-upload-toast-thumb" src="" alt="Preview" />' +
        '<div class="gw-inline-upload-toast-fields">' +
          '<label class="gw-img-field-label">Alt text</label>' +
          '<input type="text" class="gw-img-input" id="gw-toast-alt" value="' + defaultAlt.replace(/"/g, '&quot;') + '" />' +
          '<label class="gw-img-field-label">Caption <span style="color:#9ca3af">(optional)</span></label>' +
          '<input type="text" class="gw-img-input" id="gw-toast-caption" placeholder="Caption" />' +
          '<label class="gw-img-field-label">Size</label>' +
          '<select class="gw-draw-select" id="gw-toast-size">' +
            '<option value="s">Small</option>' +
            '<option value="m" selected>Medium</option>' +
            '<option value="l">Large</option>' +
          '</select>' +
        '</div>' +
      '</div>' +
      '<div class="gw-inline-upload-toast-actions">' +
        '<button type="button" class="gw-img-btn-cancel" id="gw-toast-cancel">Cancel</button>' +
        '<button type="button" class="gw-img-btn-insert" id="gw-toast-upload">Upload &amp; Insert</button>' +
      '</div>';

    wrap.appendChild(toast);

    // Show file preview
    var thumb = toast.querySelector('.gw-inline-upload-toast-thumb');
    var reader = new FileReader();
    reader.onload = function(e) { thumb.src = e.target.result; };
    reader.readAsDataURL(file);

    var altInput = toast.querySelector('#gw-toast-alt');
    altInput.focus();
    altInput.select();

    toast.querySelector('#gw-toast-cancel').addEventListener('click', function() { toast.remove(); });
    toast.querySelector('#gw-toast-upload').addEventListener('click', function() {
      var alt = altInput.value.trim();
      var caption = toast.querySelector('#gw-toast-caption').value.trim();
      var size = (toast.querySelector('#gw-toast-size') || {}).value || 'm';
      var submitBtn = toast.querySelector('#gw-toast-upload');
      submitBtn.disabled = true;
      submitBtn.textContent = 'Uploading...';

      var folder = cfg.cloudinaryFolder || 'blog/images';
      uploadToCloudinary(file, folder)
        .then(function(url) {
          return saveImageMetadata(url, alt, caption).then(function() { return url; });
        })
        .then(function(url) {
          insertAtCursor(textarea, buildImageMarkup(url, alt, caption, size));
          toast.remove();
        })
        .catch(function(err) {
          alert('Upload failed: ' + err.message);
          submitBtn.disabled = false;
          submitBtn.textContent = 'Upload & Insert';
        });
    });
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
      var h = div.getAttribute('data-height') || '520px';
      var zoom = div.getAttribute('data-zoom');
      if (!src) continue;
      // Preview always shows read-only view — strip /edit suffix
      src = src.replace(/\/edit$/, '');
      // Append zoom query param if present
      if (zoom) {
        src += (src.indexOf('?') === -1 ? '?' : '&') + 'zoom=' + encodeURIComponent(zoom);
      }
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
  // Image browser modal (image manager)
  // ---------------------------------------------------------------------------
  function setupImageBrowser(editor) {
    var listEndpoint = cfg.imageListEndpoint;
    if (!listEndpoint) return;

    var modal = document.getElementById('gw-image-modal');
    var grid = document.getElementById('gw-image-grid');
    var closeBtn = document.getElementById('gw-image-modal-close');
    var backdrop = modal && modal.querySelector('.gw-modal-backdrop');
    var tabBar = document.getElementById('gw-img-tabs');
    var browseView = document.getElementById('gw-img-browse-view');
    var uploadView = document.getElementById('gw-img-upload-view');
    var uploadBtn = document.getElementById('gw-img-upload-btn');
    var backBtn = document.getElementById('gw-img-back-btn');
    var insertPanel = document.getElementById('gw-img-insert-panel');
    if (!modal || !grid) return;

    var allImages = [];
    var selectedImage = null;
    var currentTab = 'all';

    function closeModal() {
      modal.classList.add('gw-hidden');
      hideInsertPanel();
      showBrowseView();
    }
    if (closeBtn) closeBtn.addEventListener('click', closeModal);
    if (backdrop) backdrop.addEventListener('click', closeModal);

    // Escape key closes modal
    document.addEventListener('keydown', function(e) {
      if (e.key === 'Escape' && !modal.classList.contains('gw-hidden')) closeModal();
    });

    function showBrowseView() {
      if (browseView) browseView.classList.remove('gw-hidden');
      if (uploadView) uploadView.classList.add('gw-hidden');
    }
    function showUploadView() {
      if (browseView) browseView.classList.add('gw-hidden');
      if (uploadView) uploadView.classList.remove('gw-hidden');
      hideInsertPanel();
      resetUploadForm();
    }

    // --- Tab switching ---
    if (tabBar) {
      var tabs = tabBar.querySelectorAll('.gw-img-tab');
      tabs.forEach(function(tab) {
        tab.addEventListener('click', function() {
          tabs.forEach(function(t) { t.classList.remove('active'); });
          tab.classList.add('active');
          currentTab = tab.getAttribute('data-tab');
          renderGrid();
          hideInsertPanel();
        });
      });
    }

    // --- Upload button ---
    if (uploadBtn) {
      uploadBtn.addEventListener('click', showUploadView);
    }
    if (backBtn) {
      backBtn.addEventListener('click', showBrowseView);
    }

    // --- Insert panel ---
    function showInsertPanel(img) {
      selectedImage = img;
      if (!insertPanel) return;
      insertPanel.classList.remove('gw-hidden');
      var thumb = document.getElementById('gw-img-insert-thumb');
      var altInput = document.getElementById('gw-img-insert-alt');
      var captionInput = document.getElementById('gw-img-insert-caption');
      if (thumb) thumb.src = img.url;
      if (altInput) altInput.value = img.alt_text || img.filename || '';
      if (captionInput) captionInput.value = img.caption || '';
    }

    function hideInsertPanel() {
      selectedImage = null;
      if (insertPanel) insertPanel.classList.add('gw-hidden');
      // Deselect all cards
      grid.querySelectorAll('.gw-img-card.gw-selected').forEach(function(c) { c.classList.remove('gw-selected'); });
    }

    // Insert button
    var insertBtn = document.getElementById('gw-img-insert-btn');
    if (insertBtn) {
      insertBtn.addEventListener('click', function() {
        if (!selectedImage) return;
        var alt = (document.getElementById('gw-img-insert-alt') || {}).value || '';
        var caption = (document.getElementById('gw-img-insert-caption') || {}).value || '';
        var size = (document.getElementById('gw-img-insert-size') || {}).value || 'm';
        // Save updated metadata
        saveImageMetadata(selectedImage.url, alt, caption);
        insertAtCursor(editor, buildImageMarkup(selectedImage.url, alt, caption, size));
        closeModal();
      });
    }

    // Cancel insert
    var insertCancel = document.getElementById('gw-img-insert-cancel');
    if (insertCancel) {
      insertCancel.addEventListener('click', hideInsertPanel);
    }

    // --- Grid rendering ---
    function getPostImageUrls() {
      var content = editor.value || '';
      var urls = {};
      // Match markdown images: ![...](url)
      var re1 = /!\[[^\]]*\]\(([^)]+)\)/g;
      var m;
      while ((m = re1.exec(content)) !== null) urls[m[1]] = true;
      // Match HTML img src="url"
      var re2 = /src="([^"]+)"/g;
      while ((m = re2.exec(content)) !== null) urls[m[1]] = true;
      // Match href="url" (for lightbox links)
      var re3 = /href="(https?:\/\/res\.cloudinary\.com[^"]+)"/g;
      while ((m = re3.exec(content)) !== null) urls[m[1]] = true;
      return urls;
    }

    function renderGrid() {
      grid.innerHTML = '';
      var images = allImages;

      if (currentTab === 'post') {
        var postUrls = getPostImageUrls();
        images = images.filter(function(img) { return postUrls[img.url]; });
      }

      if (images.length === 0) {
        grid.innerHTML = '<div class="gw-img-empty">' +
          (currentTab === 'post' ? 'No images in this post yet' : 'No images uploaded yet') +
          '</div>';
        return;
      }

      images.forEach(function(img) {
        var card = document.createElement('div');
        card.className = 'gw-img-card';
        card.innerHTML = '<img src="' + img.url + '" alt="' + (img.alt_text || img.filename || '') + '" loading="lazy"/>' +
          '<span class="gw-img-name">' + (img.alt_text || img.filename || '') + '</span>' +
          '<button type="button" class="gw-img-delete" title="Delete image">&times;</button>';

        // Click card → select and show insert panel
        card.addEventListener('click', function(e) {
          if (e.target.classList.contains('gw-img-delete')) return;
          // Deselect others
          grid.querySelectorAll('.gw-img-card.gw-selected').forEach(function(c) { c.classList.remove('gw-selected'); });
          card.classList.add('gw-selected');
          showInsertPanel(img);
        });

        // Delete button
        var delBtn = card.querySelector('.gw-img-delete');
        if (delBtn) {
          delBtn.addEventListener('click', function(e) {
            e.stopPropagation();
            if (!confirm('Remove this image from the manager?')) return;
            var metaEndpoint = cfg.imageMetadataEndpoint;
            if (metaEndpoint) {
              fetch(metaEndpoint + '?url=' + encodeURIComponent(img.url), { method: 'DELETE' })
                .then(function(res) {
                  if (res.ok) {
                    card.remove();
                    allImages = allImages.filter(function(i) { return i.url !== img.url; });
                    if (selectedImage && selectedImage.url === img.url) hideInsertPanel();
                  } else {
                    alert('Failed to delete image');
                  }
                })
                .catch(function() { alert('Failed to delete image'); });
            }
          });
        }
        grid.appendChild(card);
      });
    }

    // --- Upload form ---
    function resetUploadForm() {
      var dropzone = document.getElementById('gw-img-upload-dropzone');
      var preview = document.getElementById('gw-img-upload-preview');
      var fileInput = document.getElementById('gw-img-upload-input');
      if (dropzone) dropzone.classList.remove('gw-hidden');
      if (preview) preview.classList.add('gw-hidden');
      if (fileInput) fileInput.value = '';
    }

    function setupUploadForm() {
      var dropzone = document.getElementById('gw-img-upload-dropzone');
      var fileInput = document.getElementById('gw-img-upload-input');
      var preview = document.getElementById('gw-img-upload-preview');
      var thumb = document.getElementById('gw-img-upload-thumb');
      var altInput = document.getElementById('gw-img-upload-alt');
      var captionInput = document.getElementById('gw-img-upload-caption');
      var cancelBtn = document.getElementById('gw-img-upload-cancel');
      var submitBtn = document.getElementById('gw-img-upload-submit');
      if (!dropzone) return;

      var pendingFile = null;

      function showFilePreview(file) {
        pendingFile = file;
        var defaultAlt = file.name.replace(/\.[^.]+$/, '').replace(/[-_]/g, ' ');
        if (altInput) { altInput.value = defaultAlt; }
        if (captionInput) { captionInput.value = ''; }
        if (thumb) {
          var reader = new FileReader();
          reader.onload = function(e) { thumb.src = e.target.result; };
          reader.readAsDataURL(file);
        }
        dropzone.classList.add('gw-hidden');
        if (preview) preview.classList.remove('gw-hidden');
        if (altInput) { altInput.focus(); altInput.select(); }
      }

      dropzone.addEventListener('click', function() { if (fileInput) fileInput.click(); });
      if (fileInput) {
        fileInput.addEventListener('change', function() {
          if (fileInput.files && fileInput.files[0]) showFilePreview(fileInput.files[0]);
        });
      }

      // Drag & drop on the upload dropzone
      var dragCount = 0;
      dropzone.addEventListener('dragenter', function(e) { e.preventDefault(); dragCount++; dropzone.classList.add('gw-drag-over'); });
      dropzone.addEventListener('dragover', function(e) { e.preventDefault(); if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy'; });
      dropzone.addEventListener('dragleave', function(e) { e.preventDefault(); dragCount--; if (dragCount <= 0) { dragCount = 0; dropzone.classList.remove('gw-drag-over'); } });
      dropzone.addEventListener('drop', function(e) {
        e.preventDefault();
        dragCount = 0;
        dropzone.classList.remove('gw-drag-over');
        var files = e.dataTransfer && e.dataTransfer.files;
        if (files && files[0] && files[0].type.startsWith('image/')) showFilePreview(files[0]);
      });

      if (cancelBtn) {
        cancelBtn.addEventListener('click', function() {
          pendingFile = null;
          resetUploadForm();
        });
      }

      if (submitBtn) {
        submitBtn.addEventListener('click', function() {
          if (!pendingFile) return;
          var alt = (altInput ? altInput.value.trim() : '') || '';
          var caption = (captionInput ? captionInput.value.trim() : '') || '';
          var size = (document.getElementById('gw-img-upload-size') || {}).value || 'm';
          if (!alt) { alert('Alt text is required'); if (altInput) altInput.focus(); return; }

          submitBtn.disabled = true;
          submitBtn.textContent = 'Uploading...';

          var folder = cfg.cloudinaryFolder || 'blog/images';
          uploadToCloudinary(pendingFile, folder)
            .then(function(url) {
              return saveImageMetadata(url, alt, caption).then(function() { return url; });
            })
            .then(function(url) {
              insertAtCursor(editor, buildImageMarkup(url, alt, caption, size));
              // Add to allImages for immediate grid display
              allImages.unshift({ url: url, alt_text: alt, caption: caption, filename: pendingFile.name });
              pendingFile = null;
              resetUploadForm();
              showBrowseView();
              renderGrid();
            })
            .catch(function(err) {
              alert('Upload failed: ' + err.message);
              submitBtn.disabled = false;
              submitBtn.textContent = 'Upload & Insert';
            });
        });
      }
    }

    // --- Open modal ---
    function openModal() {
      modal.classList.remove('gw-hidden');
      showBrowseView();
      hideInsertPanel();

      // Show/hide tab bar based on postId
      if (tabBar) {
        if (cfg.postId) {
          tabBar.classList.remove('gw-hidden');
          // Default to Post Images tab when editing
          currentTab = 'post';
          var tabs = tabBar.querySelectorAll('.gw-img-tab');
          tabs.forEach(function(t) {
            t.classList.toggle('active', t.getAttribute('data-tab') === 'post');
          });
        } else {
          tabBar.classList.add('gw-hidden');
          currentTab = 'all';
        }
      }

      grid.innerHTML = '<div class="gw-img-loading">Loading images...</div>';

      fetch(listEndpoint)
        .then(function(res) { return res.json(); })
        .then(function(data) {
          allImages = data.images || [];
          renderGrid();
        })
        .catch(function() {
          grid.innerHTML = '<div class="gw-img-empty">Failed to load images</div>';
        });

      // Setup upload form (idempotent since we reset state)
      setupUploadForm();
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

    // SVG icons
    var ICON_PENCIL = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 3a2.828 2.828 0 114 4L7.5 20.5 2 22l1.5-5.5L17 3z"/></svg>';
    var ICON_TRASH = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>';

    function closeModal() {
      // Remove any confirm overlay when closing
      var overlay = modal.querySelector('.gw-draw-confirm-overlay');
      if (overlay) overlay.remove();
      modal.classList.add('gw-hidden');
    }
    if (closeBtn) closeBtn.addEventListener('click', closeModal);
    if (backdrop) backdrop.addEventListener('click', closeModal);

    // In-app confirmation dialog (replaces confirm())
    function showConfirm(message, onConfirm) {
      // Remove any existing overlay
      var old = modal.querySelector('.gw-draw-confirm-overlay');
      if (old) old.remove();

      var overlay = document.createElement('div');
      overlay.className = 'gw-draw-confirm-overlay';
      var box = document.createElement('div');
      box.className = 'gw-draw-confirm-box';
      var msg = document.createElement('div');
      msg.className = 'gw-draw-confirm-msg';
      msg.textContent = message;
      var actions = document.createElement('div');
      actions.className = 'gw-draw-confirm-actions';
      var cancelBtn = document.createElement('button');
      cancelBtn.type = 'button';
      cancelBtn.className = 'gw-draw-confirm-cancel';
      cancelBtn.textContent = 'Cancel';
      cancelBtn.addEventListener('click', function() { overlay.remove(); });
      var confirmBtn = document.createElement('button');
      confirmBtn.type = 'button';
      confirmBtn.className = 'gw-draw-confirm-delete';
      confirmBtn.textContent = 'Delete';
      confirmBtn.addEventListener('click', function() {
        overlay.remove();
        onConfirm();
      });
      actions.appendChild(cancelBtn);
      actions.appendChild(confirmBtn);
      box.appendChild(msg);
      box.appendChild(actions);
      overlay.appendChild(box);
      // Clicking overlay backdrop cancels
      overlay.addEventListener('click', function(e) {
        if (e.target === overlay) overlay.remove();
      });
      modal.querySelector('.gw-modal-content').appendChild(overlay);
    }

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

    function buildShortcode(id, size, zoom) {
      var sizeTag = (!size || size === 'm') ? '' : ':' + size;
      var zoomTag = (!zoom || zoom === 'fit') ? '' : ':z' + zoom;
      return '\n[draw:' + id + ':edit' + sizeTag + zoomTag + ']\n';
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

            // Size dropdown
            var sizeSelect = document.createElement('select');
            sizeSelect.className = 'gw-draw-select';
            sizeSelect.title = 'Size';
            [['s','S'],['m','M'],['l','L']].forEach(function(pair) {
              var opt = document.createElement('option');
              opt.value = pair[0];
              opt.textContent = pair[1];
              if (pair[0] === 'm') opt.selected = true;
              sizeSelect.appendChild(opt);
            });

            // Zoom dropdown
            var zoomSelect = document.createElement('select');
            zoomSelect.className = 'gw-draw-select';
            zoomSelect.title = 'Zoom';
            [['fit','fit'],['50%','50%'],['100%','100%'],['150%','150%'],['200%','200%']].forEach(function(pair) {
              var opt = document.createElement('option');
              opt.value = pair[0];
              opt.textContent = pair[1];
              if (pair[0] === 'fit') opt.selected = true;
              zoomSelect.appendChild(opt);
            });

            // Edit icon button
            var editBtn = document.createElement('button');
            editBtn.type = 'button';
            editBtn.className = 'gw-draw-icon-btn gw-draw-icon-edit';
            editBtn.title = 'Edit drawing';
            editBtn.innerHTML = ICON_PENCIL;
            editBtn.addEventListener('click', function(e) {
              e.stopPropagation();
              window.open(drawBase + '/' + drw.id + '/edit', '_blank');
            });

            // Delete icon button
            var delBtn = document.createElement('button');
            delBtn.type = 'button';
            delBtn.className = 'gw-draw-icon-btn gw-draw-icon-delete';
            delBtn.title = 'Delete drawing';
            delBtn.innerHTML = ICON_TRASH;
            delBtn.addEventListener('click', function(e) {
              e.stopPropagation();
              showConfirm('Delete "' + (drw.title || 'Untitled') + '"?', function() {
                fetch(drawBase + '/api/' + drw.id + '/delete', { method: 'POST' })
                  .then(function(res) {
                    if (res.ok) card.remove();
                    else alert('Failed to delete drawing');
                  })
                  .catch(function() { alert('Failed to delete drawing'); });
              });
            });

            actions.appendChild(sizeSelect);
            actions.appendChild(zoomSelect);
            actions.appendChild(editBtn);
            actions.appendChild(delBtn);

            card.appendChild(titleRow);
            card.appendChild(meta);
            card.appendChild(actions);

            // Click card to insert (excluding title, selects, buttons)
            card.addEventListener('click', function(e) {
              var tag = e.target.tagName;
              if (tag === 'SELECT' || tag === 'OPTION' || tag === 'BUTTON' || tag === 'SVG' || tag === 'PATH' || tag === 'POLYLINE') return;
              if (e.target.isContentEditable) return;
              if (e.target.closest('.gw-draw-icon-btn') || e.target.closest('select')) return;
              var shortcode = buildShortcode(drw.id, sizeSelect.value, zoomSelect.value);
              insertAtCursor(editor, shortcode);
              closeModal();
            });

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
              showConfirm('Delete ' + unusedIds.length + ' unused drawing(s)? This cannot be undone.', function() {
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
    initDrawEmbeds: initDrawEmbeds,
    buildImageMarkup: buildImageMarkup,
    uploadToCloudinary: uploadToCloudinary,
    saveImageMetadata: saveImageMetadata,
    isCloudinaryConfigured: isCloudinaryConfigured
  };
})();
