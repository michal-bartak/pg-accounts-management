// Package calltemplate parses DB function call templates (${field}, ARRAY[...] || ${field}).
// It must not import config, pg, commands, or batch — keeps config validation cycle-free.
package calltemplate

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	placeholderTokenRE = regexp.MustCompile(`\$\{([^}]+)\}`)
	placeholderNameRE  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	roleLiteralRE      = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	// ARRAY['gr_a', 'gr_b'] || ${parent_role}
	arrayOrNullRE = regexp.MustCompile(`ARRAY\s*\[((?:\s*'[a-zA-Z_][a-zA-Z0-9_]*'\s*,?\s*)+)\]\s*\|\|\s*\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)
	// ARRAY[${parent_role}, 'gr_a', 'gr_b'] → ARRAY['gr_a', 'gr_b'] || ${parent_role}
	arrayLiteralFormRE  = regexp.MustCompile(`ARRAY\[\s*\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}((?:\s*,\s*'[a-zA-Z_][a-zA-Z0-9_]*')+)\s*\]`)
	arrayLiteralQuoteRE = regexp.MustCompile(`'([a-zA-Z_][a-zA-Z0-9_]*)'`)
)

type placeholderKind int

const (
	placeholderSimple placeholderKind = iota
	placeholderArrayConcat // deprecated: ${array_concat:field,lit1,lit2}
)

type parsedPlaceholder struct {
	kind     placeholderKind
	field    string
	literals []string
	raw      string
}

func AllowedPlaceholders(operation string) map[string]bool {
	var names []string
	switch operation {
	case "create_role":
		names = []string{"loginname", "fullname", "email", "parent_role"}
	case "remove_role":
		names = []string{"loginname"}
	case "grant_parents", "revoke_parents":
		names = []string{"loginname", "parent_roles"}
	case "change_password":
		names = []string{"loginname", "new_password"}
	case "set_comment":
		names = []string{"loginname", "comment"}
	case "set_attribute":
		names = []string{"loginname", "attribute"}
	case "set_config":
		names = []string{"loginname", "config_name", "config_value"}
	case "reset_config":
		names = []string{"loginname", "config_name"}
	default:
		return nil
	}
	out := make(map[string]bool, len(names))
	for _, n := range names {
		out[n] = true
	}
	return out
}

func parsePlaceholderToken(inner string, operation string) (parsedPlaceholder, error) {
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return parsedPlaceholder{}, fmt.Errorf("empty placeholder")
	}

	if strings.HasPrefix(inner, "array_concat:") {
		rest := strings.TrimPrefix(inner, "array_concat:")
		items := strings.Split(rest, ",")
		for i := range items {
			items[i] = strings.TrimSpace(items[i])
		}
		if len(items) < 2 {
			return parsedPlaceholder{}, fmt.Errorf("invalid ${array_concat:...}: use ARRAY['fixed', ...] || ${field} instead")
		}
		field := items[0]
		allowed := allowedPlaceholderNames(operation)
		if allowed == nil || !allowed[field] {
			return parsedPlaceholder{}, fmt.Errorf("unknown field %q in ${array_concat:...}", field)
		}
		literals := items[1:]
		for _, lit := range literals {
			if !roleLiteralRE.MatchString(lit) {
				return parsedPlaceholder{}, fmt.Errorf("invalid role name %q", lit)
			}
		}
		return parsedPlaceholder{
			kind:     placeholderArrayConcat,
			field:    field,
			literals: literals,
			raw:      inner,
		}, nil
	}

	if !placeholderNameRE.MatchString(inner) {
		return parsedPlaceholder{}, fmt.Errorf(
			"invalid placeholder ${%s}: use ${loginname} or ARRAY['fixed_role', ...] || ${parent_role}",
			inner,
		)
	}
	return parsedPlaceholder{kind: placeholderSimple, field: inner, raw: inner}, nil
}

func normalizeTemplate(call string) string {
	call = strings.TrimSpace(call)
	// ARRAY[${parent_role}, 'a', 'b'] → ARRAY['a', 'b'] || ${parent_role}
	call = arrayLiteralFormRE.ReplaceAllStringFunc(call, func(match string) string {
		sub := arrayLiteralFormRE.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		field := sub[1]
		var literals []string
		for _, m := range arrayLiteralQuoteRE.FindAllStringSubmatch(sub[2], -1) {
			literals = append(literals, m[1])
		}
		if len(literals) == 0 {
			return match
		}
		quoted := make([]string, len(literals))
		for i, lit := range literals {
			quoted[i] = quoteSQLLiteral(lit)
		}
		return fmt.Sprintf("ARRAY[%s] || ${%s}", strings.Join(quoted, ", "), field)
	})
	return call
}

// quoteSQLLiteral renders a value as a single-quoted SQL string literal. A value containing
// a backslash is emitted as an E'…' escape string with both backslashes and single quotes
// doubled, so the result is safe regardless of the server's standard_conforming_strings
// setting (with it off, a plain '…' literal treats backslash as an escape char and a lone
// trailing backslash could swallow the closing quote). Backslash-free values keep the plain
// '…' form.
func quoteSQLLiteral(s string) string {
	if strings.Contains(s, `\`) {
		return `E'` + strings.NewReplacer(`\`, `\\`, `'`, `''`).Replace(s) + `'`
	}
	return `'` + strings.ReplaceAll(s, `'`, `''`) + `'`
}

func parseQuotedLiteralsFromArrayList(list string) []string {
	var out []string
	for _, m := range arrayLiteralQuoteRE.FindAllStringSubmatch(list, -1) {
		out = append(out, m[1])
	}
	return out
}

func formatFixedArraySQL(literals []string) string {
	quoted := make([]string, len(literals))
	for i, lit := range literals {
		quoted[i] = quoteSQLLiteral(lit)
	}
	return "ARRAY[" + strings.Join(quoted, ", ") + "]::text[]"
}

// preprocessArrayOrNull expands ARRAY['a','b'] || ${field} — empty field becomes || NULL.
func preprocessArrayOrNull(call string, args map[string]string, operation string, n *int, values *[]any) (string, error) {
	allowed := allowedPlaceholderNames(operation)
	var err error
	call = arrayOrNullRE.ReplaceAllStringFunc(call, func(match string) string {
		if err != nil {
			return match
		}
		sub := arrayOrNullRE.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		literals := parseQuotedLiteralsFromArrayList(sub[1])
		field := sub[2]
		if allowed == nil || !allowed[field] {
			err = fmt.Errorf("unknown placeholder ${%s} in ARRAY[...] || ${...}", field)
			return match
		}
		fixed := formatFixedArraySQL(literals)
		v, ok := args[field]
		if !ok {
			err = fmt.Errorf("missing value for ${%s}", field)
			return match
		}
		parts := splitList(v)
		if len(parts) == 0 {
			return fixed + " || NULL"
		}
		*n++
		*values = append(*values, parts)
		return fmt.Sprintf("%s || $%d::text[]", fixed, *n)
	})
	return call, err
}

// splitList splits a comma-separated value into trimmed, non-empty items — lets a single
// parent_role placeholder carry several preconfigured parent groups at once.
func splitList(v string) []string {
	var out []string
	for _, part := range strings.Split(v, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func buildArrayConcatValue(fieldValue string, literals []string) []string {
	parent := strings.TrimSpace(fieldValue)
	if parent == "" {
		out := make([]string, len(literals))
		copy(out, literals)
		return out
	}
	out := make([]string, 0, 1+len(literals))
	out = append(out, parent)
	return append(out, literals...)
}

