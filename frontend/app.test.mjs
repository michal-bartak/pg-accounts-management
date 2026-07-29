// Unit tests for the comment / consensus / presence logic in app.js.
//
// The frontend has no build step and no package.json (see CLAUDE.md), so these tests use only the
// Node built-ins (`node --test`, `node:vm`) — no framework, no dependencies. app.js is a classic
// browser script full of top-level DOM wiring; we load it into a vm context behind a permissive
// DOM stub (the DOMContentLoaded init never fires, so loading only defines the globals), then drive
// the real functions. Snippets run in the SAME context so they can read/write app.js's top-level
// `let` state (alterDetails, commentEditor, …) by bare name, exactly like the browser console.
//
// Run:  node --test frontend/app.test.mjs

import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import vm from 'node:vm';

const here = dirname(fileURLToPath(import.meta.url));
const source = readFileSync(join(here, 'app.js'), 'utf8');

// --- Minimal permissive DOM/BOM stub (enough for app.js to load without throwing) -------------
function makeEl() {
  return new Proxy(function () {}, {
    get(_t, prop) {
      switch (prop) {
        case 'classList':
          return { add() {}, remove() {}, toggle() {}, contains() { return false; } };
        case 'style': return {};
        case 'dataset': return {};
        case 'value': return '';
        case 'checked': case 'readOnly': case 'disabled': case 'hidden': return false;
        case 'innerHTML': case 'textContent': case 'title': case 'className':
        case 'placeholder': case 'id': case 'type': return '';
        case 'selectionStart': case 'selectionEnd': return 0;
        case 'closest': case 'querySelector': return () => null;
        case 'querySelectorAll': return () => [];
        case 'getBoundingClientRect':
          return () => ({ left: 0, top: 0, width: 0, height: 0, right: 0, bottom: 0 });
        case 'children': case 'childNodes': return [];
        case 'parentNode': case 'parentElement': case 'nextSibling': return null;
        case 'getAttribute': return () => null;
        case 'addEventListener': case 'removeEventListener': case 'setAttribute':
        case 'removeAttribute': case 'appendChild': case 'append': case 'prepend':
        case 'insertBefore': case 'remove': case 'focus': case 'blur': case 'click':
        case 'showModal': case 'close': case 'setSelectionRange': case 'dispatchEvent':
        case 'scrollIntoView': case 'cloneNode':
          return () => {};
        default:
          return makeEl();
      }
    },
    set() { return true; },
    apply() { return makeEl(); },
  });
}

const documentStub = {
  getElementById: () => makeEl(),
  querySelector: () => makeEl(),
  querySelectorAll: () => [],
  createElement: () => makeEl(),
  addEventListener: () => {},
  documentElement: makeEl(),
  body: makeEl(),
  head: makeEl(),
};

const sandbox = {};
sandbox.window = sandbox; // window === global, matches the browser
sandbox.globalThis = sandbox;
sandbox.self = sandbox;
sandbox.document = documentStub;
sandbox.navigator = { clipboard: { writeText: async () => {} } };
sandbox.CSS = { escape: (s) => String(s) };
sandbox.matchMedia = () => ({ matches: false, addEventListener() {}, removeEventListener() {} });
sandbox.addEventListener = () => {}; // window.addEventListener('DOMContentLoaded', …) at load
sandbox.removeEventListener = () => {};
sandbox.console = console;
sandbox.setTimeout = () => 0;
sandbox.clearTimeout = () => {};

const ctx = vm.createContext(sandbox);
vm.runInContext(source, ctx, { filename: 'app.js' });

// Evaluate an expression string in the app.js context and bring the (JSON-serialised) result back
// into this realm — avoids cross-realm prototype mismatches in assert.deepEqual.
function evalJSON(expr) {
  return JSON.parse(vm.runInContext(`JSON.stringify(${expr})`, ctx));
}

// Shared config: two configured comment fields, Fields as the default empty view.
const SETUP_STATE =
  `state = { ui:{commentDefaultView:'fields'}, commentFields:[{key:'full_name',label:'Full name'},{key:'e_mail',label:'Email'}] };`;

// ---------------------------------------------------------------------------------------------
test('canonicalComment: JSON is key-order / whitespace insensitive; non-JSON is verbatim', () => {
  const r = evalJSON(`(() => ({
    sameObject: canonicalComment('{"a":1,"b":2}') === canonicalComment('{ "b":2, "a":1 }'),
    differ: canonicalComment('{"a":1}') !== canonicalComment('{"a":2}'),
    plain: canonicalComment('hello') === 'hello',
  }))()`);
  assert.equal(r.sameObject, true);
  assert.equal(r.differ, true);
  assert.equal(r.plain, true);
});

// ---------------------------------------------------------------------------------------------
test('editorFromComment: null is an editable empty string (no readonly / no ⚠)', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    const e = editorFromComment('{"full_name":null}', false);
    return { isReadonly: e.readonly.has('full_name'), value: e.values['full_name'], mode: e.mode };
  })()`);
  assert.equal(r.isReadonly, false);
  assert.equal(r.value, '');
  assert.equal(r.mode, 'fields');
});

test('editorFromComment: non-string (number) stays read-only and shown as JSON', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    const e = editorFromComment('{"count":5}', false);
    return { isReadonly: e.readonly.has('count'), value: e.values['count'] };
  })()`);
  assert.equal(r.isReadonly, true);
  assert.equal(r.value, '5');
});

test('editorFromComment: mode is Raw for plain text, configured pref for empty', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    const plain = editorFromComment('just text', false).mode;
    const empty = editorFromComment('', false).mode;
    state.ui.commentDefaultView = 'raw';
    const emptyRaw = editorFromComment('', false).mode;
    return { plain, empty, emptyRaw };
  })()`);
  assert.equal(r.plain, 'raw');
  assert.equal(r.empty, 'fields');
  assert.equal(r.emptyRaw, 'raw');
});

// ---------------------------------------------------------------------------------------------
test('assembleCommentFrom: empty & all-blank comment serialises to ""', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    const e = editorFromComment('', false); // both configured fields blank, none in baseObj
    return assembleCommentFrom(e);
  })()`);
  assert.equal(r, '');
});

test('assembleCommentFrom: a loaded null round-trips as null', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    const e = editorFromComment('{"full_name":null}', false);
    return canonicalComment(assembleCommentFrom(e)) === canonicalComment('{"full_name":null}');
  })()`);
  assert.equal(r, true);
});

test('assembleCommentFrom: clearing an existing string field stores null; a never-present field is dropped', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    const e = editorFromComment('{"full_name":"A","e_mail":"x"}', false);
    e.values['full_name'] = '';                 // clear a field that was in the comment
    return canonicalComment(assembleCommentFrom(e)) === canonicalComment('{"full_name":null,"e_mail":"x"}');
  })()`);
  assert.equal(r, true);
});

test('assembleCommentFrom: read-only non-string value is preserved untouched', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    const e = editorFromComment('{"count":5,"full_name":"A"}', false);
    return canonicalComment(assembleCommentFrom(e)) === canonicalComment('{"count":5,"full_name":"A"}');
  })()`);
  assert.equal(r, true);
});

// ---------------------------------------------------------------------------------------------
// commentConsensus.varies drives the "Comments differ" banner / inline-vs-dialog mode.
function consensus(rows) {
  return evalJSON(`(() => {
    ${SETUP_STATE}
    alterDetails = ${JSON.stringify(rows)};
    return commentConsensus();
  })()`);
}
const existing = (comment) => ({ clusterId: 'c' + Math.random().toString(36).slice(2), exists: true, comment });

test('commentConsensus: identical comments do not vary', () => {
  assert.equal(consensus([existing('{"full_name":"A"}'), existing('{"full_name":"A"}')]).varies, false);
});

test('commentConsensus: an UNSET comment on one cluster (others equal) counts as a divergence', () => {
  assert.equal(consensus([existing('{"full_name":"A"}'), existing('{"full_name":"A"}'), existing('')]).varies, true);
});

test('commentConsensus: all-unset does not vary', () => {
  assert.equal(consensus([existing(''), existing('')]).varies, false);
});

test('commentConsensus: a pending presence-add (exists:false) never triggers a false divergence', () => {
  const rows = [existing('{"full_name":"A"}'), existing('{"full_name":"A"}'),
    { clusterId: 'cX', exists: false, comment: '' }];
  assert.equal(consensus(rows).varies, false);
});

// ---------------------------------------------------------------------------------------------
// buildAlterClusterOps: presence (create/remove) + comment publishing.
function buildOps(setup) {
  return evalJSON(`(() => {
    ${SETUP_STATE}
    currentOp = 'alter_user';
    alterSelected = 'bob';
    resetEditMaps();
    ${setup}
    return buildAlterClusterOps().map(c => ({ clusterId: c.clusterId, ops: c.operations.map(o => o.operation),
      comment: (c.operations.find(o => o.operation === 'set_comment') || {}).setComment &&
               c.operations.find(o => o.operation === 'set_comment').setComment.comment }));
  })()`);
}

test('buildAlterClusterOps: unchanged consistent-comment role is not dirty (idempotent load)', () => {
  const ops = buildOps(`
    alterDetails = [
      { clusterId:'c1', exists:true, comment:'{"full_name":"A"}', parents:[], attributes:{}, settings:{} },
      { clusterId:'c2', exists:true, comment:'{"full_name":"A"}', parents:[], attributes:{}, settings:{} },
    ];
    commentEditor = editorFromComment('{"full_name":"A"}', false);`);
  assert.deepEqual(ops, []);
});

test('buildAlterClusterOps: presence-add with a consistent comment → create_role + set_comment(consistent)', () => {
  const ops = buildOps(`
    alterScopeClusters = [{clusterId:'c1',alias:'c1',category:'p'},{clusterId:'cX',alias:'cX',category:'p'}];
    alterDetails = [
      { clusterId:'c1', exists:true, comment:'{"full_name":"A"}', parents:[], attributes:{}, settings:{} },
      { clusterId:'cX', exists:false, comment:'', parents:[], attributes:{}, settings:{} },
    ];
    commentEditor = editorFromComment('{"full_name":"A"}', false);`);
  const cX = ops.find((o) => o.clusterId === 'cX');
  assert.deepEqual(cX.ops, ['create_role', 'set_comment']);
  assert.equal(evalJSON(`canonicalComment(${JSON.stringify(cX.comment)})`), evalJSON(`canonicalComment('{"full_name":"A"}')`));
  assert.equal(ops.find((o) => o.clusterId === 'c1'), undefined); // existing cluster unchanged
});

test('buildAlterClusterOps: presence-add while comments vary → create_role only (created bare)', () => {
  const ops = buildOps(`
    alterScopeClusters = [{clusterId:'cX',alias:'cX',category:'p'}];
    alterDetails = [{ clusterId:'cX', exists:false, comment:'', parents:[], attributes:{}, settings:{} }];
    commentEditor = editorFromComment('', true); /* varies */`);
  assert.deepEqual(ops.find((o) => o.clusterId === 'cX').ops, ['create_role']);
});

test('buildAlterClusterOps: a cluster flagged for removal → remove_role only', () => {
  const ops = buildOps(`
    alterDetails = [
      { clusterId:'c1', exists:true, comment:'{"full_name":"A"}', parents:['gr_x'], attributes:{}, settings:{} },
    ];
    commentEditor = editorFromComment('{"full_name":"A"}', false);
    roleRemoveClusters = new Set(['c1']);`);
  assert.deepEqual(ops, [{ clusterId: 'c1', ops: ['remove_role'] }]);
});

test('buildAlterClusterOps: a pending grant emits grant_parents', () => {
  const ops = buildOps(`
    alterDetails = [{ clusterId:'c1', exists:true, comment:'{"full_name":"A"}', parents:[], attributes:{}, settings:{} }];
    commentEditor = editorFromComment('{"full_name":"A"}', false);
    alterAdd.set('gr_new', new Set(['c1']));`);
  assert.deepEqual(ops, [{ clusterId: 'c1', ops: ['grant_parents'] }]);
});

// Regression: the parents selected for a newly-created role must reach BOTH the create_role
// template's ${parent_roles} AND the follow-up grant_parents (per cluster).
test('buildCreateClusterOps: create_role and grant_parents both carry the selected parents', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    const _origResolve = resolveSelectedClusters; // shared vm context — restore to avoid leaking
    try {
      state.clusters = [{id:'c1',alias:'c1',category:'p'},{id:'c2',alias:'c2',category:'p'}];
      currentOp = 'create_role';
      resetEditMaps();
      resolveSelectedClusters = () => state.clusters;   // stub the DOM-checkbox read
      alterDetails = state.clusters.map((c) => ({ clusterId:c.id, exists:false, comment:'', parents:[], attributes:{}, settings:{} }));
      alterSelected = 'bob';
      alterAdd.set('gr_admin', new Set(['c1','c2']));
      alterAdd.set('gr_ro', new Set(['c1']));           // c1 gets two parents, c2 gets one
      return buildCreateClusterOps({ loginName:'bob' }).map((c) => ({
        id: c.clusterId,
        create: c.operations.find((o) => o.operation === 'create_role').createRole.parentRoles,
        grant: (c.operations.find((o) => o.operation === 'grant_parents') || {}).grantParents.parentRoles,
      }));
    } finally { resolveSelectedClusters = _origResolve; }
  })()`);
  assert.deepEqual(r, [
    { id: 'c1', create: 'gr_admin,gr_ro', grant: 'gr_admin,gr_ro' },
    { id: 'c2', create: 'gr_admin', grant: 'gr_admin' },
  ]);
});

// Alter presence-add: create_role for a newly-added cluster also carries that cluster's parents.
test('buildAlterClusterOps: presence-add create_role carries the granted parents', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    currentOp = 'alter_user';
    alterSelected = 'bob';
    resetEditMaps();
    alterDetails = [{ clusterId:'cX', exists:false, comment:'', parents:[], attributes:{}, settings:{} }];
    commentEditor = editorFromComment('', true); /* varies → created bare, no comment */
    alterAdd.set('gr_new', new Set(['cX']));
    const ops = buildAlterClusterOps().find((c) => c.clusterId === 'cX').operations;
    return {
      order: ops.map((o) => o.operation),
      create: ops.find((o) => o.operation === 'create_role').createRole.parentRoles,
      grant: ops.find((o) => o.operation === 'grant_parents').grantParents.parentRoles,
    };
  })()`);
  assert.deepEqual(r.order, ['create_role', 'grant_parents']);
  assert.equal(r.create, 'gr_new');
  assert.equal(r.grant, 'gr_new');
});

// addPrivilegeScope: "Add privilege" is additive — it must never revoke where the privilege
// already lives. Regression for the bug where adding P1 to cluster D revoked it from A/B/C.
test('addPrivilegeScope: adding an existing privilege to a new cluster grants D and revokes nothing', () => {
  const r = evalJSON(`(() => {
    alterDetails = [
      { clusterId:'A', exists:true, parents:['P1'], attributes:{}, settings:{}, comment:'' },
      { clusterId:'B', exists:true, parents:['P1'], attributes:{}, settings:{}, comment:'' },
      { clusterId:'C', exists:true, parents:['P1'], attributes:{}, settings:{}, comment:'' },
      { clusterId:'D', exists:true, parents:[],     attributes:{}, settings:{}, comment:'' },
    ];
    alterAdd = new Map(); alterRevoke = new Map();
    addPrivilegeScope(['P1'], new Set(['D']));
    return { add:[...(alterAdd.get('P1')||[])].sort(), rev:[...(alterRevoke.get('P1')||[])].sort() };
  })()`);
  assert.deepEqual(r.add, ['D']); // only D newly granted
  assert.deepEqual(r.rev, []);    // A/B/C stay granted — nothing revoked
});

// Adding a cluster that had a pending revoke cancels that revoke (grant wins), still no over-revoke.
test('addPrivilegeScope: granting a cluster clears its pending revoke and merges with prior adds', () => {
  const r = evalJSON(`(() => {
    alterDetails = [
      { clusterId:'A', exists:true, parents:['P1'], attributes:{}, settings:{}, comment:'' },
      { clusterId:'B', exists:true, parents:[],     attributes:{}, settings:{}, comment:'' },
      { clusterId:'C', exists:true, parents:[],     attributes:{}, settings:{}, comment:'' },
    ];
    alterAdd = new Map([['P1', new Set(['B'])]]);
    alterRevoke = new Map([['P1', new Set(['A'])]]);
    addPrivilegeScope(['P1'], new Set(['A','C']));
    return { add:[...(alterAdd.get('P1')||[])].sort(), rev:[...(alterRevoke.get('P1')||[])].sort() };
  })()`);
  assert.deepEqual(r.add, ['B', 'C']); // prior add B kept; C newly granted; A already present so not re-added
  assert.deepEqual(r.rev, []);         // pending revoke of A cancelled by granting A
});

// Shared primitives used by all three sections (privileges, attributes, settings).
test('scopeMergeAdd: additive — extends add over desired, cancels revoke there, never touches others', () => {
  const r = evalJSON(`(() => {
    const add = new Set(['x']), rev = new Set(['a','z']);
    scopeMergeAdd(add, rev, new Set(['a','b']) /*cur*/, new Set(['a','c']) /*desired*/);
    return { add:[...add].sort(), rev:[...rev].sort() };
  })()`);
  assert.deepEqual(r.add, ['c', 'x']); // c newly added; a already in cur so not added; prior x kept
  assert.deepEqual(r.rev, ['z']);      // a removed from rev (granted); z (outside desired) untouched
});

test('scopeDiff: full desired-vs-current diff → grant (desired−cur), revoke (cur−desired)', () => {
  const r = evalJSON(`(() => {
    const d = scopeDiff(new Set(['a','b','c']) /*cur*/, new Set(['b','d']) /*desired*/);
    return { add:[...d.add].sort(), rev:[...d.rev].sort() };
  })()`);
  assert.deepEqual(r.add, ['d']);
  assert.deepEqual(r.rev, ['a', 'c']);
});

// Same additive bug/fix as privileges, now shared: "Add setting" must not RESET clusters that
// already carry the value elsewhere.
test('applyConfigScope: adding an existing setting to a new cluster SETs D and RESETs nothing', () => {
  const r = evalJSON(`(() => {
    alterDetails = [
      { clusterId:'A', exists:true, parents:[], attributes:{}, settings:{work_mem:'64MB'}, comment:'' },
      { clusterId:'B', exists:true, parents:[], attributes:{}, settings:{work_mem:'64MB'}, comment:'' },
      { clusterId:'C', exists:true, parents:[], attributes:{}, settings:{work_mem:'64MB'}, comment:'' },
      { clusterId:'D', exists:true, parents:[], attributes:{}, settings:{}, comment:'' },
    ];
    alterConfigSet = new Map(); alterConfigReset = new Map();
    applyConfigScope('work_mem', '64MB', null, true /*isNew*/, new Set(['D']));
    return { set:[...(alterConfigSet.get('work_mem=64MB')||[])].sort(),
             reset:[...(alterConfigReset.get('work_mem')||[])].sort() };
  })()`);
  assert.deepEqual(r.set, ['D']); // only D newly set
  assert.deepEqual(r.reset, []);  // A/B/C keep the setting — nothing reset
});

// Editing a setting's value still RESETs the clusters that leave the old value.
test('applyConfigScope: editing a value SETs desired and RESETs clusters leaving the old value', () => {
  const r = evalJSON(`(() => {
    alterDetails = [
      { clusterId:'A', exists:true, parents:[], attributes:{}, settings:{work_mem:'64MB'}, comment:'' },
      { clusterId:'B', exists:true, parents:[], attributes:{}, settings:{work_mem:'64MB'}, comment:'' },
    ];
    alterConfigSet = new Map(); alterConfigReset = new Map();
    applyConfigScope('work_mem', '128MB', '64MB', false /*edit*/, new Set(['A']));
    return { set:[...(alterConfigSet.get('work_mem=128MB')||[])].sort(),
             reset:[...(alterConfigReset.get('work_mem')||[])].sort() };
  })()`);
  assert.deepEqual(r.set, ['A']);   // A moved to the new value
  assert.deepEqual(r.reset, ['B']); // B left 64MB and wasn't re-selected → reset
});

// ---------------------------------------------------------------------------------------------
test('clusterHasStagedEdits: true for any staged map / pending-create row, false when clean', () => {
  const r = evalJSON(`(() => {
    resetEditMaps();
    alterDetails = [
      { clusterId:'grant', exists:true }, { clusterId:'attr', exists:true },
      { clusterId:'cfg', exists:true }, { clusterId:'drop', exists:true },
      { clusterId:'cmt', exists:true }, { clusterId:'create', exists:false },
      { clusterId:'clean', exists:true },
    ];
    alterAdd.set('parent', new Set(['grant']));
    alterAttrRemove.set('LOGIN', new Set(['attr']));
    alterConfigSet.set('work_mem=128MB', new Set(['cfg']));
    roleRemoveClusters.add('drop');
    commentOverrides.set('cmt', 'hello');
    return {
      grant: clusterHasStagedEdits('grant'),
      attr: clusterHasStagedEdits('attr'),
      cfg: clusterHasStagedEdits('cfg'),
      drop: clusterHasStagedEdits('drop'),
      cmt: clusterHasStagedEdits('cmt'),
      create: clusterHasStagedEdits('create'),
      clean: clusterHasStagedEdits('clean'),
    };
  })()`);
  assert.deepEqual(r, { grant:true, attr:true, cfg:true, drop:true, cmt:true, create:true, clean:false });
});

// ---------------------------------------------------------------------------------------------
test('openSearchDialog clears cached results (no stale matches from a prior scope)', () => {
  const r = evalJSON(`(() => {
    alterGroups = [{ login:'stale', details:[] }];
    openSearchDialog();          // must not throw; must drop cached results
    return { cleared: alterGroups.length };
  })()`);
  assert.equal(r.cleared, 0);
});

// --- commentFieldArgs: per-cluster create_role / set_comment placeholder values ---------------
test('commentFieldArgs: JSON-encodes each configured field present in the comment (typed)', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    state.commentFields = [{key:'full_name'},{key:'e_mail'},{key:'age'},{key:'active'}];
    return commentFieldArgs('{"full_name":"John","age":42,"active":true,"e_mail":null}');
  })()`);
  // Values are JSON encodings so the backend can recover their type; null encodes as "null".
  assert.deepEqual(r, { full_name: '"John"', e_mail: 'null', age: '42', active: 'true' });
});

test('commentFieldArgs: an empty-string field is kept, encoded as "" (backend maps it to NULL)', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    state.commentFields = [{key:'full_name'}];
    return commentFieldArgs('{"full_name":""}');
  })()`);
  // Present (not omitted) and JSON-encoded as the two-char string "" — renderCommentFieldSQL /
  // commentFieldBindValue then turn "" into SQL NULL. Distinct from an absent key (also NULL).
  assert.deepEqual(r, { full_name: '""' });
});

test('commentFieldArgs: keys absent from the comment are omitted (→ backend NULL)', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    return commentFieldArgs('{"full_name":"Jane"}');
  })()`);
  assert.deepEqual(r, { full_name: '"Jane"' }); // e_mail omitted
});

test('commentFieldArgs: a plain-text (non-object) comment yields no field args', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    return commentFieldArgs('just some text');
  })()`);
  assert.deepEqual(r, {});
});

test('fnPlaceholderNames: create_role / set_comment inject configured fields; other ops unchanged', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    return {
      create: fnPlaceholderNames('create_role', ['loginname','parent_roles']),
      comment: fnPlaceholderNames('set_comment', ['loginname','comment']),
      other: fnPlaceholderNames('change_password', ['loginname','new_password']),
    };
  })()`);
  assert.deepEqual(r.create, ['loginname', 'full_name', 'e_mail', 'parent_roles']);
  assert.deepEqual(r.comment, ['loginname', 'full_name', 'e_mail', 'comment']);
  assert.deepEqual(r.other, ['loginname', 'new_password']);
});

// --- Run-status phases: create+load log concatenation and phase-aware chip summary -------------
test('run-status: post-create load logs a -- Load Role query on EVERY cluster + inline error', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    state.clusters = [{id:'c1',alias:'c1',category:'p'},{id:'c2',alias:'c2',category:'p'}];
    runState = null;
    beginRunStatus([{clusterId:'c1'},{clusterId:'c2'}], 'Create');
    finishRunStatus([
      {clusterId:'c1', status:'ok', message:'', durationMs:5, queries:['CREATE ROLE "bob"','GRANT "gr_a" TO "bob"']},
      {clusterId:'c2', status:'error', message:'permission denied', durationMs:3, queries:['CREATE ROLE "bob"']},
    ]);
    const createSummary = runStatusSummary();
    // Load runs on ALL selected clusters: c1 found, c2 reachable-but-role-absent (carries the SQL).
    reportRoleLoad({
      valid:  [{clusterId:'c1', alias:'c1', category:'p', durationMs:2, exists:true,  queries:['SELECT rolname']}],
      errors: [],
      all:    [
        {clusterId:'c1', alias:'c1', category:'p', durationMs:2, exists:true,  error:'', queries:['SELECT rolname']},
        {clusterId:'c2', alias:'c2', category:'p', durationMs:1, exists:false, error:'', queries:['SELECT rolname']},
      ],
    }, { appendLog:true });
    return {
      createSummary,
      afterSummary: runStatusSummary(),
      c1: rowQueries(runState.byId.get('c1')),
      c2: rowQueries(runState.byId.get('c2')),
    };
  })()`);
  assert.deepEqual(r.createSummary, { stateClass:'error', text:'Status: Create (1/2 failed)' });
  // Load ran (ok) on both clusters, so the Load phase reports 2 clusters.
  assert.equal(r.afterSummary.text, 'Status: Create (1/2 failed), Load OK (2 clusters)');
  assert.deepEqual(r.c1, ['-- Create Role','CREATE ROLE "bob"','GRANT "gr_a" TO "bob"','-- Load Role','SELECT rolname']);
  // The failed cluster now shows its create error inline AND the load query that ran there.
  assert.deepEqual(r.c2, ['-- Create Role','CREATE ROLE "bob"','-- ERROR: permission denied','-- Load Role','SELECT rolname']);
});

test('run-status: an unreachable cluster on load shows -- Load Role + -- ERROR (no SQL ran)', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    state.clusters = [{id:'c1',alias:'c1',category:'p'}];
    runState = null;
    beginRunStatus([{clusterId:'c1'}], 'Create');
    finishRunStatus([{clusterId:'c1', status:'ok', durationMs:1, queries:['CREATE ROLE "bob"']}]);
    reportRoleLoad({ valid:[], errors:[], all:[{clusterId:'c1', alias:'c1', category:'p', error:'connection refused', queries:[]}] }, { appendLog:true });
    return { summary: runStatusSummary(), c1: rowQueries(runState.byId.get('c1')) };
  })()`);
  assert.equal(r.summary.text, 'Status: Create OK (1 cluster), Load (1/1 failed)');
  assert.deepEqual(r.c1, ['-- Create Role','CREATE ROLE "bob"','-- Load Role','-- ERROR: connection refused']);
});

test('run-status: create+load all-ok chip shows OK (n clusters)', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    state.clusters = [{id:'c1',alias:'c1',category:'p'}];
    runState = null;
    beginRunStatus([{clusterId:'c1'}], 'Create');
    finishRunStatus([{clusterId:'c1', status:'ok', durationMs:1, queries:['CREATE ROLE "bob"']}]);
    reportRoleLoad({ valid:[{clusterId:'c1', alias:'c1', category:'p', durationMs:1, exists:true, queries:['SELECT 1']}], errors:[], all:[{clusterId:'c1', alias:'c1', category:'p', durationMs:1, exists:true, error:'', queries:['SELECT 1']}] }, { appendLog:true });
    return runStatusSummary();
  })()`);
  assert.deepEqual(r, { stateClass:'ok', text:'Status: OK (1 cluster)' });
});

test('run-status: a single unnamed phase keeps the legacy chip text and a verbatim log', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    state.clusters = [{id:'c1',alias:'c1',category:'p'},{id:'c2',alias:'c2',category:'p'}];
    runState = null;
    beginRunStatus([{clusterId:'c1'},{clusterId:'c2'}]); // no phase name
    finishRunStatus([
      {clusterId:'c1', status:'ok', queries:['REVOKE "x" FROM "bob"']},
      {clusterId:'c2', status:'error', message:'no', queries:[]},
    ]);
    return { summary: runStatusSummary(), c1: rowQueries(runState.byId.get('c1')) };
  })()`);
  assert.deepEqual(r.summary, { stateClass:'error', text:'Status: Error (1/2 failed)' });
  assert.deepEqual(r.c1, ['REVOKE "x" FROM "bob"']); // unnamed phase → no -- separators
});

test('run-status: a default (user-initiated) load resets the chip (create log dropped)', () => {
  const r = evalJSON(`(() => {
    ${SETUP_STATE}
    state.clusters = [{id:'c1',alias:'c1',category:'p'}];
    runState = null;
    beginRunStatus([{clusterId:'c1'}], 'Create');
    finishRunStatus([{clusterId:'c1', status:'ok', queries:['CREATE ROLE "bob"']}]);
    reportRoleLoad({ valid:[{clusterId:'c1', alias:'c1', category:'p', queries:['SELECT 1']}], errors:[] }); // default: reset
    return rowQueries(runState.byId.get('c1'));
  })()`);
  assert.deepEqual(r, ['SELECT 1']); // reset to a single unnamed load phase, verbatim
});

test('formatQueryLine: SQL gets a trailing ; but -- comment separators are left alone', () => {
  const r = evalJSON(`[formatQueryLine('SELECT 1'), formatQueryLine('CREATE ROLE "x";'), formatQueryLine('-- Create Role'), formatQueryLine('-- ERROR: permission denied')]`);
  assert.deepEqual(r, ['SELECT 1;', 'CREATE ROLE "x";', '-- Create Role', '-- ERROR: permission denied']);
});

test('enterAlterAfterCreate: switches to Alter, scopes to the sidebar selection, appends the load log', async () => {
  const p = vm.runInContext(`(async () => {
    ${SETUP_STATE}
    state.clusters = [{id:'c1',alias:'c1',category:'p'},{id:'c2',alias:'c2',category:'p'}];
    const _resolve = resolveSelectedClusters, _cats = getSelectedCategories, _ids = getSelectedClusterIDs, _reload = reloadDetails, _footer = updateOpsFooter;
    let reloadOpts = null;
    try {
      getSelectedCategories = () => ['p'];
      getSelectedClusterIDs = () => [];
      resolveSelectedClusters = () => state.clusters;
      updateOpsFooter = () => {};
      reloadDetails = async (opts) => { reloadOpts = opts; };
      currentOp = 'create_role';
      await enterAlterAfterCreate('bob');
      return JSON.stringify({
        currentOp, alterSelected,
        targets: alterTargets,
        scope: alterScopeClusters.map((c) => c.clusterId),
        appliedClusterIds: [...alterAppliedSelection.clusterIds],
        reloadOpts,
      });
    } finally {
      resolveSelectedClusters = _resolve; getSelectedCategories = _cats; getSelectedClusterIDs = _ids;
      reloadDetails = _reload; updateOpsFooter = _footer;
    }
  })()`, ctx);
  const r = JSON.parse(await p);
  assert.equal(r.currentOp, 'alter_user');
  assert.equal(r.alterSelected, 'bob');
  assert.deepEqual(r.targets, { categoryIds: ['p'], clusterIds: [] });
  assert.deepEqual(r.scope, ['c1', 'c2']);          // all originally-selected clusters
  assert.deepEqual(r.reloadOpts, { appendLog: true }); // load appends to the create log, not reset
});
