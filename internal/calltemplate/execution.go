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
	fieldIdentifier fieldKind = iota
	fieldIdentifierList // comma-separated role names (GRANT a, b TO …)
	fieldLiteral
	fieldBind // function mode only
)

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
		body, err := buildEmbedded(call, args, operation)
		if err != nil {
			return "", nil, false, err
		}
		sql = "DO $dbaccounts$\nBEGIN\n" + body + "\nEND\n$dbaccounts$;"
		return sql, nil, false, nil
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
		lower := strings.ToLower(call)
		if strings.Contains(lower, "do $dbaccounts") || strings.HasPrefix(strings.TrimSpace(lower), "do ") {
			return fmt.Errorf("block template must not include DO; the app wraps your statements")
		}
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
	case "create_role":
		switch field {
		case "loginname", "parent_role":
			return fieldIdentifier, nil
		case "fullname", "email":
			return fieldLiteral, nil
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

func quoteSQLIdentifier(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("identifier value is required")
	}
	if !roleLiteralRE.MatchString(name) {
		return "", fmt.Errorf("invalid identifier %q: use letters, digits, underscore", name)
	}
	return name, nil
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
