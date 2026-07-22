// @ts-check

const OP_KEYS = ['create_role', 'alter_user'];
const FN_KEYS = {
  create_role: 'createRole',
  remove_role: 'removeRole',
  grant_parents: 'grantParents',
  revoke_parents: 'revokeParents',
  change_password: 'changePassword',
};

const ROLE_NAME_RE = /^[A-Za-z_][A-Za-z0-9_]*$/;

/** @type {import('../internal/model/model').Config | null} */
let state = null;
let currentOp = 'create_role';
let lastResults = [];

// --- Alter role tab state ---
/** @type {Array<{loginName:string, fullName:string, clusters:Array<any>}>} */
let alterGroups = [];
/** @type {string|null} */
let alterSelected = null;
/** @type {Array<any>} */
let alterDetails = [];
// Consolidated edit model for the single user form:
/** @type {Map<string, Set<string>>} role -> set of clusterIds to grant it on */
let alterAdd = new Map();
/** @type {Map<string, Set<string>>} role -> set of clusterIds to revoke it on */
let alterRevoke = new Map();
let alterDoPassword = false;
/** @type {{categoryIds:string[], clusterIds:string[]}} target selection captured at search time */
let alterTargets = { categoryIds: [], clusterIds: [] };
/** @type {Array<{clusterId:string, alias:string, category:string}>} selected clusters (compare universe) */
let alterScopeClusters = [];
/** @type {Map<string, Set<string>>} attribute key -> clusterIds to enable */
let alterAttrAdd = new Map();
/** @type {Map<string, Set<string>>} attribute key -> clusterIds to disable */
let alterAttrRemove = new Map();
/** @type {Map<string, Set<string>>} "name=value" -> clusterIds to SET */
let alterConfigSet = new Map();
/** @type {Map<string, Set<string>>} setting name -> clusterIds to RESET */
let alterConfigReset = new Map();
/** Inline comment editor state (shared by Create/Alter). Rendered FROM and written TO by
 *  the editor; renderAlterDetail never touches it, so pending edits survive re-renders. */
let commentEditor = {
  mode: 'fields', // 'fields' | 'raw'
  baseObj: {}, // parsed object of the loaded consensus comment (preserves unknown/non-string keys)
  isObject: false, // did the loaded/raw comment parse as a JSON object?
  raw: '', // raw textarea text (authoritative in raw mode)
  shownKeys: [], // ordered keys rendered as value-inputs (configured first, then other keys)
  labels: {}, // key -> label
  values: {}, // key -> edited string value (JSON text for read-only non-string keys)
  readonly: new Set(), // keys whose value isn't a string: shown but only Raw-editable
  varies: false, // comment differs across clusters (inline editor disabled; use Comments dialog)
};
/** @type {Map<string, string>} clusterId -> desired comment, staged from the Comments dialog
 *  (varies case). Published together with the other edits on "Save changes". */
let commentOverrides = new Map();

// The role form is shared by Create and Alter. Create = editing a not-yet-existing role
// across the selected clusters with an empty baseline; Alter = editing an existing role
// over the clusters it lives on. `currentOp` selects the mode.
function isCreateMode() {
  return currentOp === 'create_role';
}

/** Clear all pending edit maps + password state (both modes). */
function resetEditMaps() {
  alterAdd = new Map();
  alterRevoke = new Map();
  alterAttrAdd = new Map();
  alterAttrRemove = new Map();
  alterConfigSet = new Map();
  alterConfigReset = new Map();
  commentOverrides = new Map();
  alterDoPassword = false;
  alterPassword = '';
}

/** Create mode: baseline = the selected clusters as empty (not-yet-existing) rows. */
function synthCreateBaseline() {
  alterScopeClusters = [];
  alterSelected = document.getElementById('role-login')?.value.trim() || null;
  alterDetails = resolveSelectedClusters().map((c) => ({
    clusterId: c.id,
    alias: c.alias,
    category: c.category,
    exists: false,
    comment: '',
    fullName: '',
    parents: [],
    attributes: {},
    settings: {},
  }));
}

/** Create mode: drop pending grants/enables/sets that point at no-longer-selected clusters. */
function reconcilePendingWithUniverse() {
  const universe = new Set(alterDetails.map((d) => d.clusterId));
  const prune = (map) => {
    for (const [key, ids] of map) {
      for (const id of [...ids]) if (!universe.has(id)) ids.delete(id);
      if (!ids.size) map.delete(key);
    }
  };
  prune(alterAdd);
  prune(alterRevoke);
  prune(alterAttrAdd);
  prune(alterAttrRemove);
  prune(alterConfigSet);
  prune(alterConfigReset);
}

// Editable role attributes (pg_roles flag -> ALTER ROLE enable/disable keywords).
const ROLE_ATTRIBUTES = [
  { key: 'super', label: 'Superuser', on: 'SUPERUSER', off: 'NOSUPERUSER' },
  { key: 'createrole', label: 'Create role', on: 'CREATEROLE', off: 'NOCREATEROLE' },
  { key: 'createdb', label: 'Create DB', on: 'CREATEDB', off: 'NOCREATEDB' },
  { key: 'inherit', label: 'Inherit', on: 'INHERIT', off: 'NOINHERIT' },
  { key: 'login', label: 'Login', on: 'LOGIN', off: 'NOLOGIN' },
  { key: 'replication', label: 'Replication', on: 'REPLICATION', off: 'NOREPLICATION' },
  { key: 'bypassrls', label: 'Bypass RLS', on: 'BYPASSRLS', off: 'NOBYPASSRLS' },
];
/** @type {MediaQueryList | null} */
let systemThemeMedia = null;

function backend() {
  return window.go?.main?.App;
}

/** @returns {Promise<boolean>} */
function askConfirm(title, message) {
  return new Promise((resolve) => {
    const dlg = document.getElementById('confirm-dialog');
    const okBtn = document.getElementById('confirm-dialog-ok');
    const cancelBtn = document.getElementById('confirm-dialog-cancel');
    document.getElementById('confirm-dialog-title').textContent = title;
    document.getElementById('confirm-dialog-message').textContent = message;

    const cleanup = () => {
      okBtn.removeEventListener('click', onOk);
      cancelBtn.removeEventListener('click', onCancel);
      dlg.removeEventListener('cancel', onCancel);
    };

    const onOk = () => {
      cleanup();
      dlg.close();
      resolve(true);
    };
    const onCancel = () => {
      cleanup();
      dlg.close();
      resolve(false);
    };

    okBtn.addEventListener('click', onOk);
    cancelBtn.addEventListener('click', onCancel);
    dlg.addEventListener('cancel', onCancel);
    dlg.showModal();
  });
}


/** Disable auto-capitalization on technical fields; only fullName uses words. */
function configureInputCapitalization(root = document) {
  root.querySelectorAll('input[type="text"], input[type="email"], input[type="password"]').forEach((el) => {
    if (el.name === 'fullName') {
      el.setAttribute('autocapitalize', 'words');
      return;
    }
    el.setAttribute('autocapitalize', 'none');
    el.setAttribute('autocomplete', 'off');
    if (el.type !== 'password') {
      el.setAttribute('spellcheck', 'false');
    }
  });
  root.querySelectorAll('textarea').forEach((el) => {
    el.setAttribute('autocapitalize', 'none');
    el.setAttribute('spellcheck', 'false');
  });
}

function showToast(msg, type = '') {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.className = 'toast' + (type ? ' ' + type : '');
  el.classList.remove('hidden');
  setTimeout(() => el.classList.add('hidden'), 4500);
}

/** Currently-selected Appearance preference (from the segmented button group). */
function currentThemePref() {
  return document.querySelector('#ui-theme .seg-btn.active')?.dataset.pref || state?.ui?.theme || 'system';
}

/** Reflect a preference in the segmented Appearance buttons. */
function setThemeButtons(pref) {
  document.querySelectorAll('#ui-theme .seg-btn').forEach((b) => {
    const on = b.dataset.pref === pref;
    b.classList.toggle('active', on);
    b.setAttribute('aria-pressed', String(on));
  });
}

/** Currently-selected preferred comment view (from the segmented button group). */
function currentCommentViewPref() {
  return document.querySelector('#comment-view-pref .seg-btn.active')?.dataset.pref || state?.ui?.commentDefaultView || 'fields';
}

/** Reflect a preference in the segmented Preferred-comment-view buttons. */
function setCommentViewButtons(pref) {
  document.querySelectorAll('#comment-view-pref .seg-btn').forEach((b) => {
    const on = b.dataset.pref === pref;
    b.classList.toggle('active', on);
    b.setAttribute('aria-pressed', String(on));
  });
}

function applyTheme(themePref) {
  const pref = themePref || 'system';
  let resolved = pref;
  if (pref === 'system') {
    if (!systemThemeMedia) {
      systemThemeMedia = window.matchMedia('(prefers-color-scheme: dark)');
      systemThemeMedia.addEventListener('change', () => {
        if (currentThemePref() === 'system') applyTheme('system');
      });
    }
    resolved = systemThemeMedia.matches ? 'dark' : 'light';
  }
  document.documentElement.setAttribute('data-theme', resolved);

  const rt = window.runtime;
  if (!rt) return;
  try {
    if (pref === 'system' && rt.WindowSetSystemDefaultTheme) {
      rt.WindowSetSystemDefaultTheme();
    } else if (resolved === 'light' && rt.WindowSetLightTheme) {
      rt.WindowSetLightTheme();
    } else if (rt.WindowSetDarkTheme) {
      rt.WindowSetDarkTheme();
    }
  } catch {
    /* native theme optional */
  }
}

async function loadConfig() {
  const app = backend();
  if (!app) {
    showToast('Wails backend not available (open via wails dev or built app)', 'error');
    return;
  }
  try {
    state = await app.GetConfig();
    document.getElementById('config-path').textContent = await app.GetConfigPath();
    setThemeButtons(state?.ui?.theme || 'system');
    applyTheme(state?.ui?.theme || 'system');
    setCommentViewButtons(state?.ui?.commentDefaultView || 'fields');
    renderAll();
  } catch (e) {
    showToast(String(e), 'error');
  }
}

function categoryLabel(id) {
  const c = state?.categories?.find((x) => x.id === id);
  return c?.label || id;
}

const DEFAULT_CAT_COLOR = '#9aa3b5';
function categoryColor(id) {
  const c = state?.categories?.find((x) => x.id === id);
  return (c && c.color) || DEFAULT_CAT_COLOR;
}

/** Whether a group requires the production-style confirm popup. */
function categoryConfirm(id) {
  return !!state?.categories?.find((x) => x.id === id)?.confirm;
}

/** #rrggbb → rgba(r,g,b,a). Falls back to the default colour on bad input. */
function hexToRgba(hex, a) {
  const m = /^#?([0-9a-f]{6})$/i.exec((hex || '').trim());
  const h = m ? m[1] : DEFAULT_CAT_COLOR.slice(1);
  const n = parseInt(h, 16);
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${a})`;
}

/** Generate per-group colour rules (badges, scope labels, checkbox highlight). */
function renderCategoryColors() {
  let el = document.getElementById('cat-colors');
  if (!el) {
    el = document.createElement('style');
    el.id = 'cat-colors';
    document.head.appendChild(el);
  }
  el.textContent = (state?.categories || [])
    .map((c) => {
      const hex = c.color || DEFAULT_CAT_COLOR;
      const id = (window.CSS && CSS.escape) ? CSS.escape(c.id) : c.id;
      return [
        `.badge[data-cat="${id}"]{color:${hex};background:transparent;border-color:${hex}}`,
        `.chip-scope.scope-kind-group[data-cat="${id}"]{color:${hex};background:transparent;border-color:${hex}}`,
        `.chip-scope.scope-kind-cluster[data-cat="${id}"]{color:${hex};background:${hexToRgba(hex, 0.16)}}`,
        `.checkbox-group label[data-category="${id}"]:has(input:checked){border-color:${hex};background:${hexToRgba(hex, 0.18)}}`,
      ].join('');
    })
    .join('\n');
}

function renderGroupsTable() {
  const tbody = document.querySelector('#groups-table tbody');
  if (!tbody) return;
  tbody.innerHTML = '';
  if (!state?.categories?.length) {
    tbody.innerHTML = '<tr><td colspan="3" class="hint">No groups defined.</td></tr>';
    return;
  }
  // Pre-check usage so a group in use can't be deleted (Delete disabled, no failed attempt).
  const usage = {};
  for (const cl of state?.clusters || []) usage[cl.category] = (usage[cl.category] || 0) + 1;

  for (const c of state.categories) {
    const used = usage[c.id] || 0;
    const del = used
      ? `<button class="small danger" data-action="delete" data-id="${escapeAttr(c.id)}" disabled title="In use by ${used} cluster${used === 1 ? '' : 's'}">Delete</button>`
      : `<button class="small danger" data-action="delete" data-id="${escapeAttr(c.id)}">Delete</button>`;
    const tr = document.createElement('tr');
    tr.innerHTML = `
      <td><span class="badge" data-cat="${escapeAttr(c.id)}">${escapeHtml(c.label)}</span></td>
      <td>${c.confirm ? 'Yes' : '—'}</td>
      <td>
        <button class="small" data-action="edit" data-id="${escapeAttr(c.id)}">Edit</button>
        ${del}
      </td>`;
    tbody.appendChild(tr);
  }
  tbody.querySelectorAll('button').forEach((btn) => btn.addEventListener('click', onGroupAction));
}

function openGroupDialog(cat) {
  const dlg = document.getElementById('group-dialog');
  const form = document.getElementById('group-form');
  document.getElementById('group-dialog-title').textContent = cat ? 'Edit group' : 'Add group';
  form.id.value = cat?.id || '';
  form.label.value = cat?.label || '';
  form.color.value = cat?.color || DEFAULT_CAT_COLOR;
  form.confirm.checked = !!cat?.confirm;
  dlg.showModal();
}

async function onGroupAction(ev) {
  const btn = ev.currentTarget;
  const id = btn.dataset.id;
  const app = backend();
  const cat = state?.categories?.find((c) => c.id === id);
  if (btn.dataset.action === 'edit') {
    openGroupDialog(cat);
    return;
  }
  if (btn.dataset.action === 'delete') {
    const ok = await askConfirm('Delete group', `Delete cluster group "${cat?.label || id}"?`);
    if (!ok) return;
    try {
      await app.DeleteCategory(id);
      await loadConfig();
      showToast('Group deleted', 'success');
    } catch (e) {
      showToast(String(e), 'error');
    }
  }
}

function renderClustersTable() {
  const tbody = document.querySelector('#clusters-table tbody');
  tbody.innerHTML = '';
  if (!state?.clusters?.length) {
    tbody.innerHTML = '<tr><td colspan="7" class="hint">No clusters configured.</td></tr>';
    return;
  }
  for (const c of state.clusters) {
    const tr = document.createElement('tr');
    tr.innerHTML = `
      <td>${escapeHtml(c.alias)}</td>
      <td>${escapeHtml(c.host)}</td>
      <td>${c.port}</td>
      <td>${escapeHtml(c.database)}</td>
      <td><span class="badge" data-cat="${escapeAttr(c.category)}">${escapeHtml(categoryLabel(c.category))}</span></td>
      <td class="cluster-status" data-status-for="${escapeAttr(c.id)}"></td>
      <td>
        <div class="row-actions">
          <button class="scope-act" data-action="edit" data-id="${c.id}" title="Edit cluster" aria-label="Edit cluster">✎</button>
          <button class="scope-act" data-action="delete" data-id="${c.id}" title="Delete cluster" aria-label="Delete cluster">×</button>
        </div>
      </td>`;
    tbody.appendChild(tr);
  }
  tbody.querySelectorAll('button').forEach((btn) => {
    btn.addEventListener('click', onClusterAction);
  });
}

function renderCategoryCheckboxes() {
  const box = document.getElementById('category-checkboxes');
  box.innerHTML = '';
  for (const cat of state?.categories || []) {
    const label = document.createElement('label');
    label.dataset.category = cat.id;
    label.innerHTML = `<input type="checkbox" name="category" value="${escapeAttr(cat.id)}" checked />
      <span class="badge" data-cat="${escapeAttr(cat.id)}">${escapeHtml(cat.label)}</span>`;
    box.appendChild(label);
    label.querySelector('input')?.addEventListener('change', updateTargetPreview);
  }
}

function renderClusterCheckboxes() {
  const box = document.getElementById('cluster-checkboxes');
  box.innerHTML = '';
  for (const c of state?.clusters || []) {
    const label = document.createElement('label');
    if (c.category) label.dataset.category = c.category;
    label.innerHTML = `<input type="checkbox" name="cluster" value="${escapeAttr(c.id)}" />
      <span class="target-cluster-text">${escapeHtml(c.alias)} <span class="target-cluster-host">(${escapeHtml(c.host)})</span></span>
      <span class="badge" data-cat="${escapeAttr(c.category)}">${escapeHtml(categoryLabel(c.category))}</span>`;
    box.appendChild(label);
    label.querySelector('input')?.addEventListener('change', updateTargetPreview);
  }
}

// DB command templates: op key → [title, state property, allowed placeholders].
const DB_FUNCTIONS = [
  ['create_role', 'Create role', 'createRole', ['loginname', 'fullname', 'email', 'parent_role']],
  ['remove_role', 'Remove role', 'removeRole', ['loginname', 'rolename']],
  ['grant_parents', 'Grant parents', 'grantParents', ['loginname', 'parent_roles']],
  ['revoke_parents', 'Revoke parents', 'revokeParents', ['loginname', 'parent_roles']],
  ['change_password', 'Change password', 'changePassword', ['loginname', 'new_password']],
  ['set_comment', 'Set comment', 'setComment', ['loginname', 'comment']],
  ['set_attribute', 'Set attribute', 'setAttribute', ['loginname', 'attribute']],
];
const EXECUTION_LABELS = { function: 'Function call', statement: 'SQL statement', block: 'PL/pgSQL block' };

/** In-memory edits for the command templates, staged until "Save settings". */
let dbFnDraft = {};
/** @type {string|null} op currently open in the template dialog */
let fnDialogKey = null;
/** In-memory edits for comment fields, staged until "Save settings". */
/** @type {Array<{key:string, label:string}>} */
let commentFieldsDraft = [];
/** In-memory edits for preconfigured parent groups, staged until "Save settings". */
/** @type {Array<string>} */
let parentRolesDraft = [];

// --- Shared draggable list editor (comment fields, parent groups) ---

/** One reorderable row: a drag handle, caller-supplied cells, and a remove button. */
function listRowHtml(idx, innerHtml) {
  return `
    <div class="le-row" data-le-idx="${idx}">
      <span class="le-grip" draggable="true" data-le-idx="${idx}" title="Drag to reorder" aria-label="Drag to reorder">⠿</span>
      ${innerHtml}
      <button type="button" class="le-del" data-le-idx="${idx}" title="Remove" aria-label="Remove">×</button>
    </div>`;
}

/** Wire drag-to-reorder + remove on a list-editor container. getDraft returns the backing
 *  array; rerender repaints it. Row indices come from data-le-idx. */
function wireListEditor(containerId, getDraft, rerender) {
  const root = document.getElementById(containerId);
  if (!root) return;
  let dragIdx = null;
  root.addEventListener('dragstart', (ev) => {
    const grip = ev.target.closest('.le-grip');
    if (!grip) return;
    dragIdx = Number(grip.dataset.leIdx);
    ev.dataTransfer.effectAllowed = 'move';
    ev.dataTransfer.setData('text/plain', String(dragIdx)); // required for DnD to arm in some webviews
  });
  root.addEventListener('dragover', (ev) => {
    if (dragIdx == null) return;
    ev.preventDefault();
    ev.dataTransfer.dropEffect = 'move';
  });
  root.addEventListener('drop', (ev) => {
    if (dragIdx == null) return;
    ev.preventDefault();
    const draft = getDraft();
    const row = ev.target.closest('.le-row');
    const [moved] = draft.splice(dragIdx, 1);
    if (!row) {
      draft.push(moved); // dropped past the last row → append
    } else {
      // Insert at the drop target's ORIGINAL index within the post-removal array: dragging
      // down lands the row after the target (so first→last works), dragging up before it.
      draft.splice(Number(row.dataset.leIdx), 0, moved);
    }
    dragIdx = null;
    rerender();
  });
  root.addEventListener('dragend', () => {
    dragIdx = null;
  });
  root.addEventListener('click', (ev) => {
    const del = ev.target.closest('.le-del');
    if (!del) return;
    getDraft().splice(Number(del.dataset.leIdx), 1);
    rerender();
  });
}

/** Rebuild the comment-fields draft from saved config, then paint it. */
function renderCommentFieldsEditor() {
  const src = state?.commentFields?.length ? state.commentFields : DEFAULT_COMMENT_FIELDS;
  commentFieldsDraft = src.map((f) => ({ key: f.key || '', label: f.label || '' }));
  renderCommentFieldsDraft();
}

/** Paint the comment-fields editor from the current draft (after add/delete/reorder). */
function renderCommentFieldsDraft() {
  const root = document.getElementById('comment-fields-editor');
  if (!root) return;
  root.innerHTML = commentFieldsDraft
    .map((f, i) =>
      listRowHtml(
        i,
        `<input class="cf-key" data-idx="${i}" value="${escapeAttr(f.key)}" placeholder="key (e.g. full_name)" autocapitalize="none" autocomplete="off" spellcheck="false" />
      <input class="cf-label" data-idx="${i}" value="${escapeAttr(f.label)}" placeholder="label (e.g. Full name)" autocomplete="off" />`
      )
    )
    .join('');
}

/** Rebuild the parent-groups draft from saved config, then paint it. */
function renderParentRolesEditor() {
  parentRolesDraft = (state?.parentRoles || []).slice();
  renderParentRolesDraft();
}

/** Paint the parent-groups editor from the current draft (after add/delete/reorder). */
function renderParentRolesDraft() {
  const root = document.getElementById('parent-roles-editor');
  if (!root) return;
  root.innerHTML = parentRolesDraft
    .map((r, i) =>
      listRowHtml(
        i,
        `<input class="pr-value" data-idx="${i}" value="${escapeAttr(r)}" placeholder="e.g. gr_devs_ro" autocapitalize="none" autocomplete="off" spellcheck="false" />`
      )
    )
    .join('');
}

/** Compact list of command names; click a row to edit it in a popup. Rebuilds the draft
 *  from the saved config each render. */
function renderDBFunctionsEditor() {
  const root = document.getElementById('db-functions-editor');
  if (!root) return;
  const fns = state?.dbFunctions;
  dbFnDraft = {};
  root.innerHTML = '';
  for (const [key, title, prop] of DB_FUNCTIONS) {
    const fn = fns?.[prop];
    dbFnDraft[key] = {
      call: fn?.call || fn?.Call || '',
      execution: fn?.execution || fn?.Execution || 'function',
    };
    const row = document.createElement('button');
    row.type = 'button';
    row.className = 'fn-row';
    row.dataset.fnKey = key;
    row.innerHTML = `<span class="fn-row-name">${escapeHtml(title)}</span>
      <span class="fn-row-exec">${escapeHtml(EXECUTION_LABELS[dbFnDraft[key].execution] || dbFnDraft[key].execution)}</span>`;
    root.appendChild(row);
  }
  document.getElementById('batch-concurrency').value = String(state?.batch?.maxConcurrency || 5);
}

/** Open the template popup for one operation. */
function openFnDialog(key) {
  const meta = DB_FUNCTIONS.find((e) => e[0] === key);
  if (!meta) return;
  fnDialogKey = key;
  const draft = dbFnDraft[key] || { call: '', execution: 'function' };
  document.getElementById('fn-dialog-title').textContent = meta[1];
  document.getElementById('fn-execution').value = draft.execution;
  document.getElementById('fn-call').value = draft.call;
  document.getElementById('fn-placeholder-list').innerHTML = meta[3]
    .map((p) => `<button type="button" class="ph-chip" data-ph="${escapeAttr(p)}">$\{${escapeHtml(p)}}</button>`)
    .join('');
  document.getElementById('fn-dialog').showModal();
  document.getElementById('fn-call').focus();
}

/** Refresh one command row's execution badge from the draft (without rebuilding the list,
 *  which would reset the draft from saved config). */
function renderDBFunctionsRow(key) {
  const row = document.querySelector(`#db-functions-editor .fn-row[data-fn-key="${key}"]`);
  const d = dbFnDraft[key];
  if (!row || !d) return;
  const execEl = row.querySelector('.fn-row-exec');
  if (execEl) execEl.textContent = EXECUTION_LABELS[d.execution] || d.execution;
}

function renderResults(rows) {
  lastResults = rows || [];
  const section = document.getElementById('status-section');
  const tbody = document.querySelector('#results-table tbody');
  tbody.innerHTML = '';
  // Hide the Status section entirely until there is something to show.
  section?.classList.toggle('hidden', !rows?.length);
  if (!rows?.length) return;
  for (const r of rows) {
    const tr = document.createElement('tr');
    tr.innerHTML = `
      <td>${escapeHtml(r.alias)}</td>
      <td>${escapeHtml(r.host)}</td>
      <td><span class="badge" data-cat="${escapeAttr(r.category)}">${escapeHtml(categoryLabel(r.category))}</span></td>
      <td class="${r.status === 'ok' ? 'status-ok' : 'status-error'}">${escapeHtml(r.status)}</td>
      <td>${r.durationMs} ms</td>
      <td>${escapeHtml(r.message || '')}</td>`;
    tbody.appendChild(tr);
  }
}

function renderAll() {
  renderCategoryColors();
  renderClustersTable();
  renderCategoryCheckboxes();
  renderClusterCheckboxes();
  renderGroupsTable();
  renderDBFunctionsEditor();
  renderCommentFieldsEditor();
  renderParentRolesEditor();
  updateTargetPreview();
}

function getSelectedCategories() {
  return [...document.querySelectorAll('#category-checkboxes input:checked')].map((el) => el.value);
}

function getSelectedClusterIDs() {
  return [...document.querySelectorAll('#cluster-checkboxes input:checked')].map((el) => el.value);
}

function getAuth() {
  // Connection controls were removed from the UI; run-time auth resolves from cluster
  // connect_user / PGUSER / PGPASSWORD / ~/.pgpass on the backend.
  return { user: '', password: '' };
}

async function updateTargetPreview() {
  const app = backend();
  const preview = document.getElementById('target-preview');
  // Create mode: the universe is the selected clusters, so rebuild the synthetic baseline
  // (and drop pending edits on deselected clusters) whenever the selection changes.
  if (isCreateMode()) {
    synthCreateBaseline();
    reconcilePendingWithUniverse();
    renderAlterDetail();
  }
  if (!app || !state) {
    preview.textContent = '';
    return;
  }
  try {
    const targets = await app.PreviewTargets({
      operation: currentOp,
      categoryIds: getSelectedCategories(),
      clusterIds: getSelectedClusterIDs(),
      auth: getAuth(),
      confirmProduction: true,
    });
    preview.textContent = `${targets.length} cluster(s) will be targeted.`;
  } catch (e) {
    preview.textContent = String(e);
  }
}

function fillCategorySelect(select) {
  select.innerHTML = '';
  for (const cat of state?.categories || []) {
    const opt = document.createElement('option');
    opt.value = cat.id;
    opt.textContent = cat.label;
    select.appendChild(opt);
  }
}

function openClusterDialog(cluster) {
  const dlg = document.getElementById('cluster-dialog');
  const form = document.getElementById('cluster-form');
  document.getElementById('cluster-dialog-title').textContent = cluster ? 'Edit cluster' : 'Add cluster';
  fillCategorySelect(form.category);
  form.id.value = cluster?.id || '';
  form.alias.value = cluster?.alias || '';
  form.host.value = cluster?.host || '';
  form.port.value = cluster?.port || 5432;
  form.database.value = cluster?.database || '';
  form.category.value = cluster?.category || 'uat';
  form.sslMode.value = cluster?.sslmode || 'prefer';
  form.connectUser.value = cluster?.connectUser || '';
  // Reset any leftover test result.
  const errEl = document.getElementById('cluster-test-error');
  errEl.classList.add('hidden');
  errEl.textContent = '';
  const testBtn = document.getElementById('btn-test-cluster');
  clusterTestTimers.forEach(clearTimeout);
  clusterTestTimers = [];
  testBtn.classList.remove('test-ok', 'test-fade');
  testBtn.style.minWidth = '';
  if (testBtn.dataset.label) testBtn.textContent = testBtn.dataset.label;
  dlg.showModal();
}

function clusterInputFromForm(form) {
  return {
    alias: form.alias.value.trim(),
    host: form.host.value.trim(),
    port: parseInt(form.port.value, 10) || 5432,
    database: form.database.value.trim(),
    category: form.category.value,
    sslMode: form.sslMode.value,
    connectUser: form.connectUser.value.trim(),
  };
}

async function onClusterAction(ev) {
  const btn = ev.currentTarget;
  const id = btn.dataset.id;
  const action = btn.dataset.action;
  const cluster = state?.clusters?.find((c) => c.id === id);
  const app = backend();

  if (action === 'edit') {
    openClusterDialog(cluster);
    return;
  }
  if (action === 'delete') {
    const ok = await askConfirm('Delete cluster', `Delete cluster "${cluster?.alias}"?`);
    if (!ok) return;
    try {
      await app.DeleteCluster(id);
      await loadConfig();
      showToast('Cluster deleted', 'success');
    } catch (e) {
      showToast(String(e), 'error');
    }
    return;
  }
}

/** Preconfigured parent groups defined in Settings. */
function preconfiguredParentRoles() {
  return state?.parentRoles || [];
}


/** Create-role privilege universe = the clusters covered by the sidebar target selection. */
/** Read the shared login input. Full name / email now live in the comment editor. */
function roleIdentityInputs() {
  return {
    loginName: document.getElementById('role-login')?.value.trim() || '',
  };
}

/** Build one create_role per selected cluster: base role + template base groups only
 *  (empty parent_role). Parents/attributes/settings/password/comment layer on via
 *  buildAlterRequests. full_name/e_mail feed the create_role template placeholders as a
 *  convenience; the follow-up set_comment (from the editor) is authoritative. */
function buildCreateRoleRequests(base) {
  return resolveSelectedClusters().map((c) => ({
    op: 'create_role',
    clusterId: c.id,
    params: {
      createRole: {
        loginName: base.loginName,
        fullName: commentEditor.values['full_name'] || '',
        email: commentEditor.values['e_mail'] || '',
        parentRole: '',
      },
    },
  }));
}

async function runOperation() {
  const app = backend();
  if (!app) {
    showToast('Wails backend not available', 'error');
    return;
  }
  if (!isCreateMode()) return;

  const base = roleIdentityInputs();
  if (!base.loginName || !ROLE_NAME_RE.test(base.loginName)) {
    showToast('Enter a valid login name (letters, digits, underscore)', 'error');
    return;
  }
  const clusters = resolveSelectedClusters();
  if (!clusters.length) {
    showToast('Select at least one category or cluster', 'error');
    return;
  }
  alterSelected = base.loginName; // buildAlterRequests keys its ops on this login

  // Pre-flight: warn if the role already exists on any selected cluster (create would error).
  try {
    const found = await app.LoadRoleDetails({
      loginName: base.loginName,
      categoryIds: getSelectedCategories(),
      clusterIds: getSelectedClusterIDs(),
      auth: getAuth(),
    });
    const present = (found || []).filter((d) => d.exists).map((d) => d.alias);
    if (present.length) {
      const ok = await askConfirm(
        'Role exists',
        `"${base.loginName}" already exists on ${present.join(', ')}. create_role will error there. Continue anyway?`
      );
      if (!ok) return;
    }
  } catch {
    /* pre-flight is best-effort; per-cluster errors will still surface on run */
  }

  // create_role first (each cluster), then the grant/attr/setting/password/comment diff.
  const requests = buildCreateRoleRequests(base).concat(buildAlterRequests());
  warnIfPlainTextComment(requests);
  const ok = await executeAlterRequests(requests, 'Role created');
  if (!ok) return;
  resetEditMaps();
  loadCommentEditor(); // clears the identity inputs (create mode)
  synthCreateBaseline();
  renderAlterDetail();
}

/** Update the inline Status cell for a cluster row on the Clusters page. */
function setClusterStatus(id, status, message) {
  const cell = [...document.querySelectorAll('#clusters-table td.cluster-status')]
    .find((td) => td.dataset.statusFor === id);
  if (!cell) return;
  const cls = status === 'ok' ? 'status-ok' : status === 'error' ? 'status-error' : 'status-pending';
  cell.className = `cluster-status ${cls}`;
  cell.textContent = message;
  cell.title = message;
}

/** Test every configured cluster and show the result inline in the table. */
async function testAllClusters() {
  const app = backend();
  if (!app) {
    showToast('Wails backend not available', 'error');
    return;
  }
  const clusters = state?.clusters || [];
  if (!clusters.length) {
    showToast('No clusters configured', 'error');
    return;
  }
  const btn = document.getElementById('btn-test-clusters');
  if (btn) btn.disabled = true;
  const auth = getAuth();
  clusters.forEach((c) => setClusterStatus(c.id, 'pending', 'testing…'));
  let ok = 0;
  let failed = 0;
  for (const c of clusters) {
    try {
      await app.TestConnection({ clusterId: c.id, auth });
      setClusterStatus(c.id, 'ok', 'connected');
      ok += 1;
    } catch (e) {
      setClusterStatus(c.id, 'error', String(e));
      failed += 1;
    }
  }
  if (btn) btn.disabled = false;
  showToast(
    `Tested ${clusters.length} cluster${clusters.length === 1 ? '' : 's'}: ${ok} OK, ${failed} failed`,
    failed ? 'error' : 'success',
  );
}

function readDBFunctionsFromEditor() {
  const out = {};
  for (const [key, , prop] of DB_FUNCTIONS) {
    const d = dbFnDraft[key] || { call: '', execution: 'function' };
    out[prop] = { call: (d.call || '').trim(), execution: d.execution || 'function' };
  }
  return out;
}

async function saveSettings() {
  const app = backend();
  try {
    await app.SaveDBFunctions(readDBFunctionsFromEditor());
    await app.SaveBatchSettings({
      maxConcurrency: parseInt(document.getElementById('batch-concurrency').value, 10) || 5,
    });
    await app.SaveUISettings({
      theme: currentThemePref(),
      commentDefaultView: currentCommentViewPref(),
    });
    await app.SaveParentRoles(parentRolesDraft.map((r) => r.trim()).filter(Boolean));
    await app.SaveCommentFields(
      commentFieldsDraft
        .map((f) => ({ key: f.key.trim(), label: f.label.trim() }))
        .filter((f) => f.key)
    );
    await loadConfig();
    showToast('Settings saved', 'success');
  } catch (e) {
    showToast(String(e), 'error');
  }
}

function escapeHtml(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function escapeAttr(s) {
  return escapeHtml(s).replace(/'/g, '&#39;');
}

// ------------------------------------------------------------------
// "?" help badges — hide a feature description behind a hover/focus popover.
// ------------------------------------------------------------------

/** Markup for a "?" badge whose hover popover shows `text`. */
function hintBadge(text) {
  return `<button type="button" class="q-hint" tabindex="0" aria-label="${escapeAttr(text)}" data-hint="${escapeAttr(text)}">?</button>`;
}

/** @type {HTMLDivElement | null} */
let qHintPop = null;

function positionQHint(btn) {
  const text = btn.dataset.hint || '';
  if (!text) return;
  if (!qHintPop) {
    qHintPop = document.createElement('div');
    qHintPop.className = 'q-hint-pop';
    qHintPop.setAttribute('role', 'tooltip');
  }
  // A modal <dialog> renders in the browser top layer, above any z-index; when the
  // badge lives in one, host the popover inside it so it isn't drawn under the dialog.
  const host = btn.closest('dialog[open]') || document.body;
  if (qHintPop.parentElement !== host) host.appendChild(qHintPop);
  qHintPop.textContent = text;
  qHintPop.hidden = false;
  const m = 8; // viewport margin
  const r = btn.getBoundingClientRect();
  const pw = qHintPop.offsetWidth;
  const ph = qHintPop.offsetHeight;
  // Centre on the badge, clamped inside the viewport.
  const left = Math.max(m, Math.min(r.left + r.width / 2 - pw / 2, window.innerWidth - pw - m));
  // Prefer below the badge; flip above if it would overflow the bottom edge.
  let top = r.bottom + 6;
  if (top + ph > window.innerHeight - m) top = r.top - ph - 6;
  if (top < m) top = m;
  qHintPop.style.left = `${left}px`;
  qHintPop.style.top = `${top}px`;
}

function hideQHint() {
  if (qHintPop) qHintPop.hidden = true;
}

// Delegated so badges rendered later (e.g. the Alter-role detail) work too.
document.addEventListener('mouseover', (e) => {
  const btn = e.target.closest?.('.q-hint');
  if (btn) positionQHint(btn);
});
document.addEventListener('mouseout', (e) => {
  if (e.target.closest?.('.q-hint')) hideQHint();
});
document.addEventListener('focusin', (e) => {
  const btn = e.target.closest?.('.q-hint');
  if (btn) positionQHint(btn);
});
document.addEventListener('focusout', (e) => {
  if (e.target.closest?.('.q-hint')) hideQHint();
});

// ------------------------------------------------------------------
// Alter role tab
// ------------------------------------------------------------------

let alterPassword = '';
/** @type {{kind:string, key:string}|null} scope-dialog context; null = adding a new privilege */
let scopeDialogCtx = null;
/** @type {Array<{text:string, ids:string[]}>} distinct comment values for the popup */
let commentVersions = [];
/** @type {Array<object>} one comment-editor model per version (parallel to commentVersions) */
let commentVersionEditors = [];

function clusterCategory(clusterId) {
  return state?.clusters?.find((c) => c.id === clusterId)?.category || '';
}

/** Group flat RoleMatch rows by login name; fullName = first non-empty in the group. */
function groupMatches(matches) {
  const map = new Map();
  for (const m of matches) {
    if (m.error || !m.loginName) continue;
    let g = map.get(m.loginName);
    if (!g) {
      g = { loginName: m.loginName, fullName: '', clusters: [] };
      map.set(m.loginName, g);
    }
    if (!g.fullName && m.fullName) g.fullName = m.fullName;
    g.clusters.push(m);
  }
  return [...map.values()].sort((a, b) => a.loginName.localeCompare(b.loginName));
}

/** Clusters covered by the current sidebar target selection (categories ∪ clusters). */
function resolveSelectedClusters() {
  const cats = new Set(getSelectedCategories());
  const ids = new Set(getSelectedClusterIDs());
  const out = [];
  const seen = new Set();
  for (const c of state?.clusters || []) {
    if ((ids.has(c.id) || cats.has(c.category)) && !seen.has(c.id)) {
      seen.add(c.id);
      out.push(c);
    }
  }
  return out;
}

async function runRoleSearch() {
  const app = backend();
  if (!app) {
    showToast('Wails backend not available', 'error');
    return;
  }
  const term = document.getElementById('alter-search-term').value.trim();
  if (term.length < 2) {
    showToast('Enter at least 2 characters to search', 'error');
    return;
  }
  // Only the selected clusters/groups are compared.
  const targets = { categoryIds: getSelectedCategories(), clusterIds: getSelectedClusterIDs() };
  const scopeClusters = resolveSelectedClusters();
  if (!scopeClusters.length) {
    showToast('Select at least one category or cluster in Target selection', 'error');
    return;
  }
  alterTargets = targets;
  alterScopeClusters = scopeClusters.map((c) => ({ clusterId: c.id, alias: c.alias, category: c.category }));

  // Reset any open detail view for a fresh search.
  alterSelected = null;
  alterDetails = [];
  document.getElementById('alter-detail').classList.add('hidden');
  document.getElementById('alter-results').innerHTML = '<p class="hint">Searching selected clusters…</p>';

  let matches;
  try {
    matches = await app.SearchRoles({ term, categoryIds: targets.categoryIds, clusterIds: targets.clusterIds, auth: getAuth() });
  } catch (e) {
    document.getElementById('alter-results').innerHTML = '';
    showToast(String(e), 'error');
    return;
  }

  const errors = (matches || []).filter((m) => m.error);
  alterGroups = groupMatches(matches || []);
  renderAlterErrors(errors);
  renderAlterResults();
}

function renderAlterErrors(errors) {
  const box = document.getElementById('alter-search-errors');
  if (!errors.length) {
    box.innerHTML = '';
    return;
  }
  box.innerHTML =
    '<strong>Some clusters could not be scanned:</strong>' +
    errors
      .map(
        (e) =>
          `<div class="alter-error-row">${escapeHtml(e.alias)} <span class="alter-cluster-host">(${escapeHtml(e.host)})</span>: ${escapeHtml(e.error)}</div>`
      )
      .join('');
}

function renderAlterResults() {
  const box = document.getElementById('alter-results');
  if (!alterGroups.length) {
    box.innerHTML = '<p class="hint">No matching roles found.</p>';
    return;
  }
  box.innerHTML = alterGroups
    .map((g) => {
      const labels = scopeLabelsHtml(describeScope(new Set(g.clusters.map((m) => m.clusterId))));
      const full = g.fullName
        ? `<span class="alter-fullname">${escapeHtml(g.fullName)}</span>`
        : '';
      return `<button type="button" class="alter-result-row" data-login="${escapeAttr(g.loginName)}">
        <span class="alter-login">${escapeHtml(g.loginName)}</span>
        ${full}
        <span class="alter-cluster-badges">${labels}</span>
      </button>`;
    })
    .join('');
}

async function pickUser(login) {
  alterSelected = login;
  resetEditMaps();

  document.getElementById('search-dialog')?.close();
  // The detail header shows which role is being edited; hide the empty-state prompt.
  document.getElementById('alter-current-hint')?.classList.add('hidden');
  const detail = document.getElementById('alter-detail');
  detail.classList.remove('hidden');
  detail.innerHTML = '<p class="hint">Loading role details…</p>';

  await reloadDetails();
}

/** Reload alterDetails for the selected login, preserving pending edits. */
async function reloadDetails() {
  const app = backend();
  let details;
  try {
    details = await app.LoadRoleDetails({
      loginName: alterSelected,
      categoryIds: alterTargets.categoryIds,
      clusterIds: alterTargets.clusterIds,
      auth: getAuth(),
    });
  } catch (e) {
    showToast(String(e), 'error');
    return;
  }
  alterDetails = (details || []).filter((d) => d.exists && !d.error);
  const errors = (details || []).filter((d) => d.error);
  loadCommentEditor();
  renderAlterDetail(errors);
}

/** Distinct sorted parent roles across every cluster the user exists on. */
function allPrivileges() {
  const s = new Set();
  for (const d of alterDetails) for (const p of d.parents || []) s.add(p);
  return [...s].sort();
}

/** Set of clusterIds on which the given role is currently granted. */
function clusterIdsWith(role) {
  return new Set(alterDetails.filter((d) => (d.parents || []).includes(role)).map((d) => d.clusterId));
}

/** Set of clusterIds on which the given attribute is currently enabled. */
function clusterIdsWithAttr(key) {
  return new Set(alterDetails.filter((d) => d.attributes && d.attributes[key]).map((d) => d.clusterId));
}

/** Set of clusterIds where setting `name` currently equals `value`. */
function clusterIdsWithConfig(name, value) {
  return new Set(alterDetails.filter((d) => d.settings && d.settings[name] === value).map((d) => d.clusterId));
}

/** Distinct {name,value} pairs across present clusters, sorted by name then value. */
function allSettings() {
  const seen = new Set();
  const out = [];
  for (const d of alterDetails) {
    for (const [name, value] of Object.entries(d.settings || {})) {
      const k = name + '=' + value;
      if (!seen.has(k)) {
        seen.add(k);
        out.push({ name, value });
      }
    }
  }
  return out.sort((a, b) => a.name.localeCompare(b.name) || a.value.localeCompare(b.value));
}

const CFG_SEP = ' '; // internal key separator (GUC names/values won't contain NUL)
// Row key is "name=value"; GUC names never contain '=', so split on the first one.
const cfgKey = (name, value) => name + '=' + value;
const cfgParse = (key) => {
  const i = key.indexOf('=');
  return { name: key.slice(0, i), value: key.slice(i + 1) };
};

/**
 * Describe a set of clusterIds against a universe (the selected comparison scope).
 * Completeness is judged per group only: if every universe cluster of a category is in
 * the set → one group label (named after the group); otherwise one label per matched
 * cluster of that group. There is no cross-group "all". Parts are ordered by the
 * configured category order and carry their category for colouring.
 */
function scopeUniverse() {
  return alterScopeClusters.length ? alterScopeClusters : alterDetails;
}

function describeScope(idSet, universe = scopeUniverse()) {
  const order = (state?.categories || []).map((c) => c.id);
  const byCat = new Map();
  for (const d of universe) {
    if (!byCat.has(d.category)) byCat.set(d.category, []);
    byCat.get(d.category).push(d);
  }
  const cats = [...byCat.keys()].sort((a, b) => {
    const ia = order.indexOf(a), ib = order.indexOf(b);
    return (ia < 0 ? 999 : ia) - (ib < 0 ? 999 : ib) || a.localeCompare(b);
  });
  const parts = [];
  for (const cat of cats) {
    const clusters = byCat.get(cat);
    const inSet = clusters.filter((d) => idSet.has(d.clusterId));
    if (!inSet.length) continue;
    if (inSet.length === clusters.length) {
      parts.push({ kind: 'group', cat, label: categoryLabel(cat) || cat });
    } else {
      for (const d of inSet) parts.push({ kind: 'cluster', cat, label: d.alias });
    }
  }
  return parts;
}

function scopeLabelsHtml(parts, extraCls = '') {
  // Pending labels (add=green / strike=red) carry only the overlay class so it wins;
  // normal labels carry data-cat so the generated per-group colour applies.
  return parts
    .map((p) => {
      const cat = extraCls ? '' : ` data-cat="${escapeAttr(p.cat)}"`;
      return `<span class="chip-scope scope-kind-${p.kind} ${extraCls}"${cat}>${escapeHtml(p.label)}</span>`;
    })
    .join('');
}

/** Canonical form of a comment for comparison: sorted-key JSON when valid, else raw. */
/** Built-in comment-key mapping, used when config omits comment_fields. */
const DEFAULT_COMMENT_FIELDS = [
  { key: 'full_name', label: 'Full name' },
  { key: 'e_mail', label: 'Email' },
];

/** Ordered configured comment fields, with the built-in default fallback. */
function commentFields() {
  const cf = state?.commentFields;
  return Array.isArray(cf) && cf.length ? cf : DEFAULT_COMMENT_FIELDS;
}

function canonicalComment(text) {
  const t = (text || '').trim();
  if (!t || t[0] !== '{') return text || '';
  try {
    return JSON.stringify(sortKeysDeep(JSON.parse(t)));
  } catch {
    return text || '';
  }
}

function sortKeysDeep(value) {
  if (Array.isArray(value)) return value.map(sortKeysDeep);
  if (value && typeof value === 'object') {
    const out = {};
    for (const k of Object.keys(value).sort()) out[k] = sortKeysDeep(value[k]);
    return out;
  }
  return value;
}

/** Consolidated identity: single value when all clusters agree, else "varies".
 *  Comments are compared canonically (JSON by value, ignoring formatting). */
function identityConsensus() {
  const fns = [...new Set(alterDetails.map((d) => d.fullName).filter(Boolean))];
  const rawComments = alterDetails.map((d) => d.comment).filter(Boolean);
  const canonSet = new Set(rawComments.map(canonicalComment));
  return {
    fullName: fns.length === 1 ? fns[0] : '',
    fullNameVaries: fns.length > 1,
    comment: rawComments.length ? rawComments[0] : '',
    commentVaries: canonSet.size > 1,
    hasComment: rawComments.length >= 1,
  };
}

/** Parse a role comment. isObject=true only for a JSON object (not array/scalar/plain text).
 *  Single source of truth for reading structured comments. */
function parseCommentObject(comment) {
  const t = (comment || '').trim();
  if (!t || t[0] !== '{') return { isObject: false, obj: {} };
  try {
    const o = JSON.parse(t);
    if (o && typeof o === 'object' && !Array.isArray(o)) return { isObject: true, obj: o };
  } catch {
    /* not JSON */
  }
  return { isObject: false, obj: {} };
}

/** Read a string field from a JSON role comment (e.g. full_name, email); '' if absent. */
function parseCommentField(comment, key) {
  const { isObject, obj } = parseCommentObject(comment);
  return isObject && typeof obj[key] === 'string' ? obj[key].trim() : '';
}

/** Preferred comment mode for an EMPTY comment (create / no-comment role): 'fields' | 'raw'. */
function preferredCommentView() {
  return state?.ui?.commentDefaultView === 'raw' ? 'raw' : 'fields';
}

/** Consensus comment across the loaded clusters + whether they disagree. */
function commentConsensus() {
  const raws = alterDetails.map((d) => d.comment).filter(Boolean);
  const canon = new Set(raws.map(canonicalComment));
  return { comment: raws.length ? raws[0] : '', varies: canon.size > 1, hasComment: raws.length >= 1 };
}

/** Build the editor model from a comment string. Configured fields always render (blank if
 *  absent); extra string keys render too (labeled by raw key); non-string keys stay in
 *  baseObj only (Raw-editable). Default mode = Fields for a JSON object, else Raw. */
function editorFromComment(comment, varies) {
  const { isObject, obj } = parseCommentObject(comment);
  const labels = {};
  const values = {};
  const shownKeys = [];
  const seen = new Set();
  const readonly = new Set(); // keys whose value isn't a string: shown but only Raw-editable
  const addKey = (key, label) => {
    if (seen.has(key)) return;
    seen.add(key);
    shownKeys.push(key);
    labels[key] = label;
    if (isObject && key in obj && typeof obj[key] !== 'string') {
      // Non-string value (number/bool/array/object): display it (JSON) but read-only, so it's
      // visible per the design and its type is preserved via baseObj on save.
      values[key] = JSON.stringify(obj[key]);
      readonly.add(key);
    } else {
      values[key] = isObject && typeof obj[key] === 'string' ? obj[key] : '';
    }
  };
  // Configured fields always render (in order); then every other key present in the comment.
  for (const f of commentFields()) addKey(f.key, f.label || f.key);
  if (isObject) for (const k of Object.keys(obj)) addKey(k, k);
  // Mode: JSON object -> Fields; non-JSON *content* -> Raw; empty -> the configured preference.
  const hasPlainText = !isObject && !!(comment || '').trim();
  let mode;
  if (isObject) mode = 'fields';
  else if (hasPlainText) mode = 'raw';
  else mode = preferredCommentView();
  return {
    mode,
    baseObj: isObject ? obj : {},
    isObject,
    raw: comment || '',
    shownKeys,
    labels,
    values,
    readonly,
    varies,
  };
}

function freshEditor() {
  return editorFromComment('', false);
}

/** Load #role-login + build the comment editor model. Called on mode entry / role load only
 *  — NOT on every re-render, so pending user edits aren't clobbered. */
function loadCommentEditor() {
  const login = document.getElementById('role-login');
  if (!login) return;
  if (isCreateMode()) {
    login.value = '';
    commentEditor = freshEditor();
  } else {
    login.value = alterSelected || '';
    const c = commentConsensus();
    commentEditor = editorFromComment(c.varies ? '' : c.comment, c.varies);
  }
  renderCommentEditor();
}

/** Paint the comment editor (toggle state + per-key value inputs + notice) from commentEditor. */
function renderCommentEditor() {
  const fieldsBox = document.getElementById('rce-fields');
  const raw = document.getElementById('rce-raw');
  const notice = document.getElementById('rce-notice');
  if (!fieldsBox || !raw || !notice) return;
  const e = commentEditor;
  document
    .querySelectorAll('#rce-mode .seg-btn')
    .forEach((b) => b.classList.toggle('active', b.dataset.mode === e.mode));

  const plainTextLoaded = !e.isObject && !!e.raw.trim();
  const fieldsDisabled = e.varies || (plainTextLoaded && !isCreateMode());
  fieldsBox.classList.toggle('hidden', e.mode !== 'fields');
  raw.classList.toggle('hidden', e.mode !== 'raw');
  raw.value = e.raw;

  fieldsBox.innerHTML = commentFieldInputsHtml(e, null, fieldsDisabled);

  if (e.varies) {
    notice.textContent = 'Comments differ across clusters — reconcile via "View / edit comments" below.';
  } else if (e.mode === 'fields' && plainTextLoaded) {
    notice.textContent = 'This comment is plain text — switch to Raw to edit it.';
  } else {
    notice.textContent = '';
  }
}

/** The comment an editor model currently represents (may be plain text in raw mode). */
function assembleCommentFrom(e) {
  if (e.mode === 'raw') return e.raw.trim();
  const out = { ...e.baseObj };
  for (const k of e.shownKeys) {
    if (e.readonly && e.readonly.has(k)) continue; // non-string value: keep baseObj's typed value
    const v = (e.values[k] || '').trim();
    if (v) out[k] = v;
    else delete out[k];
  }
  return Object.keys(out).length ? JSON.stringify(sortKeysDeep(out)) : '';
}
function assembleComment() {
  return assembleCommentFrom(commentEditor);
}

/** Switch an editor model between Fields<->Raw without losing data (mutates e). */
function switchEditorMode(e, mode) {
  if (mode === e.mode) return;
  if (mode === 'raw') {
    e.raw = assembleCommentFrom(e); // serialize field state so nothing is lost
  } else {
    const rebuilt = editorFromComment(e.raw, e.varies);
    e.baseObj = rebuilt.baseObj;
    e.isObject = rebuilt.isObject;
    e.shownKeys = rebuilt.shownKeys;
    e.labels = rebuilt.labels;
    e.values = rebuilt.values;
    e.readonly = rebuilt.readonly;
  }
  e.mode = mode;
}
function switchCommentMode(mode) {
  switchEditorMode(commentEditor, mode);
  renderCommentEditor();
}

/** Field value inputs for an editor model, namespaced with data-cv-idx for reuse in dialogs.
 *  Non-string keys (e.readonly) render disabled with a note — visible but edited via Raw. */
function commentFieldInputsHtml(e, idx, disabled) {
  const idxAttr = idx == null ? '' : ` data-cv-idx="${idx}"`;
  return e.shownKeys
    .map((k) => {
      const ro = e.readonly && e.readonly.has(k);
      const note = ro
        ? ` <button type="button" class="q-hint q-warn" tabindex="0" aria-label="Non-text value — edit in Raw" data-hint="Non-text value — edit in Raw">⚠</button>`
        : '';
      return `
    <label class="rce-field"><span class="rce-field-label">${escapeHtml(e.labels[k] || k)}${note}</span>
      <input type="text"${idxAttr} data-cf-key="${escapeAttr(k)}" value="${escapeAttr(e.values[k] || '')}"
        autocapitalize="none" autocomplete="off" spellcheck="false" ${disabled || ro ? 'disabled' : ''} />
    </label>`;
    })
    .join('');
}

function renderAlterDetail(errors = []) {
  const root = document.getElementById('alter-detail');
  const identity = document.getElementById('role-identity');
  const present = document.getElementById('role-present');
  const hint = document.getElementById('alter-current-hint');
  const login = document.getElementById('role-login');
  const create = isCreateMode();

  // Identity block + "Present on" (edit only, above the form) + empty-state hint.
  if (create) {
    identity.classList.remove('hidden');
    hint.classList.add('hidden');
    if (login) login.readOnly = false;
    present.classList.add('hidden');
    present.innerHTML = '';
  } else if (alterSelected && alterDetails.length) {
    identity.classList.remove('hidden');
    hint.classList.add('hidden');
    if (login) login.readOnly = true;
    const presentLabels = scopeLabelsHtml(describeScope(new Set(alterDetails.map((d) => d.clusterId))));
    present.innerHTML = `<span class="alter-meta-label">Present on</span><span class="alter-cluster-badges">${presentLabels}</span>`;
    present.classList.remove('hidden');
  } else {
    identity.classList.add('hidden');
    hint.classList.remove('hidden');
    present.classList.add('hidden');
    present.innerHTML = '';
  }

  if (!create && !alterSelected) {
    root.classList.add('hidden');
    root.innerHTML = '';
    return;
  }
  root.classList.remove('hidden');

  if (!alterDetails.length) {
    root.innerHTML = create
      ? '<p class="hint">Select at least one target cluster to define privileges, attributes and settings.</p>'
      : `<p class="hint">Role <strong>${escapeHtml(alterSelected)}</strong> was not found on any reachable cluster.</p>` +
        renderDetailErrors(errors);
    return;
  }

  const id = identityConsensus();

  // The inline comment editor handles the consistent case; hide it when comments vary across
  // clusters (it can't sensibly edit them all at once) and reconcile via the Comments dialog.
  const commentEditorEl = document.getElementById('role-comment-editor');
  if (commentEditorEl) commentEditorEl.classList.toggle('hidden', !create && id.commentVaries);

  // Edit-only: when comments vary, surface a "reconcile per cluster" entry (Comments dialog).
  // When consistent, the inline editor above is the only comment UI (no redundant section).
  let editHead = '';
  if (!create) {
    editHead = renderDetailErrors(errors);
    if (id.commentVaries) {
      const staged = alterDetails.some(
        (d) => commentOverrides.has(d.clusterId) && canonicalComment(commentOverrides.get(d.clusterId)) !== canonicalComment(d.comment || '')
      );
      editHead += `
    <div class="alter-section">
      <div class="alter-privs-label">Comment</div>
      <div class="alter-add-priv">
        <button type="button" class="small btn-two-line${staged ? ' is-added' : ''}" id="btn-alter-comments">Comments differ across clusters<br>view / edit per cluster</button>
      </div>
    </div>`;
    }
  }

  const existing = allPrivileges();
  const existingSet = new Set(existing);
  const newRoles = [...alterAdd.keys()].filter((r) => !existingSet.has(r));
  const privRows = existing
    .concat(newRoles)
    .map((r) => scopeRowHtml('priv', r, r, clusterIdsWith(r), alterAdd.get(r) || new Set(), alterRevoke.get(r) || new Set()));
  const privHtml = privRows.length ? privRows.join('') : '<p class="hint">No privileges.</p>';

  const attrRows = ROLE_ATTRIBUTES.map((a) =>
    scopeRowHtml('attr', a.key, a.label, clusterIdsWithAttr(a.key), alterAttrAdd.get(a.key) || new Set(), alterAttrRemove.get(a.key) || new Set())
  ).join('');

  // SETTINGS: existing (name,value) pairs plus any brand-new pending sets.
  const settingPairs = allSettings();
  const seenCfg = new Set(settingPairs.map((p) => cfgKey(p.name, p.value)));
  for (const key of alterConfigSet.keys()) {
    if (!seenCfg.has(key)) {
      seenCfg.add(key);
      settingPairs.push(cfgParse(key));
    }
  }
  const cfgRows = settingPairs.map((p) => configRowHtml(p.name, p.value)).join('');
  const cfgHtml = cfgRows || '<p class="hint">No settings.</p>';

  root.innerHTML = `
    ${editHead}
    <div class="alter-section">
      <div class="alter-privs-label">Privileges ${hintBadge('Each privilege shows the clusters/groups it is granted on. Use ✎ to add or remove clusters, × to revoke everywhere.')}</div>
      <div class="scope-rows" id="alter-privs">${privHtml}</div>
      <div class="alter-add-priv">
        <button type="button" class="list-add" id="btn-alter-add">Add privilege…</button>
      </div>
    </div>

    <div class="alter-section">
      <div class="alter-privs-label">Attributes ${hintBadge('Each attribute shows where it is enabled. Use ✎ to enable/disable per cluster, × to disable everywhere.')}</div>
      <div class="scope-rows" id="alter-attrs">${attrRows}</div>
    </div>

    <div class="alter-section">
      <div class="alter-privs-label">Settings ${hintBadge('Role GUCs (ALTER ROLE … SET/RESET). Use ✎ to set on chosen clusters, × to reset everywhere it has that value.')}</div>
      <div class="scope-rows" id="alter-configs">${cfgHtml}</div>
      <div class="alter-add-priv">
        <button type="button" class="list-add" id="btn-alter-add-config">Add setting…</button>
      </div>
    </div>

    <div class="alter-section">
      <div class="alter-privs-label">Password</div>
      <div class="alter-password">
        <input type="password" id="alter-password" autocapitalize="none" autocomplete="off" placeholder="new password" />
        <label class="inline"><input type="checkbox" id="alter-do-pw"${alterDoPassword ? ' checked' : ''} /> Change password</label>
      </div>
    </div>`;

  const pwInput = /** @type {HTMLInputElement} */ (document.getElementById('alter-password'));
  if (pwInput) pwInput.value = alterPassword;
  updateOpsFooter();
}

/** Show the right pinned footer for the active op: Create → Run; Alter → Save/Remove
 *  (only once a role is loaded). Hide the footer entirely when neither applies. */
function updateOpsFooter() {
  const isCreate = isCreateMode();
  const showAlter = !isCreate && !!alterSelected && alterDetails.length > 0;
  document.getElementById('create-run-bar')?.classList.toggle('hidden', !isCreate);
  document.getElementById('alter-actions')?.classList.toggle('hidden', !showAlter);
  document.getElementById('ops-footer')?.classList.toggle('hidden', !isCreate && !showAlter);
}

function renderDetailErrors(errors) {
  if (!errors || !errors.length) return '';
  return (
    '<div class="alter-errors"><strong>Unreachable clusters:</strong>' +
    errors
      .map((e) => `<div class="alter-error-row">${escapeHtml(e.alias)}: ${escapeHtml(e.error)}</div>`)
      .join('') +
    '</div>'
  );
}

/** Subtract set b from set a (returns a new Set). */
function setMinus(a, b) {
  const out = new Set();
  for (const x of a) if (!b.has(x)) out.add(x);
  return out;
}

/**
 * One row for a privilege or attribute: name on the left, scope labels on the right,
 * then actions. kind is 'priv' or 'attr'. curSet = current clusters; addSet = pending
 * grants/enables; revSet = pending revokes/disables.
 */
function scopeRowHtml(kind, key, name, curSet, addSet, revSet) {
  const isNew = curSet.size === 0;
  const k = escapeAttr(key);
  const pending = addSet.size > 0 || revSet.size > 0;
  const kept = setMinus(curSet, revSet);
  const emptyNote = kind === 'attr' ? '<span class="hint scope-off">off</span>' : '<span class="hint">none</span>';

  const keptLabels = scopeLabelsHtml(describeScope(kept));
  const addLabels = addSet.size ? scopeLabelsHtml(describeScope(addSet), 'chip-scope-add') : '';
  const revLabels = revSet.size ? scopeLabelsHtml(describeScope(revSet), 'chip-scope-strike') : '';
  const labels = keptLabels + addLabels + revLabels || emptyNote;

  // All three actions always render as square buttons beside the row; the ones that don't
  // apply are disabled (greyed, inert to the mouse) rather than hidden, so layout is stable.
  const editBtn = `<button type="button" class="scope-act" data-kind="${kind}" data-act="scope" data-key="${k}" title="Edit clusters">✎</button>`;

  const verbRemove = kind === 'attr' ? 'Disable everywhere' : kind === 'config' ? 'Reset everywhere' : 'Revoke everywhere';
  let xAct = 'revoke';
  let xTitle = verbRemove;
  if (isNew && addSet.size) {
    xAct = 'deladd';
    xTitle = 'Cancel';
  }
  // × only acts while a grant remains (kept) or a pending add exists; once everything
  // is revoked/cancelled it has nothing left to do, so it goes inactive.
  const xOn = kept.size > 0 || addSet.size > 0;
  const xBtn = `<button type="button" class="scope-act" data-kind="${kind}" data-act="${xAct}" data-key="${k}" title="${escapeAttr(xTitle)}"${xOn ? '' : ' disabled'}>×</button>`;

  const resetBtn = `<button type="button" class="scope-act" data-kind="${kind}" data-act="reset" data-key="${k}" title="Discard pending changes"${pending ? '' : ' disabled'}>↺</button>`;

  const fullyRemoved = kept.size === 0 && addSet.size === 0 && revSet.size > 0;
  const stateCls = fullyRemoved ? 'is-removed' : isNew && addSet.size ? 'is-added' : pending ? 'is-extending' : '';
  return `<div class="scope-line">
    <div class="scope-row ${stateCls}">
      <span class="scope-row-name">${escapeHtml(name)}</span>
      <span class="scope-row-labels">${labels}</span>
    </div>
    <div class="scope-row-actions">${editBtn}${xBtn}${resetBtn}</div>
  </div>`;
}

/** A SETTINGS row (name=value): current scope = clusters where the setting has that value;
 *  a cluster is "removed" if pending-reset for the name or pending-set to a different value. */
function configRowHtml(name, value) {
  const key = cfgKey(name, value);
  const cur = clusterIdsWithConfig(name, value);
  const add = alterConfigSet.get(key) || new Set();
  const resetIds = alterConfigReset.get(name) || new Set();
  const rev = new Set();
  for (const cid of cur) {
    if (resetIds.has(cid)) {
      rev.add(cid);
      continue;
    }
    for (const [k, ids] of alterConfigSet) {
      if (k !== key && cfgParse(k).name === name && ids.has(cid)) {
        rev.add(cid);
        break;
      }
    }
  }
  return scopeRowHtml('config', key, `${name} = ${value}`, cur, add, rev);
}

// --- Scope dialog (add a new privilege, or extend a privilege/attribute) ---

/** ctx: null → new privilege; {kind:'priv',key} → edit privilege; {kind:'attr',key} → edit attribute. */
function openScopeDialog(ctx) {
  scopeDialogCtx = ctx || null;
  const dlg = document.getElementById('scope-dialog');
  const title = document.getElementById('scope-dialog-title');
  const roleLabel = document.getElementById('scope-role-label');
  const roleInput = /** @type {HTMLInputElement} */ (document.getElementById('scope-role'));
  const cnameLabel = document.getElementById('scope-cname-label');
  const cvalueLabel = document.getElementById('scope-cvalue-label');
  const ok = document.getElementById('scope-dialog-ok');

  roleLabel.classList.add('hidden');
  cnameLabel.classList.add('hidden');
  cvalueLabel.classList.add('hidden');
  document.getElementById('scope-cname').readOnly = false;
  document.getElementById('scope-preconfigured')?.classList.add('hidden');

  if (!ctx) {
    title.textContent = 'Add privilege';
    roleLabel.classList.remove('hidden');
    roleInput.value = '';
    renderScopePreconfigured();
    ok.textContent = 'Add';
  } else if (ctx.kind === 'config' && ctx.isNew) {
    title.textContent = 'Add setting';
    cnameLabel.classList.remove('hidden');
    cvalueLabel.classList.remove('hidden');
    document.getElementById('scope-cname').value = '';
    document.getElementById('scope-cvalue').value = '';
    ok.textContent = 'Add';
  } else if (ctx.kind === 'config') {
    // Edit an existing setting: value editable, name fixed (read-only).
    const { name, value } = cfgParse(ctx.key);
    title.textContent = name;
    cnameLabel.classList.remove('hidden');
    cvalueLabel.classList.remove('hidden');
    const cname = document.getElementById('scope-cname');
    cname.value = name;
    cname.readOnly = true;
    document.getElementById('scope-cvalue').value = value;
    ok.textContent = 'Apply';
  } else if (ctx.kind === 'attr') {
    const a = ROLE_ATTRIBUTES.find((x) => x.key === ctx.key);
    title.textContent = a ? a.label : ctx.key;
    ok.textContent = 'Apply';
  } else {
    title.textContent = ctx.key;
    ok.textContent = 'Apply';
  }
  buildScopeTargets(ctx);
  dlg.showModal();
  if (!ctx) roleInput.focus();
  else if (ctx.kind === 'config' && ctx.isNew) document.getElementById('scope-cname').focus();
  else if (ctx.kind === 'config') document.getElementById('scope-cvalue').focus();
}

/** Preconfigured-group checkboxes shown when adding a new privilege. */
function renderScopePreconfigured() {
  const box = document.getElementById('scope-preconfigured');
  if (!box) return;
  const roles = preconfiguredParentRoles();
  if (!roles.length) {
    box.innerHTML = '';
    box.classList.add('hidden');
    return;
  }
  box.innerHTML =
    '<span class="picker-label">Add preconfigured:</span>' +
    roles
      .map(
        (r) =>
          `<button type="button" class="pick-chip" data-role="${escapeAttr(r)}">${escapeHtml(r)}</button>`
      )
      .join('');
  box.classList.remove('hidden');
}

/** Desired-state set currently reflected for a ctx: (current − pendingRevoke) ∪ pendingAdd. */
function scopeDesired(ctx) {
  if (!ctx) return new Set();
  if (ctx.kind === 'config') {
    if (ctx.isNew) return new Set();
    const { name, value } = cfgParse(ctx.key);
    const cur = clusterIdsWithConfig(name, value);
    const rev = alterConfigReset.get(name) || new Set();
    const add = alterConfigSet.get(ctx.key) || new Set();
    const desired = setMinus(cur, rev);
    for (const cid of add) desired.add(cid);
    return desired;
  }
  const cur = ctx.kind === 'attr' ? clusterIdsWithAttr(ctx.key) : clusterIdsWith(ctx.key);
  const add = (ctx.kind === 'attr' ? alterAttrAdd : alterAdd).get(ctx.key) || new Set();
  const rev = (ctx.kind === 'attr' ? alterAttrRemove : alterRevoke).get(ctx.key) || new Set();
  const desired = setMinus(cur, rev);
  for (const cid of add) desired.add(cid);
  return desired;
}

function buildScopeTargets(ctx) {
  const box = document.getElementById('scope-targets');
  const desired = scopeDesired(ctx);

  const byCat = new Map();
  for (const d of alterDetails) {
    if (!byCat.has(d.category)) byCat.set(d.category, []);
    byCat.get(d.category).push(d);
  }
  let html = '';
  for (const [cat, clusters] of byCat) {
    html += `<div class="scope-group">
      <label class="scope-group-head"><input type="checkbox" class="scope-group-check" data-cat="${escapeAttr(cat)}" /> All ${escapeHtml(categoryLabel(cat) || cat)}</label>
      <div class="scope-clusters">`;
    for (const d of clusters) {
      html += `<label class="scope-cluster">
        <input type="checkbox" class="scope-cluster-check" data-cluster="${escapeAttr(d.clusterId)}" data-cat="${escapeAttr(cat)}" ${desired.has(d.clusterId) ? 'checked' : ''} />
        ${escapeHtml(d.alias)}
      </label>`;
    }
    html += `</div></div>`;
  }
  box.innerHTML = html;
}

function confirmScopeDialog() {
  const ctx = scopeDialogCtx;
  const desired = new Set(
    [...document.querySelectorAll('#scope-targets .scope-cluster-check')]
      .filter((cb) => cb.checked)
      .map((cb) => cb.dataset.cluster)
  );

  if (ctx && ctx.kind === 'config') {
    confirmConfigScope(ctx, desired);
    return;
  }

  // New privilege: one or more roles at once (typed name + any picked preconfigured groups).
  if (!ctx) {
    const roles = [];
    const typed = document.getElementById('scope-role').value.trim();
    if (typed) {
      if (!ROLE_NAME_RE.test(typed)) {
        showToast('Invalid role name: use letters, digits, underscore', 'error');
        return;
      }
      roles.push(typed);
    }
    for (const chip of document.querySelectorAll('#scope-preconfigured .pick-chip.active')) {
      if (!roles.includes(chip.dataset.role)) roles.push(chip.dataset.role);
    }
    if (!roles.length) {
      showToast('Enter a role name or pick at least one preconfigured group', 'error');
      return;
    }
    for (const key of roles) {
      const cur = clusterIdsWith(key);
      const grant = setMinus(desired, cur);
      const revoke = setMinus(cur, desired);
      if (grant.size) alterAdd.set(key, grant);
      else alterAdd.delete(key);
      if (revoke.size) alterRevoke.set(key, revoke);
      else alterRevoke.delete(key);
    }
    document.getElementById('scope-dialog').close();
    renderAlterDetail();
    return;
  }

  // Edit an existing privilege/attribute (single key).
  const key = ctx.key;
  const isAttr = ctx.kind === 'attr';
  const addMap = isAttr ? alterAttrAdd : alterAdd;
  const revMap = isAttr ? alterAttrRemove : alterRevoke;
  const cur = isAttr ? clusterIdsWithAttr(key) : clusterIdsWith(key);

  // Diff desired vs current → grant (desired−cur) and revoke (cur−desired).
  const grant = setMinus(desired, cur);
  const revoke = setMinus(cur, desired);
  if (grant.size) addMap.set(key, grant);
  else addMap.delete(key);
  if (revoke.size) revMap.set(key, revoke);
  else revMap.delete(key);

  document.getElementById('scope-dialog').close();
  renderAlterDetail();
}

/** Apply the scope dialog for a role SETTING (name=value): SET on desired clusters,
 *  RESET on clusters that had this value but are no longer desired. */
function confirmConfigScope(ctx, desired) {
  let name;
  let origValue = null; // the value of the row being edited (its baseline clusters)
  if (ctx.isNew) {
    name = document.getElementById('scope-cname').value.trim();
    if (!/^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$/.test(name)) {
      showToast('Invalid setting name (letters, digits, underscore, optional dot)', 'error');
      return;
    }
  } else {
    ({ name, value: origValue } = cfgParse(ctx.key));
  }
  // The value comes from the (editable) input in both add and edit modes; the name is
  // fixed when editing.
  const value = document.getElementById('scope-cvalue').value;
  const key = cfgKey(name, value);
  const curNew = clusterIdsWithConfig(name, value); // clusters already at this value
  // Reset baseline: the edited row's clusters (its original value), or — when adding —
  // the clusters that already carry this exact value.
  const baseline = ctx.isNew ? curNew : clusterIdsWithConfig(name, origValue);
  const set = new Set(alterConfigSet.get(key) || []);
  const reset = new Set(alterConfigReset.get(name) || []);

  const clearOtherSets = (cid) => {
    for (const [k, ids] of alterConfigSet) {
      if (k !== key && cfgParse(k).name === name && ids.has(cid)) {
        ids.delete(cid);
        if (!ids.size) alterConfigSet.delete(k);
      }
    }
  };
  for (const cid of desired) {
    if (curNew.has(cid)) set.delete(cid); // already this value → no SET needed
    else set.add(cid);
    reset.delete(cid);
    clearOtherSets(cid);
  }
  for (const cid of baseline) {
    if (!desired.has(cid)) {
      reset.add(cid);
      set.delete(cid);
    }
  }
  if (set.size) alterConfigSet.set(key, set);
  else alterConfigSet.delete(key);
  if (reset.size) alterConfigReset.set(name, reset);
  else alterConfigReset.delete(name);

  document.getElementById('scope-dialog').close();
  renderAlterDetail();
}

// --- Comments dialog (group clusters by comment, edit per group) ---

function buildCommentVersions() {
  // Group by canonical value so JSON comments differing only in formatting collapse into one
  // version. Seed each group from any staged override (so reopening shows pending edits),
  // else from the current DB comment.
  const map = new Map();
  for (const d of alterDetails) {
    const key = canonicalComment(d.comment);
    if (!map.has(key)) {
      const text = commentOverrides.has(d.clusterId) ? commentOverrides.get(d.clusterId) : d.comment || '';
      map.set(key, { text, ids: [] });
    }
    map.get(key).ids.push(d.clusterId);
  }
  commentVersions = [...map.values()];
  // One 2-mode editor model per version (same Fields/Raw editor as the inline one).
  commentVersionEditors = commentVersions.map((v) => editorFromComment(v.text, false));
}

function openCommentsDialog() {
  buildCommentVersions();
  renderCommentsDialog();
  const dlg = document.getElementById('comments-dialog');
  dlg.showModal();
  // showModal() auto-focuses the first focusable element — the header "?" badge — which would
  // pop its hint open. Move focus to the first editable control (or OK) and clear the hint.
  const first = dlg.querySelector('#comments-list textarea:not(.hidden), #comments-list input:not([disabled])');
  (first || document.getElementById('comments-dialog-ok'))?.focus();
  hideQHint();
}

function renderCommentsDialog() {
  const box = document.getElementById('comments-list');
  box.innerHTML = commentVersions
    .map((v, i) => {
      const e = commentVersionEditors[i];
      const labels = scopeLabelsHtml(describeScope(new Set(v.ids)));
      const plainText = !e.isObject && !!e.raw.trim();
      const fieldsDisabled = plainText; // plain text: edit in Raw (don't silently convert)
      const notice = e.mode === 'fields' && plainText ? 'This comment is plain text — switch to Raw to edit it.' : '';
      return `<div class="comment-version">
        <div class="comment-scope">${labels || '<span class="hint">no clusters</span>'}</div>
        <div class="rce-toolbar">
          <div class="segmented rce-mode" data-cv-mode="${i}" role="group" aria-label="Comment editing mode">
            <button type="button" class="seg-btn ${e.mode === 'fields' ? 'active' : ''}" data-mode="fields">Fields</button>
            <button type="button" class="seg-btn ${e.mode === 'raw' ? 'active' : ''}" data-mode="raw">Raw</button>
          </div>
        </div>
        <div class="rce-fields${e.mode !== 'fields' ? ' hidden' : ''}" data-cv-fields="${i}">${commentFieldInputsHtml(e, i, fieldsDisabled)}</div>
        <textarea class="rce-raw comment-edit${e.mode !== 'raw' ? ' hidden' : ''}" data-cv-raw="${i}" rows="4" autocapitalize="none" spellcheck="false">${escapeHtml(e.raw)}</textarea>
        <p class="hint rce-notice">${notice}</p>
      </div>`;
    })
    .join('');
}

/** OK: stage each version's assembled comment into commentOverrides (per cluster) and close.
 *  Nothing is sent here — the edits publish with the other changes on "Save changes". */
function commitCommentsDialog() {
  commentVersions.forEach((v, i) => {
    const desired = assembleCommentFrom(commentVersionEditors[i]);
    for (const cid of v.ids) commentOverrides.set(cid, desired);
  });
  document.getElementById('comments-dialog').close();
  renderAlterDetail(); // reflect the staged/"edited" state in the differ section
}

function buildAlterRequests() {
  /** @type {Array<{op:string, clusterId:string, params:object}>} */
  const requests = [];
  // The comment editor owns the consistent case: assemble the desired comment once and emit
  // set_comment per cluster where it actually changes. When comments vary across clusters the
  // inline editor is disabled and per-cluster desired comments come from the Comments dialog
  // (commentOverrides). In create mode this runs after create_role, making set_comment
  // authoritative.
  const desiredInline = commentEditor.varies ? null : assembleComment();

  for (const d of alterDetails) {
    const parents = d.parents || [];
    const toGrant = [...alterAdd.entries()]
      .filter(([role, ids]) => ids.has(d.clusterId) && !parents.includes(role))
      .map(([role]) => role);
    const toRevoke = [...alterRevoke.entries()]
      .filter(([role, ids]) => ids.has(d.clusterId) && parents.includes(role))
      .map(([role]) => role);
    if (toGrant.length) {
      requests.push({
        op: 'grant_parents',
        clusterId: d.clusterId,
        params: { grantParents: { loginName: alterSelected, parentRoles: toGrant.join(',') } },
      });
    }
    if (toRevoke.length) {
      requests.push({
        op: 'revoke_parents',
        clusterId: d.clusterId,
        params: { revokeParents: { loginName: alterSelected, parentRoles: toRevoke.join(',') } },
      });
    }
    if (alterDoPassword && alterPassword) {
      requests.push({
        op: 'change_password',
        clusterId: d.clusterId,
        params: { changePassword: { loginName: alterSelected, newPassword: alterPassword } },
      });
    }

    // Attribute enable/disable, one ALTER ROLE per attribute per cluster.
    for (const a of ROLE_ATTRIBUTES) {
      const on = !!(d.attributes && d.attributes[a.key]);
      const enableIds = alterAttrAdd.get(a.key);
      const disableIds = alterAttrRemove.get(a.key);
      if (enableIds && enableIds.has(d.clusterId) && !on) {
        requests.push({
          op: 'set_attribute',
          clusterId: d.clusterId,
          params: { setAttribute: { loginName: alterSelected, attribute: a.on } },
        });
      } else if (disableIds && disableIds.has(d.clusterId) && on) {
        requests.push({
          op: 'set_attribute',
          clusterId: d.clusterId,
          params: { setAttribute: { loginName: alterSelected, attribute: a.off } },
        });
      }
    }

    // Settings: SET name=value where pending & not already that value; RESET where pending.
    const settings = d.settings || {};
    for (const [key, ids] of alterConfigSet) {
      if (!ids.has(d.clusterId)) continue;
      const { name, value } = cfgParse(key);
      if (settings[name] !== value) {
        requests.push({
          op: 'set_config',
          clusterId: d.clusterId,
          params: { setConfig: { loginName: alterSelected, configName: name, configValue: value } },
        });
      }
    }
    for (const [name, ids] of alterConfigReset) {
      if (ids.has(d.clusterId) && Object.prototype.hasOwnProperty.call(settings, name)) {
        requests.push({
          op: 'reset_config',
          clusterId: d.clusterId,
          params: { resetConfig: { loginName: alterSelected, configName: name } },
        });
      }
    }

    // Comment via set_comment: a staged per-cluster override (varies case, from the Comments
    // dialog) wins; otherwise the inline editor's assembled comment. Emit only on a real change.
    const desiredComment = commentOverrides.has(d.clusterId) ? commentOverrides.get(d.clusterId) : desiredInline;
    if (desiredComment !== null && canonicalComment(desiredComment) !== canonicalComment(d.comment || '')) {
      requests.push({
        op: 'set_comment',
        clusterId: d.clusterId,
        params: { setComment: { loginName: alterSelected, comment: desiredComment } },
      });
    }
  }
  return requests;
}

/** Warn if any emitted set_comment stores a non-empty, non-JSON comment (plain text). */
function warnIfPlainTextComment(requests) {
  const plain = requests.some(
    (r) => r.op === 'set_comment' && r.params.setComment.comment && !parseCommentObject(r.params.setComment.comment).isObject
  );
  if (plain) showToast('Comment saved as plain text (not JSON)', 'warn');
}

async function saveAlterations() {
  const app = backend();
  if (!app) {
    showToast('Wails backend not available', 'error');
    return;
  }
  // Validate: password required when "Change password" is checked.
  if (alterDoPassword && !alterPassword) {
    showToast('Enter a new password ("Change password" is checked)', 'error');
    return;
  }

  const requests = buildAlterRequests();
  if (!requests.length) {
    showToast('No changes to save', 'error');
    return;
  }
  warnIfPlainTextComment(requests);

  const ok = await executeAlterRequests(requests, 'Changes saved');
  if (!ok) return;
  // Applied — clear pending edits and refresh from the DB.
  resetEditMaps();
  if (alterSelected) await reloadDetails();
}

/** Remove the role on every cluster where it exists (immediate, red button). */
async function removeRole() {
  const app = backend();
  if (!app) {
    showToast('Wails backend not available', 'error');
    return;
  }
  if (!alterDetails.length) return;
  const okConfirm = await askConfirm(
    'Remove role',
    `Remove "${alterSelected}" on all clusters where it exists? This cannot be undone from the app.`
  );
  if (!okConfirm) return;
  const requests = alterDetails.map((d) => ({
    op: 'remove_role',
    clusterId: d.clusterId,
    params: { removeRole: { loginName: alterSelected } },
  }));
  const ok = await executeAlterRequests(requests, 'Role removed');
  if (!ok) return;
  await reloadDetails();
}

/** Run a list of per-cluster operations, gating production and reporting status.
 *  Returns false when blocked/cancelled before execution. */
async function executeAlterRequests(requests, successMsg) {
  const app = backend();
  const targetIds = [...new Set(requests.map((r) => r.clusterId))];
  const prodInvolved = targetIds.some((id) => categoryConfirm(clusterCategory(id)));
  if (prodInvolved) {
    const ok = await askConfirm('Production', 'This action includes PRODUCTION clusters. Continue?');
    if (!ok) return false;
  }

  const auth = getAuth();
  const results = [];
  for (const r of requests) {
    try {
      const res = await app.RunOperation({
        operation: r.op,
        categoryIds: [],
        clusterIds: [r.clusterId],
        auth,
        confirmProduction: true,
        ...r.params,
      });
      results.push(...res);
    } catch (e) {
      const c = state?.clusters?.find((x) => x.id === r.clusterId);
      results.push({
        clusterId: r.clusterId,
        alias: c?.alias || r.clusterId,
        host: c?.host || '',
        category: c?.category || '',
        status: 'error',
        message: String(e),
        durationMs: 0,
      });
    }
  }
  renderResults(results);
  const failed = results.filter((r) => r.status !== 'ok').length;
  showToast(failed ? `${successMsg} with ${failed} error(s)` : successMsg, failed ? 'error' : 'success');
  return true;
}

document.querySelectorAll('.tab').forEach((tab) => {
  tab.addEventListener('click', () => {
    document.querySelectorAll('.tab').forEach((t) => t.classList.remove('active'));
    document.querySelectorAll('.panel').forEach((p) => p.classList.remove('active'));
    tab.classList.add('active');
    document.getElementById(`panel-${tab.dataset.tab}`).classList.add('active');
    // Create role / Alter role live in the tabs bar but belong to Operations only.
    document.getElementById('op-tabs')?.classList.toggle('hidden', tab.dataset.tab !== 'operations');
  });
});

document.querySelectorAll('.op-tab').forEach((tab) => {
  tab.addEventListener('click', () => {
    currentOp = tab.dataset.op;
    document.querySelectorAll('.op-tab').forEach((t) => t.classList.remove('active'));
    tab.classList.add('active');
    if (isCreateMode()) {
      // Fresh, empty create form over the current target selection.
      resetEditMaps();
      loadCommentEditor(); // clears the identity inputs
      synthCreateBaseline();
      renderAlterDetail();
      updateOpsFooter();
    } else {
      // Alter doubles as "find role": clear state, show the prompt, open the search popup.
      alterSelected = null;
      alterDetails = [];
      resetEditMaps();
      renderAlterDetail();
      updateOpsFooter();
      openSearchDialog();
    }
  });
});

document.getElementById('btn-add-cluster').addEventListener('click', () => openClusterDialog(null));

document.getElementById('btn-import-env').addEventListener('click', async () => {
  const app = backend();
  try {
    const env = await app.ImportFromEnvironment();
    openClusterDialog({
      alias: env.host ? `Imported ${env.host}` : 'Imported cluster',
      host: env.host,
      port: env.port,
      database: env.database,
      category: 'uat',
      connectUser: env.user,
    });
  } catch (e) {
    showToast(String(e), 'error');
  }
});

document.getElementById('cluster-form').addEventListener('submit', async (ev) => {
  ev.preventDefault();
  const app = backend();
  const form = ev.target;
  const input = clusterInputFromForm(form);
  try {
    if (form.id.value) {
      await app.UpdateCluster(form.id.value, input);
    } else {
      await app.AddCluster(input);
    }
    document.getElementById('cluster-dialog').close();
    await loadConfig();
    showToast('Cluster saved', 'success');
  } catch (e) {
    showToast(String(e), 'error');
  }
});

document.getElementById('cluster-form').addEventListener('click', (ev) => {
  if (ev.target.value === 'cancel') {
    document.getElementById('cluster-dialog').close();
  }
});

let clusterTestTimers = [];
/** Flash the Test-connection button green with "OK", then fade back over ~1s. */
function flashTestOk(btn) {
  clusterTestTimers.forEach(clearTimeout);
  clusterTestTimers = [];
  const original = btn.dataset.label || (btn.dataset.label = btn.textContent);
  btn.style.minWidth = `${btn.getBoundingClientRect().width}px`; // hold width while text is "OK"
  btn.classList.remove('test-fade');
  btn.classList.add('test-ok'); // instant green
  btn.textContent = 'OK';
  clusterTestTimers.push(
    setTimeout(() => {
      btn.textContent = original;
      // Enable the transition while still green, flush styles so that state is committed,
      // then drop green so the colour actually animates (changing both at once would snap).
      btn.classList.add('test-fade');
      void btn.offsetWidth; // force reflow to commit the green + transition state
      btn.classList.remove('test-ok');
      clusterTestTimers.push(setTimeout(() => btn.classList.remove('test-fade'), 1100));
    }, 1000)
  );
}

document.getElementById('btn-test-cluster').addEventListener('click', async () => {
  const app = backend();
  const form = document.getElementById('cluster-form');
  const errEl = document.getElementById('cluster-test-error');
  errEl.classList.add('hidden');
  errEl.textContent = '';

  const input = clusterInputFromForm(form);
  const auth = { user: input.connectUser, password: '' };
  try {
    // Test the on-screen values (works for unsaved/new clusters too).
    await app.TestConnectionInput(input, auth);
    flashTestOk(document.getElementById('btn-test-cluster'));
  } catch (e) {
    errEl.textContent = String(e);
    errEl.classList.remove('hidden');
  }
});

document.getElementById('btn-run').addEventListener('click', runOperation);
document.getElementById('btn-test-clusters').addEventListener('click', testAllClusters);
document.getElementById('btn-save-settings').addEventListener('click', saveSettings);

const COPY_ICON_SVG = '<svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="5" y="1" width="10" height="11" rx="2"/><rect x="1" y="4" width="10" height="11" rx="2"/></svg>';
const CHECK_ICON_SVG = '<svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="2,8 6,13 14,3"/></svg>';
document.getElementById('btn-copy-config-path')?.addEventListener('click', async (ev) => {
  const btn = ev.currentTarget;
  const path = document.getElementById('config-path').textContent || '';
  if (!path) return;
  try {
    await navigator.clipboard.writeText(path);
    btn.innerHTML = CHECK_ICON_SVG;
    showToast('Config path copied', 'success');
    setTimeout(() => { btn.innerHTML = COPY_ICON_SVG; }, 1500);
  } catch (e) {
    showToast('Copy failed: ' + e, 'error');
  }
});

document.getElementById('btn-toggle-clusters')?.addEventListener('click', (ev) => {
  const btn = ev.currentTarget;
  const list = document.getElementById('cluster-checkboxes');
  const expanded = list.classList.toggle('hidden') === false;
  btn.setAttribute('aria-expanded', String(expanded));
  const caret = btn.querySelector('.caret');
  if (caret) caret.textContent = expanded ? '▾' : '▸';
});

// Cluster groups editor: a popup (list) reached from the Clusters toolbar, with a
// second popup for add/edit — mirrors how clusters are managed.
document.getElementById('btn-manage-groups')?.addEventListener('click', () => {
  renderGroupsTable();
  document.getElementById('groups-dialog').showModal();
});
document.getElementById('groups-dialog-close')?.addEventListener('click', () => {
  document.getElementById('groups-dialog').close();
});
document.getElementById('btn-add-group')?.addEventListener('click', () => openGroupDialog(null));

document.getElementById('group-form')?.addEventListener('submit', async (ev) => {
  ev.preventDefault();
  const app = backend();
  const form = ev.target;
  const input = {
    label: form.label.value.trim(),
    color: form.color.value,
    confirm: form.confirm.checked,
  };
  try {
    if (form.id.value) await app.UpdateCategory(form.id.value, input);
    else await app.AddCategory(input);
    document.getElementById('group-dialog').close();
    await loadConfig();
    showToast('Group saved', 'success');
  } catch (e) {
    showToast(String(e), 'error');
  }
});

document.getElementById('group-form')?.addEventListener('click', (ev) => {
  if (ev.target.value === 'cancel') document.getElementById('group-dialog').close();
});

// Alter role tab wiring
function openSearchDialog() {
  const dlg = document.getElementById('search-dialog');
  const scope = document.getElementById('alter-search-scope');
  if (scope) {
    const clusters = resolveSelectedClusters();
    scope.textContent = clusters.length
      ? `Comparing ${clusters.length} selected cluster(s): ${clusters.map((c) => c.alias).join(', ')}`
      : 'No clusters selected — pick categories/clusters in Target selection first.';
  }
  dlg.showModal();
  document.getElementById('alter-search-term')?.focus();
}
document.getElementById('search-dialog-close')?.addEventListener('click', () => {
  document.getElementById('search-dialog').close();
});
document.getElementById('btn-alter-search')?.addEventListener('click', runRoleSearch);
document.getElementById('alter-search-term')?.addEventListener('keydown', (ev) => {
  if (ev.key === 'Enter') {
    ev.preventDefault();
    runRoleSearch();
  } else if (ev.key === 'Escape') {
    // type=search would otherwise just clear the field (swallowing Esc) instead of the
    // native <dialog> Esc-to-close. Force the close regardless of the field's content.
    ev.preventDefault();
    document.getElementById('search-dialog').close();
  }
});

document.getElementById('alter-results')?.addEventListener('click', (ev) => {
  const row = ev.target.closest('.alter-result-row');
  if (row) pickUser(row.dataset.login);
});

document.getElementById('alter-detail')?.addEventListener('click', (ev) => {
  const target = ev.target;
  const chipBtn = target.closest('[data-act]');
  if (chipBtn) {
    const { act, kind, key } = chipBtn.dataset;
    if (act === 'scope') {
      openScopeDialog({ kind, key });
      return;
    }
    if (kind === 'config') {
      const { name, value } = cfgParse(key);
      const cur = clusterIdsWithConfig(name, value);
      if (act === 'revoke') {
        alterConfigSet.delete(key);
        if (cur.size) {
          const reset = new Set(alterConfigReset.get(name) || []);
          for (const cid of cur) reset.add(cid);
          alterConfigReset.set(name, reset);
        }
      } else if (act === 'deladd') {
        alterConfigSet.delete(key);
      } else if (act === 'reset') {
        alterConfigSet.delete(key);
        const reset = alterConfigReset.get(name);
        if (reset) {
          for (const cid of cur) reset.delete(cid);
          if (!reset.size) alterConfigReset.delete(name);
        }
      }
      renderAlterDetail();
      return;
    }
    const isAttr = kind === 'attr';
    const addMap = isAttr ? alterAttrAdd : alterAdd;
    const revMap = isAttr ? alterAttrRemove : alterRevoke;
    const cur = isAttr ? clusterIdsWithAttr(key) : clusterIdsWith(key);
    if (act === 'revoke') {
      // Quick remove everywhere it currently applies.
      addMap.delete(key);
      if (cur.size) revMap.set(key, new Set(cur));
    } else if (act === 'deladd') {
      addMap.delete(key);
    } else if (act === 'reset') {
      addMap.delete(key);
      revMap.delete(key);
    }
    renderAlterDetail();
    return;
  }
  if (target.closest('#btn-alter-add')) {
    if (isCreateMode() && !alterDetails.length) {
      showToast('Select at least one category or cluster first', 'error');
      return;
    }
    openScopeDialog(null);
    return;
  }
  if (target.closest('#btn-alter-add-config')) {
    if (isCreateMode() && !alterDetails.length) {
      showToast('Select at least one category or cluster first', 'error');
      return;
    }
    openScopeDialog({ kind: 'config', isNew: true });
    return;
  }
  if (target.closest('#btn-alter-comments')) {
    openCommentsDialog();
  }
});

// Save / Remove live in the pinned footer (outside #alter-detail).
document.getElementById('btn-alter-save')?.addEventListener('click', saveAlterations);
document.getElementById('btn-alter-remove')?.addEventListener('click', removeRole);

document.getElementById('alter-detail')?.addEventListener('change', (ev) => {
  if (ev.target.id === 'alter-do-pw') {
    alterDoPassword = ev.target.checked;
  }
});

document.getElementById('alter-detail')?.addEventListener('input', (ev) => {
  if (ev.target.id === 'alter-password') {
    alterPassword = ev.target.value;
  }
});

// Comment editor: value/raw edits write straight into commentEditor (never clobbered by
// renderAlterDetail, which does not touch #role-comment-editor).
document.getElementById('role-comment-editor')?.addEventListener('input', (ev) => {
  const t = ev.target;
  if (t.dataset && t.dataset.cfKey) commentEditor.values[t.dataset.cfKey] = t.value;
  else if (t.id === 'rce-raw') commentEditor.raw = t.value;
});
document.getElementById('rce-mode')?.addEventListener('click', (ev) => {
  const btn = ev.target.closest('.seg-btn');
  if (btn) switchCommentMode(btn.dataset.mode);
});

// Scope dialog
document.getElementById('scope-dialog-ok')?.addEventListener('click', confirmScopeDialog);
document.getElementById('scope-dialog-cancel')?.addEventListener('click', () => {
  document.getElementById('scope-dialog').close();
});
document.getElementById('scope-preconfigured')?.addEventListener('click', (ev) => {
  const chip = ev.target.closest('.pick-chip');
  if (chip) chip.classList.toggle('active');
});
document.getElementById('scope-targets')?.addEventListener('change', (ev) => {
  const t = ev.target;
  if (t.classList.contains('scope-group-check')) {
    // Ticking a group toggles all its (enabled) clusters.
    document
      .querySelectorAll(`#scope-targets .scope-cluster-check[data-cat="${CSS.escape(t.dataset.cat)}"]`)
      .forEach((cb) => {
        if (!cb.disabled) cb.checked = t.checked;
      });
  }
});

// Comments dialog — edits stage locally; Cancel discards, OK keeps them (published on
// "Save changes" together with the other edits).
document.getElementById('comments-dialog-cancel')?.addEventListener('click', () => {
  document.getElementById('comments-dialog').close();
});
document.getElementById('comments-dialog-ok')?.addEventListener('click', commitCommentsDialog);
document.getElementById('comments-list')?.addEventListener('click', (ev) => {
  // Fields/Raw toggle for a version.
  const modeBtn = ev.target.closest('[data-cv-mode] .seg-btn');
  if (modeBtn) {
    const idx = Number(modeBtn.closest('[data-cv-mode]').dataset.cvMode);
    const e = commentVersionEditors[idx];
    if (e) {
      switchEditorMode(e, modeBtn.dataset.mode);
      renderCommentsDialog();
    }
  }
});
// Value/raw edits write straight into the per-version editor model.
document.getElementById('comments-list')?.addEventListener('input', (ev) => {
  const t = ev.target;
  const idx = t.dataset.cvIdx != null ? Number(t.dataset.cvIdx) : (t.dataset.cvRaw != null ? Number(t.dataset.cvRaw) : null);
  if (idx == null || !commentVersionEditors[idx]) return;
  if (t.dataset.cfKey) commentVersionEditors[idx].values[t.dataset.cfKey] = t.value;
  else if (t.dataset.cvRaw != null) commentVersionEditors[idx].raw = t.value;
});

document.getElementById('ui-theme')?.addEventListener('click', (ev) => {
  const btn = ev.target.closest('.seg-btn');
  if (!btn) return;
  setThemeButtons(btn.dataset.pref);
  applyTheme(btn.dataset.pref);
});

document.getElementById('comment-view-pref')?.addEventListener('click', (ev) => {
  const btn = ev.target.closest('.seg-btn');
  if (btn) setCommentViewButtons(btn.dataset.pref);
});

document.getElementById('btn-template-help')?.addEventListener('click', () => {
  document.getElementById('template-help-dialog')?.showModal();
});

document.getElementById('template-help-close')?.addEventListener('click', () => {
  document.getElementById('template-help-dialog')?.close();
});

// DB command templates: click a name to edit its type + template in a popup.
document.getElementById('db-functions-editor')?.addEventListener('click', (ev) => {
  const row = ev.target.closest('.fn-row');
  if (row) openFnDialog(row.dataset.fnKey);
});

// Comment fields editor (Settings): edit key/label; reorder/remove/add rows (staged in draft).
document.getElementById('comment-fields-editor')?.addEventListener('input', (ev) => {
  const t = ev.target;
  const idx = Number(t.dataset.idx);
  if (!commentFieldsDraft[idx]) return;
  if (t.classList.contains('cf-key')) commentFieldsDraft[idx].key = t.value;
  else if (t.classList.contains('cf-label')) commentFieldsDraft[idx].label = t.value;
});
wireListEditor('comment-fields-editor', () => commentFieldsDraft, renderCommentFieldsDraft);
document.getElementById('btn-add-comment-field')?.addEventListener('click', () => {
  commentFieldsDraft.push({ key: '', label: '' });
  renderCommentFieldsDraft();
});

// Preconfigured parent groups editor (Settings): edit value; reorder/remove/add rows.
document.getElementById('parent-roles-editor')?.addEventListener('input', (ev) => {
  const t = ev.target;
  if (!t.classList.contains('pr-value')) return;
  const idx = Number(t.dataset.idx);
  if (parentRolesDraft[idx] != null) parentRolesDraft[idx] = t.value;
});
wireListEditor('parent-roles-editor', () => parentRolesDraft, renderParentRolesDraft);
document.getElementById('btn-add-parent-role')?.addEventListener('click', () => {
  parentRolesDraft.push('');
  renderParentRolesDraft();
});
document.getElementById('fn-dialog-done')?.addEventListener('click', () => {
  if (fnDialogKey && dbFnDraft[fnDialogKey]) {
    dbFnDraft[fnDialogKey] = {
      call: document.getElementById('fn-call').value,
      execution: document.getElementById('fn-execution').value,
    };
  }
  document.getElementById('fn-dialog').close();
  renderDBFunctionsRow(fnDialogKey);
  fnDialogKey = null;
});
document.getElementById('fn-dialog-cancel')?.addEventListener('click', () => {
  document.getElementById('fn-dialog').close();
  fnDialogKey = null;
});
// Insert a ${placeholder} at the cursor in the call template.
document.getElementById('fn-placeholder-list')?.addEventListener('click', (ev) => {
  const chip = ev.target.closest('.ph-chip');
  if (!chip) return;
  const ta = document.getElementById('fn-call');
  const token = '${' + chip.dataset.ph + '}';
  const start = ta.selectionStart ?? ta.value.length;
  const end = ta.selectionEnd ?? ta.value.length;
  ta.value = ta.value.slice(0, start) + token + ta.value.slice(end);
  ta.focus();
  const caret = start + token.length;
  ta.setSelectionRange(caret, caret);
});

window.addEventListener('DOMContentLoaded', async () => {
  document.documentElement.setAttribute('data-theme', 'dark');
  configureInputCapitalization();
  await loadConfig();
  // The app opens on Create role; build the comment editor now so it honours the configured
  // preferred view (loadCommentEditor otherwise runs only on op-tab entry / role load).
  loadCommentEditor();
});
