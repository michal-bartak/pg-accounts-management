package calltemplate

import (
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
	fieldBind // function mode only
)

// gucNameRE validates a role-GUC name (optionally namespaced, e.g. auto_explain.log_min_duration).
// GUC names are case-insensitive bare identifiers, so they are embedded unquoted.
var gucNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

// Build produces SQL for the given execution mode. useQuery is true for function mode (pgx Query).
func Build(call string, args map[string]string, operation, execution string) (sql string, values []any, useQuery bool, err error) {
	execution = model.NormalizeExecution(execution)
	call = normalizeTemplate(call)
	if err := validateCallTemplate(call, operation, execution); err != nil {
		return "", nil, false, err
	}

	switch execution {
	case model.ExecutionStatement:
		sql, err = buildEmbedded(call, args, operation)
		return sql, nil, false, err
	case model.ExecutionBlock:
		// The template is a complete anonymous code block (e.g. DO $tag$ … $tag$;) written
		// by the user. The app runs it verbatim after embedding placeholder values; it adds
		// no DO/delimiter wrapper of its own. Delimiter choice and block structure are the
		// template author's responsibility.
		sql, err = buildEmbedded(call, args, operation)
		return sql, nil, false, err
	default:
		sql, values, err = buildFunctionQuery(call, args, operation)
		return sql, values, true, err
	}
}

// BuildQueryFromTemplate builds function-mode SQL (SELECT + binds). Kept for tests and clarity.
func BuildQueryFromTemplate(call string, args map[string]string, operation string) (query string, values []any, err error) {
	query, values, useQuery, err := Build(call, args, operation, model.ExecutionFunction)
	if err != nil {
		return "", nil, err
	}
	if !useQuery {
		return "", nil, fmt.Errorf("expected function execution mode")
	}
	return query, values, nil
}

func validateCallTemplate(call, operation, execution string) error {
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

	allowed := allowedPlaceholderNames(operation)
	if allowed == nil {
		return fmt.Errorf("unknown operation: %s", operation)
	}
	for _, m := range placeholderTokenRE.FindAllStringSubmatch(call, -1) {
		ph, perr := parsePlaceholderToken(m[1], operation)
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
	return validateCallTemplate(call, operation, model.ExecutionFunction)
}

// ValidateCallTemplateWithExecution validates for the given execution mode.
func ValidateCallTemplateWithExecution(call, operation, execution string) error {
	return validateCallTemplate(call, operation, execution)
}

func allowedPlaceholderNames(operation string) map[string]bool {
	out := AllowedPlaceholders(operation)
	if operation == "remove_role" && out != nil {
		out["rolename"] = true
	}
	return out
}

func placeholderKindForField(operation, field string) (fieldKind, error) {
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
		case "loginname", "parent_role":
			return fieldIdentifier, nil
		case "fullname", "email":
			return fieldLiteral, nil
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

func buildEmbedded(call string, args map[string]string, operation string) (string, error) {
	var b strings.Builder
	last := 0
	for _, loc := range placeholderTokenRE.FindAllStringSubmatchIndex(call, -1) {
		b.WriteString(call[last:loc[0]])
		inner := call[loc[2]:loc[3]]
		ph, err := parsePlaceholderToken(inner, operation)
		if err != nil {
			return "", err
		}
		v, ok := resolveArg(args, ph.field)
		if !ok {
			return "", fmt.Errorf("missing value for ${%s}", ph.field)
		}
		kind, err := placeholderKindForField(operation, ph.field)
		if err != nil {
			return "", err
		}
		if kind == fieldBind {
			return "", fmt.Errorf("${%s} cannot be used in statement/block mode for %s", ph.field, operation)
		}
		switch kind {
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

func buildFunctionQuery(call string, args map[string]string, operation string) (query string, values []any, err error) {
	if err := validateCallTemplate(call, operation, model.ExecutionFunction); err != nil {
		return "", nil, err
	}

	n := 0
	values = make([]any, 0, 8)
	call, err = preprocessArrayOrNull(call, args, operation, &n, &values)
	if err != nil {
		return "", nil, err
	}

	var b strings.Builder
	b.WriteString("SELECT ")
	last := 0
	for _, loc := range placeholderTokenRE.FindAllStringSubmatchIndex(call, -1) {
		b.WriteString(call[last:loc[0]])
		inner := call[loc[2]:loc[3]]
		ph, perr := parsePlaceholderToken(inner, operation)
		if perr != nil {
			return "", nil, perr
		}
		v, ok := resolveArg(args, ph.field)
		if !ok {
			return "", nil, fmt.Errorf("missing value for ${%s}", ph.field)
		}
		switch ph.kind {
		case placeholderArrayConcat:
			n++
			b.WriteString(fmt.Sprintf("$%d::text[]", n))
			values = append(values, buildArrayConcatValue(v, ph.literals))
		case placeholderSimple:
			n++
			b.WriteString(fmt.Sprintf("$%d", n))
			values = append(values, v)
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
