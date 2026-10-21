package services

import (
	"errors"

	"github.com/cambodia-finance/income-expense/internal/accounting"
)

// FieldError is a validation problem with one input; the API reports it under "fields" so the
// form can mark that input.
type FieldError struct {
	Field   string
	Message string
}

func (e *FieldError) Error() string { return e.Message }

func fieldError(field, message string) error {
	return &FieldError{Field: field, Message: message}
}

// Shared errors that always concern the same input.
var errorFieldNames = []struct {
	err   error
	field string
}{
	{accounting.ErrInvalidAmount, "amount"},
	{accounting.ErrAmountTooLarge, "amount"},
	{accounting.ErrInvalidCurrency, "currency"},
	{accounting.ErrInvalidTxnType, "type"},
	{ErrCategoryTypeMismatch, "category_id"},
	{ErrCategoryBranch, "category_id"},
	{ErrDescriptionTooLong, "description"},
	{ErrBranchAccessDenied, "branch_id"},
}

// FieldOf returns the input a shared error is about, or "".
func FieldOf(err error) string {
	for _, m := range errorFieldNames {
		if errors.Is(err, m.err) {
			return m.field
		}
	}
	return ""
}
