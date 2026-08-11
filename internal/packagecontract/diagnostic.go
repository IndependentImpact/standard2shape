package packagecontract

import "github.com/IndependentImpact/standard2shape/internal/contract"

type Diagnostic = contract.Diagnostic

type ContractError = contract.Error

func contractError(diagnostics []Diagnostic) error {
	return contract.ErrorFor(diagnostics)
}

func diagnostic(code, location, format string, args ...any) Diagnostic {
	return contract.Diag(code, location, format, args...)
}
