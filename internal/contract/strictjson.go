package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type ObjectSpec map[string]any

type ArraySpec struct{ Element any }

// Go's JSON decoder matches struct fields case-insensitively and lets later
// duplicate keys overwrite earlier values; closed contract schemas allow
// neither, so field names are checked against the spec exactly. codePrefix
// names the document kind in diagnostic codes, e.g. "manifest" yields
// manifest.field.unknown and manifest.field.duplicate.
func CheckExactFields(data []byte, codePrefix, document string, spec ObjectSpec) []Diagnostic {
	walker := fieldWalker{codePrefix: codePrefix}
	decoder := json.NewDecoder(bytes.NewReader(data))
	diagnostics, err := walker.checkSpecValue(decoder, document, spec)
	if err != nil {
		diagnostics = append(diagnostics, Diag(codePrefix+".invalid", document, "cannot inspect fields: %v", err))
	}
	return diagnostics
}

type fieldWalker struct {
	codePrefix string
}

func (walker fieldWalker) checkSpecValue(decoder *json.Decoder, location string, spec any) ([]Diagnostic, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delim, isDelim := token.(json.Delim)
	switch spec := spec.(type) {
	case ObjectSpec:
		if !isDelim || delim != '{' {
			return nil, skipOpened(decoder, token)
		}
		var diagnostics []Diagnostic
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return diagnostics, err
			}
			key := keyToken.(string)
			child, allowed := spec[key]
			if !allowed {
				diagnostics = append(diagnostics, Diag(walker.codePrefix+".field.unknown", location, "field %q is not an exact-case contract field", key))
				if err := skipValue(decoder); err != nil {
					return diagnostics, err
				}
				continue
			}
			if seen[key] {
				diagnostics = append(diagnostics, Diag(walker.codePrefix+".field.duplicate", location+"."+key, "field is declared more than once"))
			}
			seen[key] = true
			childDiagnostics, err := walker.checkSpecValue(decoder, location+"."+key, child)
			diagnostics = append(diagnostics, childDiagnostics...)
			if err != nil {
				return diagnostics, err
			}
		}
		_, err := decoder.Token()
		return diagnostics, err
	case ArraySpec:
		if !isDelim || delim != '[' {
			return nil, skipOpened(decoder, token)
		}
		var diagnostics []Diagnostic
		for index := 0; decoder.More(); index++ {
			childDiagnostics, err := walker.checkSpecValue(decoder, fmt.Sprintf("%s[%d]", location, index), spec.Element)
			diagnostics = append(diagnostics, childDiagnostics...)
			if err != nil {
				return diagnostics, err
			}
		}
		_, err := decoder.Token()
		return diagnostics, err
	default:
		return nil, skipOpened(decoder, token)
	}
}

func skipValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	return skipOpened(decoder, token)
}

func skipOpened(decoder *json.Decoder, token json.Token) error {
	delim, isDelim := token.(json.Delim)
	if !isDelim || (delim != '{' && delim != '[') {
		return nil
	}
	for depth := 1; depth > 0; {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
		}
	}
	return nil
}
