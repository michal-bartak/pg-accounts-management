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

function validateActiveOpForm() {
  const form = document.getElementById(`form-${currentOp}`);
  if (!form) {
    showToast('Internal error: operation form not found', 'error');
    return false;
  }
  if (!form.reportValidity()) {
    return false;
  }
  return true;
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

function applyTheme(themePref) {
  const pref = themePref || 'system';
  let resolved = pref;
  if (pref === 'system') {
    if (!systemThemeMedia) {
      systemThemeMedia = window.matchMedia('(prefers-color-scheme: dark)');
      systemThemeMedia.addEventListener('change', () => {
        const current = document.getElementById('ui-theme')?.value || state?.ui?.theme || 'system';
        if (current === 'system') applyTheme('system');
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
    const themeEl = document.getElementById('ui-theme');
    if (themeEl) {
      themeEl.value = state?.ui?.theme || 'system';
    }
    applyTheme(state?.ui?.theme || 'system');
    renderAll();
  } catch (e) {
    showToast(String(e), 'error');
  }
}

function categoryLabel(id) {
  const c = state?.categories?.find((x) => x.id === id);
  return c?.label || id;
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
    const catClass = c.category === 'production' ? 'production' : c.category === 'uat' ? 'uat' : '';
    tr.innerHTML = `
      <td>${escapeHtml(c.alias)}</td>
      <td>${escapeHtml(c.host)}</td>
      <td>${c.port}</td>
      <td>${escapeHtml(c.database)}</td>
      <td><span class="badge ${catClass}">${escapeHtml(categoryLabel(c.category))}</span></td>
      <td>${escapeHtml(c.sslmode || 'prefer')}</td>
      <td>
        <button class="small" data-action="edit" data-id="${c.id}">Edit</button>
        <button class="small" data-action="test" data-id="${c.id}">Test</button>
        <button class="small danger" data-action="delete" data-id="${c.id}">Delete</button>
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
    const catClass = cat.id === 'production' ? 'production' : cat.id === 'uat' ? 'uat' : '';
    label.innerHTML = `<input type="checkbox" name="category" value="${cat.id}" checked />
      <span class="badge ${catClass}">${escapeHtml(cat.label)}</span>`;
    box.appendChild(label);
    label.querySelector('input')?.addEventListener('change', updateTargetPreview);
  }
}

function renderClusterCheckboxes() {
  const box = document.getElementById('cluster-checkboxes');
  box.innerHTML = '';
  for (const c of state?.clusters || []) {
    const label = document.createElement('label');
    const catClass = c.category === 'production' ? 'production' : c.category === 'uat' ? 'uat' : '';
    if (c.category) label.dataset.category = c.category;
    label.innerHTML = `<input type="checkbox" name="cluster" value="${c.id}" />
      <span class="target-cluster-text">${escapeHtml(c.alias)} <span class="target-cluster-host">(${escapeHtml(c.host)})</span></span>
      <span class="badge ${catClass}">${escapeHtml(categoryLabel(c.category))}</span>`;
    box.appendChild(label);
    label.querySelector('input')?.addEventListener('change', updateTargetPreview);
  }
}

function renderDBFunctionsEditor() {
  const root = document.getElementById('db-functions-editor');
  root.innerHTML = '';
  const fns = state?.dbFunctions;
  if (!fns) return;

  const entries = [
    ['create_role', 'Create role', fns.createRole],
    ['remove_role', 'Remove role', fns.removeRole],
    ['grant_parents', 'Grant parents', fns.grantParents],
    ['revoke_parents', 'Revoke parents', fns.revokeParents],
    ['change_password', 'Change password', fns.changePassword],
    ['set_comment', 'Set comment', fns.setComment],
    ['set_attribute', 'Set attribute', fns.setAttribute],
  ];

  const executionOptions = [
    ['function', 'Function call (SELECT fn($1, …))'],
    ['statement', 'SQL statement (e.g. DROP ROLE ${loginname})'],
    ['block', 'PL/pgSQL block (app wraps DO $dbaccounts$ …)'],
  ];

  for (const [key, title, fn] of entries) {
    const block = document.createElement('div');
    block.className = 'fn-block';
    block.dataset.fnKey = key;
    const call = fn?.call || fn?.Call || '';
    const execution = fn?.execution || fn?.Execution || 'function';
    const execSelect = executionOptions
      .map(
        ([val, label]) =>
          `<option value="${val}"${execution === val ? ' selected' : ''}>${escapeHtml(label)}</option>`
      )
      .join('');
    block.innerHTML = `
      <strong>${escapeHtml(title)}</strong>
      <label>Execution
        <select data-field="execution">${execSelect}</select>
      </label>
      <label>Call template
        <textarea data-field="call" rows="4" class="call-template" placeholder="e.g. your_schema.fn(\${loginname}, …)">${escapeHtml(call)}</textarea>
      </label>`;
    root.appendChild(block);
  }

  document.getElementById('batch-concurrency').value = String(state?.batch?.maxConcurrency || 5);
  const editor = document.getElementById('db-functions-editor');
  if (editor) configureInputCapitalization(editor);
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
    const catClass = r.category === 'production' ? 'production' : r.category === 'uat' ? 'uat' : '';
    tr.innerHTML = `
      <td>${escapeHtml(r.alias)}</td>
      <td>${escapeHtml(r.host)}</td>
      <td><span class="badge ${catClass}">${escapeHtml(categoryLabel(r.category))}</span></td>
      <td class="${r.status === 'ok' ? 'status-ok' : 'status-error'}">${escapeHtml(r.status)}</td>
      <td>${r.durationMs} ms</td>
      <td>${escapeHtml(r.message || '')}</td>`;
    tbody.appendChild(tr);
  }
}

function renderAll() {
  renderClustersTable();
  renderCategoryCheckboxes();
  renderClusterCheckboxes();
  renderDBFunctionsEditor();
  updateTargetPreview();
}

function getSelectedCategories() {
  return [...document.querySelectorAll('#category-checkboxes input:checked')].map((el) => el.value);
}

function getSelectedClusterIDs() {
  return [...document.querySelectorAll('#cluster-checkboxes input:checked')].map((el) => el.value);
}

function getAuth() {
  return {
    user: document.getElementById('auth-user').value.trim(),
    password: document.getElementById('auth-password').value,
  };
}

async function updateTargetPreview() {
  const app = backend();
  const preview = document.getElementById('target-preview');
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
  if (action === 'test') {
    const password = prompt('Password (leave empty if not required, e.g. trust auth):') ?? '';
    try {
      await app.TestConnection({
        clusterId: id,
        auth: { user: document.getElementById('auth-user')?.value?.trim() || '', password },
      });
      showToast('Connection OK', 'success');
    } catch (e) {
      showToast(String(e), 'error');
    }
  }
}

function buildRunRequest() {
  const req = {
    operation: currentOp,
    categoryIds: getSelectedCategories(),
    clusterIds: getSelectedClusterIDs(),
    auth: getAuth(),
    confirmProduction: document.getElementById('confirm-production').checked,
  };

  const form = document.getElementById(`form-${currentOp}`);
  const fd = new FormData(form);

  if (currentOp === 'create_role') {
    req.createRole = {
      loginName: fd.get('loginName')?.toString().trim() || '',
      fullName: fd.get('fullName')?.toString().trim() || '',
      email: fd.get('email')?.toString().trim() || '',
      parentRole: fd.get('parentRole')?.toString().trim() || '',
    };
  }
  return req;
}

function hasProductionTargets() {
  const catIds = getSelectedCategories();
  const clusterIds = new Set(getSelectedClusterIDs());
  if (catIds.includes('production')) return true;
  return state?.clusters?.some((c) => clusterIds.has(c.id) && c.category === 'production');
}

async function runOperation() {
  const app = backend();
  if (!app) {
    showToast('Wails backend not available', 'error');
    return;
  }
  if (currentOp !== 'create_role') {
    return;
  }
  if (!validateActiveOpForm()) {
    return;
  }
  if (getSelectedCategories().length === 0 && getSelectedClusterIDs().length === 0) {
    showToast('Select at least one category or cluster', 'error');
    return;
  }

  if (currentOp === 'remove_role') {
    const ok = await askConfirm(
      'Remove role',
      'Remove this login on all selected clusters? This cannot be undone from the app.'
    );
    if (!ok) {
      return;
    }
  }
  if (hasProductionTargets() && !document.getElementById('confirm-production').checked) {
    showToast('Check "I confirm production execution" to run against production.', 'error');
    return;
  }
  if (hasProductionTargets()) {
    const ok = await askConfirm(
      'Production',
      'This run includes PRODUCTION clusters. Continue?'
    );
    if (!ok) {
      return;
    }
  }

  const req = buildRunRequest();

  try {
    const results = await app.RunOperation(req);
    renderResults(results);
    const failed = results.filter((r) => r.status !== 'ok').length;
    if (failed) {
      showToast(`Completed with ${failed} error(s)`, 'error');
    } else {
      showToast('All clusters succeeded', 'success');
    }
  } catch (e) {
    showToast(String(e), 'error');
  }
}

async function testSelectedConnections() {
  const app = backend();
  let targets;
  try {
    targets = await app.PreviewTargets({
      operation: currentOp,
      categoryIds: getSelectedCategories(),
      clusterIds: getSelectedClusterIDs(),
      auth: getAuth(),
      confirmProduction: true,
    });
  } catch (e) {
    showToast(String(e), 'error');
    return;
  }

  const auth = getAuth();
  const results = [];
  for (const c of targets) {
    const start = Date.now();
    try {
      await app.TestConnection({ clusterId: c.id, auth });
      results.push({
        clusterId: c.id,
        alias: c.alias,
        host: c.host,
        category: c.category,
        status: 'ok',
        message: 'connected',
        durationMs: Date.now() - start,
      });
    } catch (e) {
      results.push({
        clusterId: c.id,
        alias: c.alias,
        host: c.host,
        category: c.category,
        status: 'error',
        message: String(e),
        durationMs: Date.now() - start,
      });
    }
  }
  renderResults(results);
}

function readDBFunctionsFromEditor() {
  const blocks = document.querySelectorAll('#db-functions-editor .fn-block');
  const out = {
    createRole: { name: '', params: [] },
    removeRole: { name: '', params: [] },
    grantParents: { name: '', params: [] },
    revokeParents: { name: '', params: [] },
    changePassword: { name: '', params: [] },
    setComment: { name: '', params: [] },
    setAttribute: { name: '', params: [] },
  };
  const map = {
    create_role: 'createRole',
    remove_role: 'removeRole',
    grant_parents: 'grantParents',
    revoke_parents: 'revokeParents',
    change_password: 'changePassword',
    set_comment: 'setComment',
    set_attribute: 'setAttribute',
  };
  blocks.forEach((block) => {
    const key = block.dataset.fnKey;
    const prop = map[key];
    if (!prop) return;
    const call = block.querySelector('[data-field="call"]')?.value?.trim() || '';
    const execution = block.querySelector('[data-field="execution"]')?.value?.trim() || 'function';
    out[prop] = { call, execution };
  });
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
      theme: document.getElementById('ui-theme')?.value || 'system',
    });
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
// Alter role tab
// ------------------------------------------------------------------

let alterPassword = '';
/** @type {{kind:string, key:string}|null} scope-dialog context; null = adding a new privilege */
let scopeDialogCtx = null;
/** @type {Array<{text:string, ids:string[]}>} distinct comment values for the popup */
let commentVersions = [];

function categoryClass(cat) {
  return cat === 'production' ? 'production' : cat === 'uat' ? 'uat' : '';
}

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
  alterAdd = new Map();
  alterRevoke = new Map();
  alterAttrAdd = new Map();
  alterAttrRemove = new Map();
  alterDoPassword = false;
  alterPassword = '';

  document.getElementById('search-dialog')?.close();
  const detail = document.getElementById('alter-detail');
  detail.classList.remove('hidden');
  detail.innerHTML = '<p class="hint">Loading role details…</p>';

  await reloadDetails();
  document.getElementById('alter-current-hint').textContent = `Editing ${login}.`;
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
  return parts
    .map(
      (p) =>
        `<span class="chip-scope scope-${categoryClass(p.cat) || 'other'} scope-kind-${p.kind} ${extraCls}">${escapeHtml(p.label)}</span>`
    )
    .join('');
}

/** Canonical form of a comment for comparison: sorted-key JSON when valid, else raw. */
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

function renderAlterDetail(errors = []) {
  const root = document.getElementById('alter-detail');
  if (!alterSelected) {
    root.classList.add('hidden');
    root.innerHTML = '';
    return;
  }
  root.classList.remove('hidden');

  if (!alterDetails.length) {
    root.innerHTML =
      `<p class="hint">Role <strong>${escapeHtml(alterSelected)}</strong> was not found on any reachable cluster.</p>` +
      renderDetailErrors(errors);
    return;
  }

  const id = identityConsensus();
  const headFull = !id.fullNameVaries && id.fullName
    ? ` — <span class="alter-fullname">${escapeHtml(id.fullName)}</span>`
    : '';

  const presentLabels = scopeLabelsHtml(describeScope(new Set(alterDetails.map((d) => d.clusterId))));

  let identityRows = '';
  if (id.fullNameVaries) {
    identityRows += `<div class="alter-identity-row">Full name <em>varies across clusters</em></div>`;
  }
  if (id.commentVaries) {
    identityRows += `<div class="alter-identity-row">Comment <em>varies across clusters</em></div>`;
  } else if (id.hasComment) {
    identityRows += `<div class="alter-identity-row">Comment <code>${escapeHtml(id.comment)}</code></div>`;
  } else {
    identityRows += `<div class="alter-identity-row">Comment <em>none</em></div>`;
  }
  const commentsBtn = `<button type="button" class="small" id="btn-alter-comments">${id.commentVaries ? 'Comments differ — view / edit' : 'View / edit comments'}</button>`;

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

  root.innerHTML = `
    <div class="alter-detail-head">
      <h3>Editing <strong>${escapeHtml(alterSelected)}</strong>${headFull}</h3>
    </div>
    ${renderDetailErrors(errors)}
    <div class="alter-present">
      <span class="alter-meta-label">Present on</span>
      <span class="alter-cluster-badges">${presentLabels}</span>
    </div>
    <div class="alter-identity">
      ${identityRows}
      <div class="alter-identity-actions">${commentsBtn}</div>
    </div>

    <div class="alter-section">
      <div class="alter-privs-label">Privileges</div>
      <div class="scope-rows" id="alter-privs">${privHtml}</div>
      <div class="alter-add-priv">
        <button type="button" class="small" id="btn-alter-add">Add privilege…</button>
      </div>
      <p class="hint">Each privilege shows the clusters/groups it is granted on. Use ✎ to add or remove clusters, × to revoke everywhere.</p>
    </div>

    <div class="alter-section">
      <div class="alter-privs-label">Attributes</div>
      <div class="scope-rows" id="alter-attrs">${attrRows}</div>
      <p class="hint">Each attribute shows where it is enabled. Use ✎ to enable/disable per cluster, × to disable everywhere.</p>
    </div>

    <div class="alter-section">
      <div class="alter-privs-label">Password</div>
      <div class="alter-password">
        <input type="password" id="alter-password" autocapitalize="none" autocomplete="off" placeholder="new password" />
        <label class="inline"><input type="checkbox" id="alter-do-pw"${alterDoPassword ? ' checked' : ''} /> Change password</label>
      </div>
    </div>

    <div class="alter-save-bar">
      <button type="button" id="btn-alter-save" class="primary">Save changes</button>
      <button type="button" id="btn-alter-remove" class="danger">Remove role</button>
    </div>`;

  const pwInput = /** @type {HTMLInputElement} */ (document.getElementById('alter-password'));
  if (pwInput) pwInput.value = alterPassword;
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
  const emptyNote = kind === 'attr' ? '<span class="hint">off</span>' : '<span class="hint">none</span>';

  const keptLabels = scopeLabelsHtml(describeScope(kept));
  const addLabels = addSet.size ? scopeLabelsHtml(describeScope(addSet), 'chip-scope-add') : '';
  const revLabels = revSet.size ? scopeLabelsHtml(describeScope(revSet), 'chip-scope-strike') : '';
  const labels = keptLabels + addLabels + revLabels || emptyNote;

  const editBtn = `<button type="button" class="chip-extend" data-kind="${kind}" data-act="scope" data-key="${k}" title="Edit clusters">✎</button>`;
  const verbRemove = kind === 'attr' ? 'Disable everywhere' : 'Revoke everywhere';
  let xBtn = '';
  if (!isNew) {
    xBtn = `<button type="button" class="chip-x" data-kind="${kind}" data-act="revoke" data-key="${k}" title="${verbRemove}">×</button>`;
  } else if (addSet.size) {
    xBtn = `<button type="button" class="chip-x" data-kind="${kind}" data-act="deladd" data-key="${k}" title="Cancel">×</button>`;
  }
  const resetBtn = pending
    ? `<button type="button" class="chip-restore" data-kind="${kind}" data-act="reset" data-key="${k}" title="Discard pending changes">↺</button>`
    : '';

  const fullyRemoved = kept.size === 0 && addSet.size === 0 && revSet.size > 0;
  const stateCls = fullyRemoved ? 'is-removed' : isNew && addSet.size ? 'is-added' : pending ? 'is-extending' : '';
  return `<div class="scope-row ${stateCls}">
    <span class="scope-row-name">${escapeHtml(name)}</span>
    <span class="scope-row-labels">${labels}</span>
    <span class="scope-row-actions">${editBtn}${xBtn}${resetBtn}</span>
  </div>`;
}

// --- Scope dialog (add a new privilege, or extend a privilege/attribute) ---

/** ctx: null → new privilege; {kind:'priv',key} → edit privilege; {kind:'attr',key} → edit attribute. */
function openScopeDialog(ctx) {
  scopeDialogCtx = ctx || null;
  const dlg = document.getElementById('scope-dialog');
  const title = document.getElementById('scope-dialog-title');
  const roleLabel = document.getElementById('scope-role-label');
  const roleInput = /** @type {HTMLInputElement} */ (document.getElementById('scope-role'));
  const ok = document.getElementById('scope-dialog-ok');

  if (!ctx) {
    title.textContent = 'Add privilege';
    roleLabel.classList.remove('hidden');
    roleInput.value = '';
    ok.textContent = 'Add';
  } else if (ctx.kind === 'attr') {
    const a = ROLE_ATTRIBUTES.find((x) => x.key === ctx.key);
    title.textContent = `Edit "${a ? a.label : ctx.key}" clusters`;
    roleLabel.classList.add('hidden');
    ok.textContent = 'Apply';
  } else {
    title.textContent = `Edit "${ctx.key}" clusters`;
    roleLabel.classList.add('hidden');
    ok.textContent = 'Apply';
  }
  buildScopeTargets(ctx);
  dlg.showModal();
  if (!ctx) roleInput.focus();
}

/** Desired-state set currently reflected for a ctx: (current − pendingRevoke) ∪ pendingAdd. */
function scopeDesired(ctx) {
  if (!ctx) return new Set();
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

  const key = ctx ? ctx.key : document.getElementById('scope-role').value.trim();
  if (!ctx && !ROLE_NAME_RE.test(key)) {
    showToast('Invalid role name: use letters, digits, underscore', 'error');
    return;
  }
  const isAttr = !!ctx && ctx.kind === 'attr';
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

// --- Comments dialog (group clusters by comment, edit per group) ---

function buildCommentVersions() {
  // Group by canonical value so JSON comments differing only in formatting collapse
  // into one version. The first raw comment seen for a group is shown/editable.
  const map = new Map();
  for (const d of alterDetails) {
    const key = canonicalComment(d.comment);
    if (!map.has(key)) map.set(key, { text: d.comment || '', ids: [] });
    map.get(key).ids.push(d.clusterId);
  }
  commentVersions = [...map.values()];
}

function openCommentsDialog() {
  buildCommentVersions();
  renderCommentsDialog();
  document.getElementById('comments-dialog').showModal();
}

function renderCommentsDialog() {
  const box = document.getElementById('comments-list');
  box.innerHTML = commentVersions
    .map((v, i) => {
      const labels = scopeLabelsHtml(describeScope(new Set(v.ids)));
      return `<div class="comment-version">
        <div class="comment-scope">${labels || '<span class="hint">no clusters</span>'}</div>
        <textarea class="comment-edit" data-idx="${i}" rows="3" autocapitalize="none" spellcheck="false">${escapeHtml(v.text)}</textarea>
        <div class="comment-actions">
          <button type="button" class="small comment-save" data-idx="${i}">Save to these clusters</button>
        </div>
      </div>`;
    })
    .join('');
}

async function saveCommentVersion(idx) {
  const v = commentVersions[idx];
  if (!v) return;
  const ta = document.querySelector(`.comment-edit[data-idx="${idx}"]`);
  const text = ta ? ta.value : v.text;

  const prod = v.ids.some((cid) => clusterCategory(cid) === 'production');
  if (prod && !document.getElementById('confirm-production').checked) {
    showToast('Check "I confirm production execution" to write to production.', 'error');
    return;
  }
  if (prod) {
    const ok = await askConfirm('Production', 'This comment update includes PRODUCTION clusters. Continue?');
    if (!ok) return;
  }

  const app = backend();
  try {
    const res = await app.RunOperation({
      operation: 'set_comment',
      categoryIds: [],
      clusterIds: v.ids,
      auth: getAuth(),
      confirmProduction: true,
      setComment: { loginName: alterSelected, comment: text },
    });
    renderResults(res);
    const failed = res.filter((r) => r.status !== 'ok').length;
    if (failed) showToast(`Comment saved with ${failed} error(s)`, 'error');
    else showToast('Comment updated', 'success');
  } catch (e) {
    showToast(String(e), 'error');
    return;
  }
  await reloadDetails();
  buildCommentVersions();
  renderCommentsDialog();
}

function buildAlterRequests() {
  /** @type {Array<{op:string, clusterId:string, params:object}>} */
  const requests = [];

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
  }
  return requests;
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

  const ok = await executeAlterRequests(requests, 'Changes saved');
  if (!ok) return;
  // Applied — clear pending edits and refresh from the DB.
  alterAdd = new Map();
  alterRevoke = new Map();
  alterAttrAdd = new Map();
  alterAttrRemove = new Map();
  alterDoPassword = false;
  alterPassword = '';
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
  const prodInvolved = targetIds.some((id) => clusterCategory(id) === 'production');
  if (prodInvolved && !document.getElementById('confirm-production').checked) {
    showToast('Check "I confirm production execution" to write to production.', 'error');
    return false;
  }
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
    document.querySelectorAll('.op-form').forEach((f) => f.classList.remove('active'));
    tab.classList.add('active');
    document.getElementById(`form-${currentOp}`).classList.add('active');
    // The create-role run bar only applies to the Create role tab.
    document.getElementById('create-run-bar')?.classList.toggle('hidden', currentOp !== 'create_role');
    if (currentOp === 'create_role') {
      updateTargetPreview();
    } else if (currentOp === 'alter_user') {
      // Alter role doubles as "find user": open the search popup.
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

document.getElementById('btn-test-cluster').addEventListener('click', async () => {
  const app = backend();
  const form = document.getElementById('cluster-form');
  const password = prompt('Password (leave empty if not required, e.g. trust auth):') ?? '';
  const auth = {
    user: form.connectUser.value.trim() || document.getElementById('auth-user')?.value?.trim() || '',
    password,
  };

  if (form.id.value) {
    try {
      await app.TestConnection({ clusterId: form.id.value, auth });
      showToast('Connection OK', 'success');
    } catch (e) {
      showToast(String(e), 'error');
    }
    return;
  }

  showToast('Save the cluster first, then test connection.', 'error');
});

document.getElementById('btn-run').addEventListener('click', runOperation);
document.getElementById('btn-test-selected').addEventListener('click', testSelectedConnections);
document.getElementById('btn-save-settings').addEventListener('click', saveSettings);
document.getElementById('confirm-production').addEventListener('change', updateTargetPreview);

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
    openScopeDialog(null);
    return;
  }
  if (target.closest('#btn-alter-comments')) {
    openCommentsDialog();
    return;
  }
  if (target.id === 'btn-alter-save') {
    saveAlterations();
    return;
  }
  if (target.id === 'btn-alter-remove') {
    removeRole();
  }
});

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

// Scope dialog
document.getElementById('scope-dialog-ok')?.addEventListener('click', confirmScopeDialog);
document.getElementById('scope-dialog-cancel')?.addEventListener('click', () => {
  document.getElementById('scope-dialog').close();
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

// Comments dialog
document.getElementById('comments-dialog-close')?.addEventListener('click', () => {
  document.getElementById('comments-dialog').close();
});
document.getElementById('comments-list')?.addEventListener('click', (ev) => {
  const btn = ev.target.closest('.comment-save');
  if (btn) saveCommentVersion(Number(btn.dataset.idx));
});

document.getElementById('ui-theme')?.addEventListener('change', (ev) => {
  applyTheme(ev.target.value);
});

document.getElementById('btn-template-help')?.addEventListener('click', () => {
  document.getElementById('template-help-dialog')?.showModal();
});

document.getElementById('template-help-close')?.addEventListener('click', () => {
  document.getElementById('template-help-dialog')?.close();
});

window.addEventListener('DOMContentLoaded', () => {
  document.documentElement.setAttribute('data-theme', 'dark');
  configureInputCapitalization();
  loadConfig();
});
