// @ts-check

const OP_KEYS = ['create_role', 'remove_role', 'grant_parents', 'revoke_parents', 'change_password'];
const FN_KEYS = {
  create_role: 'createRole',
  remove_role: 'removeRole',
  grant_parents: 'grantParents',
  revoke_parents: 'revokeParents',
  change_password: 'changePassword',
};

/** @type {import('../internal/model/model').Config | null} */
let state = null;
let currentOp = 'create_role';
let lastResults = [];
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
  const tbody = document.querySelector('#results-table tbody');
  tbody.innerHTML = '';
  if (!rows?.length) {
    tbody.innerHTML = '<tr><td colspan="6" class="hint">No runs yet.</td></tr>';
    return;
  }
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
  } else if (currentOp === 'remove_role') {
    req.removeRole = { loginName: fd.get('loginName')?.toString().trim() || '' };
  } else if (currentOp === 'grant_parents') {
    req.grantParents = {
      loginName: fd.get('loginName')?.toString().trim() || '',
      parentRoles: fd.get('parentRoles')?.toString().trim() || '',
    };
  } else if (currentOp === 'revoke_parents') {
    req.revokeParents = {
      loginName: fd.get('loginName')?.toString().trim() || '',
      parentRoles: fd.get('parentRoles')?.toString().trim() || '',
    };
  } else if (currentOp === 'change_password') {
    req.changePassword = {
      loginName: fd.get('loginName')?.toString().trim() || '',
      newPassword: fd.get('newPassword')?.toString() || '',
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
  };
  const map = {
    create_role: 'createRole',
    remove_role: 'removeRole',
    grant_parents: 'grantParents',
  revoke_parents: 'revokeParents',
    change_password: 'changePassword',
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

document.querySelectorAll('.tab').forEach((tab) => {
  tab.addEventListener('click', () => {
    document.querySelectorAll('.tab').forEach((t) => t.classList.remove('active'));
    document.querySelectorAll('.panel').forEach((p) => p.classList.remove('active'));
    tab.classList.add('active');
    document.getElementById(`panel-${tab.dataset.tab}`).classList.add('active');
  });
});

document.querySelectorAll('.op-tab').forEach((tab) => {
  tab.addEventListener('click', () => {
    currentOp = tab.dataset.op;
    document.querySelectorAll('.op-tab').forEach((t) => t.classList.remove('active'));
    document.querySelectorAll('.op-form').forEach((f) => f.classList.remove('active'));
    tab.classList.add('active');
    document.getElementById(`form-${currentOp}`).classList.add('active');
    updateTargetPreview();
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
