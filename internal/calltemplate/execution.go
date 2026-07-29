package calltemplate

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/michalbartak/dbaccounts/internal/model"
)

var pgParamRE = regexp.MustCompile(`\$[0-9]+`)

type fieldKind int

const (
	fieldIdentifier     fieldKind = iota
	fieldIdentifierList           // comma-separated role names (GRANT a, b TO …)
	fieldKeywordList              // space-separated keywords (ALTER ROLE … WITH SUPERUSER NOLOGIN)
	fieldConfigName               // a role-GUC name, emitted unquoted (ALTER ROLE … SET work_mem = …)
	fieldLiteral
	fieldCommentValue // a configured comment field: JSON-encoded value → typed SQL (NULL / number / bool / literal)
	fieldBind         // function mode only
)

// gucNameRE validates a role-GUC name (optionally namespaced, e.g. auto_explain.log_min_duration).
// GUC names are case-insensitive bare identifiers, so they are embedded unquoted.
var gucNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

// Build produces SQL for the given execution mode. useQuery is true for function mode (pgx Query).
// commentFields are the configured comment-field keys additionally allowed as placeholders for
// create_role / set_comment (nil for other operations / callers that don't offer them).
func Build(call string, args map[string]string, operation, execution string, commentFields ...string) (sql string, values []any, useQuery bool, err error) {
	execution = model.NormalizeExecution(execution)
	call = normalizeTemplate(call)
	if err := validateCallTemplate(call, operation, execution, commentFields); err != nil {
		return "", nil, false, err
	}

	switch execution {
	case model.ExecutionStatement:
		sql, err = buildEmbedded(call, args, operation, commentFields)
		return sql, nil, false, err
	case model.ExecutionBlock:
		// The template is a complete anonymous code block (e.g. DO $tag$ … $tag$;) written
		// by the user. The app runs it verbatim after embedding placeholder values; it adds
		// no DO/delimiter wrapper of its own. Delimiter choice and block structure are the
		// template author's responsibility.
		sql, err = buildEmbedded(call, args, operation, commentFields)
		return sql, nil, false, err
	default:
		sql, values, err = buildFunctionQuery(call, args, operation, commentFields)
		return sql, values, true, err
	}
}

// BuildQueryFromTemplate builds function-mode SQL (SELECT + binds). Kept for tests and clarity.
func BuildQueryFromTemplate(call string, args map[string]string, operation string, commentFields ...string) (query string, values []any, err error) {
	query, values, useQuery, err := Build(call, args, operation, model.ExecutionFunction, commentFields...)
	if err != nil {
		return "", nil, err
	}
	if !useQuery {
		return "", nil, fmt.Errorf("expected function execution mode")
	}
	return query, values, nil
}

func validateCallTemplate(call, operation, execution string, commentFields []string) error {
	execution = model.NormalizeExecution(execution)
	call = normalizeTemplate(call)
	if call == "" {
		return fmt.Errorf("call template is required")
	}
	if pgParamRE.MatchString(call) {
		return fmt.Errorf("use ${loginname} placeholders, not $1/$2 in the template")
	}

	switch execution {
	case model.ExecutionFunction:
		if strings.Contains(strings.ToUpper(call), "SELECT") {
			return fmt.Errorf("call template must not include SELECT")
		}
		if strings.Contains(call, ";") {
			return fmt.Errorf("call template must not contain semicolons")
		}
	case model.ExecutionStatement:
		if strings.Contains(call, ";") {
			return fmt.Errorf("statement template must not contain semicolons")
		}
	case model.ExecutionBlock:
		// A block template is a complete anonymous code block supplied by the user (DO $tag$
		// … $tag$;). The app runs it verbatim, so semicolons and the DO wrapper are allowed
		// and expected; nothing block-specific to reject here beyond the shared checks below.
	}

	if execution != model.ExecutionFunction && operation == "create_role" {
		if arrayOrNullRE.MatchString(call) || strings.Contains(call, "array_concat:") {
			return fmt.Errorf("create_role with ARRAY || syntax requires execution: function")
		}
	}

	allowed := allowedPlaceholderNames(operation, commentFields)
	if allowed == nil {
		return fmt.Errorf("unknown operation: %s", operation)
	}
	for _, m := range placeholderTokenRE.FindAllStringSubmatch(call, -1) {
		ph, perr := parsePlaceholderToken(m[1], operation, commentFields)
		if perr != nil {
			return perr
		}
		if ph.kind == placeholderSimple && !allowed[ph.field] {
			return fmt.Errorf("unknown placeholder ${%s} for operation %s", ph.field, operation)
		}
	}
	return nil
}

// ValidateCallTemplate validates a function-mode template (config save default).
func ValidateCallTemplate(call, operation string) error {
	return validateCallTemplate(call, operation, model.ExecutionFunction, nil)
}

// ValidateCallTemplateWithExecution validates for the given execution mode. commentFields are the
// configured comment-field keys allowed as placeholders for create_role / set_comment.
func ValidateCallTemplateWithExecution(call, operation, execution string, commentFields ...string) error {
	return validateCallTemplate(call, operation, execution, commentFields)
}

func allowedPlaceholderNames(operation string, commentFields []string) map[string]bool {
	out := AllowedPlaceholders(operation)
	if out == nil {
		return nil
	}
	if operation == "remove_role" {
		out["rolename"] = true
	}
	for f := range commentFieldSet(operation, commentFields) {
		out[f] = true
	}
	return out
}

func placeholderKindForField(operation, field string, commentFields []string) (fieldKind, error) {
	if field == "rolename" {
		field = "loginname"
	}
	switch operation {
	case "remove_role":
		if field == "loginname" {
			return fieldIdentifier, nil
		}
	case "grant_parents", "revoke_parents":
		switch field {
		case "loginname":
			return fieldIdentifier, nil
		case "parent_roles":
			return fieldIdentifierList, nil
		}
	case "change_password":
		switch field {
		case "loginname":
			return fieldIdentifier, nil
		case "new_password":
			return fieldLiteral, nil
		}
	case "set_comment":
		switch field {
		case "loginname":
			return fieldIdentifier, nil
		case "comment":
			return fieldLiteral, nil
		}
		if commentFieldSet(operation, commentFields)[field] {
			return fieldCommentValue, nil
		}
	case "set_attribute":
		switch field {
		case "loginname":
			return fieldIdentifier, nil
		case "attributes", "attribute":
			// One or more keywords (SUPERUSER NOLOGIN …): embedded unquoted, whitelisted upstream.
			// ${attribute} is the legacy alias for ${attributes}.
			return fieldKeywordList, nil
		}
	case "create_role":
		switch field {
		case "loginname":
			return fieldIdentifier, nil
		case "parent_roles":
			// Same field as grant/revoke: statement → double-quoted identifier list; function →
			// inline ARRAY['a','b'] literal (buildFunctionQuery special-cases fieldIdentifierList).
			return fieldIdentifierList, nil
		}
		if commentFieldSet(operation, commentFields)[field] {
			return fieldCommentValue, nil
		}
	case "set_config":
		switch field {
		case "loginname":
			return fieldIdentifier, nil
		case "config_name":
			return fieldConfigName, nil
		case "config_value":
			return fieldLiteral, nil
		}
	case "reset_config":
		switch field {
		case "loginname":
			return fieldIdentifier, nil
		case "config_name":
			return fieldConfigName, nil
		}
	}
	return fieldBind, nil
}

func resolveArg(args map[string]string, field string) (string, bool) {
	if v, ok := args[field]; ok {
		return v, true
	}
	if field == "rolename" {
		v, ok := args["loginname"]
		return v, ok
	}
	return "", false
}

// quoteSQLIdentifier double-quotes a role identifier so case is preserved and special
// characters are safe (embedded `"` are doubled). A comma is rejected because it is the
// delimiter for identifier lists (parent_roles); a NUL byte is invalid in an identifier.
func quoteSQLIdentifier(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("identifier value is required")
	}
	if strings.ContainsAny(name, ",\x00") {
		return "", fmt.Errorf("invalid identifier %q: commas and NUL bytes are not allowed", name)
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`, nil
}

func quoteSQLIdentifierList(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("at least one role name is required")
	}
	parts := strings.Split(value, ",")
	quoted := make([]string, 0, len(parts))
	for _, p := range parts {
		q, err := quoteSQLIdentifier(p)
		if err != nil {
			return "", err
		}
		quoted = append(quoted, q)
	}
	return strings.Join(quoted, ", "), nil
}

// renderRoleArrayVerbatim renders a comma-separated role list as an inline text[] array literal for
// FUNCTION mode: ARRAY['gr_a', 'gr_b']. Elements are single-quoted but placed VERBATIM — no `'`→`''`
// doubling and no identifier double-quoting (per the ${parent_roles} function-mode contract). A
// value containing a single quote or NUL is rejected rather than embedded, as the only guard against
// producing broken / injectable SQL. An empty list becomes bare NULL.
func renderRoleArrayVerbatim(value string) (string, error) {
	parts := splitList(value)
	if len(parts) == 0 {
		return "NULL", nil
	}
	quoted := make([]string, len(parts))
	for i, p := range parts {
		if strings.ContainsAny(p, "'\x00") {
			return "", fmt.Errorf("invalid role name %q in ${parent_roles}: single quotes are not allowed", p)
		}
		quoted[i] = "'" + p + "'"
	}
	return "ARRAY[" + strings.Join(quoted, ", ") + "]", nil
}

// quoteSQLKeywordList renders a whitespace-separated keyword list (e.g. role attributes) as a
// single space-joined clause. Each token must be a bare identifier; the keyword whitelist is
// enforced upstream in commands.ValidateOperation.
func quoteSQLKeywordList(value string) (string, error) {
	kws := strings.Fields(value)
	if len(kws) == 0 {
		return "", fmt.Errorf("at least one keyword is required")
	}
	for _, kw := range kws {
		if !roleLiteralRE.MatchString(kw) {
			return "", fmt.Errorf("invalid keyword %q: use letters, digits, underscore", kw)
		}
	}
	return strings.Join(kws, " "), nil
}

// validConfigName returns a role-GUC name embedded unquoted after validating its syntax
// (bare, optionally namespaced identifier). GUC names are case-insensitive, so they are not
// double-quoted; validation is the injection guard.
func validConfigName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if !gucNameRE.MatchString(name) {
		return "", fmt.Errorf("invalid setting name: %q", name)
	}
	return name, nil
}

// renderCommentFieldSQL turns a comment field's JSON-encoded value into an embedded SQL literal:
// absent/empty/JSON null/empty-string → bare NULL (unquoted); JSON number → the numeric token
// verbatim; JSON bool → TRUE/FALSE; JSON string → a quoted literal; JSON array/object → the compact
// JSON text as a quoted literal. A value that fails to parse as JSON is treated as a plain string.
func renderCommentFieldSQL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "NULL"
	}
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return quoteSQLLiteral(raw) // not JSON: plain, non-empty string
	}
	switch t := v.(type) {
	case nil:
		return "NULL"
	case bool:
		if t {
			return "TRUE"
		}
		return "FALSE"
	case float64:
		return raw // re-emit the original JSON number token (avoids float reformatting)
	case string:
		if strings.TrimSpace(t) == "" {
			return "NULL"
		}
		return quoteSQLLiteral(t)
	default: // array / object → raw JSON text
		b, err := json.Marshal(v)
		if err != nil {
			return "NULL"
		}
		return quoteSQLLiteral(string(b))
	}
}

// commentFieldBindValue converts a comment field's JSON-encoded value into a typed Go value for a
// pgx bind (function mode): nil for NULL, the number/bool/string as-is, arrays/objects as JSON text.
func commentFieldBindValue(raw string) any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw // not JSON: plain string
	}
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		if strings.TrimSpace(t) == "" {
			return nil
		}
		return t
	case bool, float64:
		return t
	default: // array / object → JSON text
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		return string(b)
	}
}

func buildEmbedded(call string, args map[string]string, operation string, commentFields []string) (string, error) {
	var b strings.Builder
	last := 0
	for _, loc := range placeholderTokenRE.FindAllStringSubmatchIndex(call, -1) {
		b.WriteString(call[last:loc[0]])
		inner := call[loc[2]:loc[3]]
		ph, err := parsePlaceholderToken(inner, operation, commentFields)
		if err != nil {
			return "", err
		}
		kind, err := placeholderKindForField(operation, ph.field, commentFields)
		if err != nil {
			return "", err
		}
		if kind == fieldBind {
			return "", fmt.Errorf("${%s} cannot be used in statement/block mode for %s", ph.field, operation)
		}
		v, ok := resolveArg(args, ph.field)
		// A comment field is optional — an absent/empty value resolves to SQL NULL.
		if !ok && kind != fieldCommentValue {
			return "", fmt.Errorf("missing value for ${%s}", ph.field)
		}
		switch kind {
		case fieldCommentValue:
			b.WriteString(renderCommentFieldSQL(v))
		case fieldIdentifier:
			quoted, err := quoteSQLIdentifier(v)
			if err != nil {
				return "", err
			}
			b.WriteString(quoted)
		case fieldIdentifierList:
			quoted, err := quoteSQLIdentifierList(v)
			if err != nil {
				return "", err
			}
			b.WriteString(quoted)
		case fieldKeywordList:
			quoted, err := quoteSQLKeywordList(v)
			if err != nil {
				return "", err
			}
			b.WriteString(quoted)
		case fieldConfigName:
			name, err := validConfigName(v)
			if err != nil {
				return "", err
			}
			b.WriteString(name)
		case fieldLiteral:
			b.WriteString(quoteSQLLiteral(v))
		}
		last = loc[1]
	}
	b.WriteString(call[last:])
	out := b.String()
	if strings.Contains(out, "${") {
		return "", fmt.Errorf("call template has unresolved placeholders")
	}
	return out, nil
}

func buildFunctionQuery(call string, args map[string]string, operation string, commentFields []string) (query string, values []any, err error) {
	if err := validateCallTemplate(call, operation, model.ExecutionFunction, commentFields); err != nil {
		return "", nil, err
	}

	cfSet := commentFieldSet(operation, commentFields)
	n := 0
	values = make([]any, 0, 8)
	call, err = preprocessArrayOrNull(call, args, operation, &n, &values, commentFields)
	if err != nil {
		return "", nil, err
	}

	var b strings.Builder
	b.WriteString("SELECT ")
	last := 0
	for _, loc := range placeholderTokenRE.FindAllStringSubmatchIndex(call, -1) {
		b.WriteString(call[last:loc[0]])
		inner := call[loc[2]:loc[3]]
		ph, perr := parsePlaceholderToken(inner, operation, commentFields)
		if perr != nil {
			return "", nil, perr
		}
		v, ok := resolveArg(args, ph.field)
		// A comment field is optional — an absent value binds as SQL NULL.
		if !ok && !cfSet[ph.field] {
			return "", nil, fmt.Errorf("missing value for ${%s}", ph.field)
		}
		switch ph.kind {
		case placeholderArrayConcat:
			n++
			b.WriteString(fmt.Sprintf("$%d::text[]", n))
			values = append(values, buildArrayConcatValue(v, ph.literals))
		case placeholderSimple:
			kind, kerr := placeholderKindForField(operation, ph.field, commentFields)
			if kerr != nil {
				return "", nil, kerr
			}
			if kind == fieldIdentifierList {
				// parent_roles in function mode → inline ARRAY['a','b'] literal, not a bind.
				arr, aerr := renderRoleArrayVerbatim(v)
				if aerr != nil {
					return "", nil, aerr
				}
				b.WriteString(arr)
				break
			}
			n++
			b.WriteString(fmt.Sprintf("$%d", n))
			if cfSet[ph.field] {
				values = append(values, commentFieldBindValue(v))
			} else {
				values = append(values, v)
			}
		}
		last = loc[1]
	}
	b.WriteString(call[last:])
	query = b.String()
	if strings.Contains(query, "${") {
		return "", nil, fmt.Errorf("call template has unresolved placeholders")
	}
	return query, values, nil
}
