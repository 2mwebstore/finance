package controllers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"fmt"
	"reflect"
	"strings"

	"github.com/cambodia-finance/income-expense/internal/auth"
	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// Report validation problems by their JSON names (branch_id), not Go field names (BranchID).
func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(f reflect.StructField) string {
			name := strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
	}
}

// fieldLabels are the names people see in the forms; anything else is shown with spaces for underscores.
var fieldLabels = map[string]string{
	"branch_id":        "branch",
	"category_id":      "category",
	"transaction_date": "date",
	"full_name":        "full name",
	"password":         "PIN",
	"new_password":     "new PIN",
	"current_password": "current PIN",
	"identifier":       "username or email",
	"setup_token":      "setup token",
	"role_codes":       "roles",
	"branch_ids":       "branches",
	"user_ids":         "users",
	"permission_codes": "permissions",
	"is_active":        "status",
	"balance_date":     "date",
	"reconcile_date":   "date",
	"expected_amount":  "expected amount",
	"actual_amount":    "actual amount",
}

func fieldLabel(field string) string {
	if label, ok := fieldLabels[field]; ok {
		return label
	}
	return strings.ReplaceAll(field, "_", " ")
}

// bindJSON decodes the request body and checks its binding tags. On failure it answers 400 with
// a readable message ("name is required") and a per-field map the forms use to mark inputs.
func bindJSON(c *gin.Context, dst interface{}) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		writeBindError(c, err)
		return false
	}
	return true
}

func writeBindError(c *gin.Context, err error) {
	var verrs validator.ValidationErrors
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &verrs):
		fields := map[string]string{}
		first := ""
		for _, fe := range verrs {
			if _, seen := fields[fe.Field()]; seen {
				continue
			}
			msg := validationMessage(fe)
			fields[fe.Field()] = msg
			if first == "" {
				first = msg
			}
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": first, "fields": fields})
	case errors.As(err, &typeErr) && typeErr.Field != "":
		field := typeErr.Field[strings.LastIndex(typeErr.Field, ".")+1:]
		msg := fieldLabel(field) + " is not valid"
		c.JSON(http.StatusBadRequest, gin.H{"error": msg, "fields": gin.H{field: msg}})
	case errors.Is(err, io.EOF):
		c.JSON(http.StatusBadRequest, gin.H{"error": "request body is required"})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "request body is not valid JSON"})
	}
}

func validationMessage(fe validator.FieldError) string {
	label := fieldLabel(fe.Field())
	isPIN := fe.Field() == "password" || fe.Field() == "new_password"
	isList := fe.Kind() == reflect.Slice || fe.Kind() == reflect.Array || fe.Kind() == reflect.Map
	switch fe.Tag() {
	case "required":
		if isList {
			return "choose at least one of the " + label
		}
		return label + " is required"
	case "min", "max":
		if isPIN {
			return fmt.Sprintf("%s must be %d to %d digits", label, auth.MinPINLength, auth.MaxPINLength)
		}
		bound := "at least "
		if fe.Tag() == "max" {
			bound = "at most "
		}
		if isList {
			return "choose " + bound + fe.Param() + " " + label
		}
		return label + " must be " + bound + fe.Param() + " characters"
	case "len":
		if isPIN {
			return label + " must be exactly " + fe.Param() + " digits"
		}
		return label + " must be exactly " + fe.Param() + " characters"
	case "numeric":
		if isPIN {
			return label + " must contain digits only"
		}
		return label + " must be a number"
	case "email":
		return label + " must be a valid email address"
	case "uuid":
		return "choose a valid " + label
	case "oneof":
		return label + " must be " + strings.Join(strings.Fields(fe.Param()), " or ")
	default:
		return label + " is not valid"
	}
}

// errorFields points a service error at the form field it is about, when there is one.
func errorFields(err error) gin.H {
	var fe *services.FieldError
	if errors.As(err, &fe) {
		return gin.H{fe.Field: fe.Message}
	}
	if field := services.FieldOf(err); field != "" {
		return gin.H{field: err.Error()}
	}
	return nil
}
