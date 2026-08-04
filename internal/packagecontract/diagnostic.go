package packagecontract

import (
	"fmt"
	"sort"
	"strings"
)

type Diagnostic struct {
	Code     string `json:"code"`
	Location string `json:"location"`
	Message  string `json:"message"`
}

type ContractError struct {
	Diagnostics []Diagnostic
}

func (err *ContractError) Error() string {
	if err == nil || len(err.Diagnostics) == 0 {
		return "package contract failed"
	}
	first := err.Diagnostics[0]
	if len(err.Diagnostics) == 1 {
		return fmt.Sprintf("%s at %s: %s", first.Code, first.Location, first.Message)
	}
	return fmt.Sprintf("%s at %s: %s (and %d more diagnostics)", first.Code, first.Location, first.Message, len(err.Diagnostics)-1)
}

func contractError(diagnostics []Diagnostic) error {
	if len(diagnostics) == 0 {
		return nil
	}
	sort.SliceStable(diagnostics, func(i, j int) bool {
		left := diagnostics[i].Location + "\x00" + diagnostics[i].Code + "\x00" + diagnostics[i].Message
		right := diagnostics[j].Location + "\x00" + diagnostics[j].Code + "\x00" + diagnostics[j].Message
		return strings.Compare(left, right) < 0
	})
	return &ContractError{Diagnostics: diagnostics}
}

func diagnostic(code, location, format string, args ...any) Diagnostic {
	return Diagnostic{Code: code, Location: location, Message: fmt.Sprintf(format, args...)}
}
