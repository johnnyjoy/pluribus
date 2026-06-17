package parity

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"control-plane/internal/recall"
)

// ContractField names compared between REST and MCP surfaces.
const (
	FieldMemoryID       = "memory_id"
	FieldStatement      = "statement"
	FieldSchemaType     = "schema_type"
	FieldLifecycleRole  = "lifecycle_role"
	FieldStatus         = "status"
	FieldApplicability  = "applicability"
	FieldScope          = "scope"
	FieldNegativeScope  = "negative_scope"
	FieldUseInstruction = "use_instruction"
	FieldMisuseWarning  = "misuse_warning"
	FieldSourceType     = "source_type"
	FieldAuthorityBasis = "authority_basis"
	FieldAuthority      = "authority"
	FieldUtilityScore   = "utility_score"
	FieldQualityState   = "quality_state"
	FieldQualityScore   = "quality_score"
	FieldSupersededBy   = "superseded_by"
)

// ComparedFields is the canonical ordered list of agent contract fields for parity.
var ComparedFields = []string{
	FieldMemoryID, FieldStatement, FieldSchemaType, FieldLifecycleRole, FieldStatus,
	FieldApplicability, FieldScope, FieldNegativeScope, FieldUseInstruction, FieldMisuseWarning,
	FieldSourceType, FieldAuthorityBasis, FieldAuthority, FieldUtilityScore,
	FieldQualityState, FieldQualityScore, FieldSupersededBy,
}

// Mismatch codes emitted by the parity comparator.
const (
	MismatchMissingFieldInREST     = "missing_field_in_rest"
	MismatchMissingFieldInMCP      = "missing_field_in_mcp"
	MismatchFieldValue             = "field_value_mismatch"
	MismatchLifecycleRole          = "lifecycle_role_mismatch"
	MismatchStatus                 = "status_mismatch"
	MismatchSchemaType             = "schema_type_mismatch"
	MismatchScope                  = "scope_mismatch"
	MismatchNegativeScope          = "negative_scope_mismatch"
	MismatchUseInstruction         = "use_instruction_mismatch"
	MismatchMisuseWarning          = "misuse_warning_mismatch"
	MismatchProvenance             = "provenance_mismatch"
	MismatchAuthorityBasis         = "authority_basis_mismatch"
	MismatchUtilityScore           = "utility_score_mismatch"
	MismatchQualityState           = "quality_state_mismatch"
	MismatchQualityScore           = "quality_score_mismatch"
	MismatchSupersededBy           = "superseded_by_mismatch"
	MismatchMCPTextOnlyWithoutJSON = "mcp_text_only_without_json"
)

func fieldMismatchCode(field string) string {
	switch field {
	case FieldLifecycleRole:
		return MismatchLifecycleRole
	case FieldStatus:
		return MismatchStatus
	case FieldSchemaType:
		return MismatchSchemaType
	case FieldScope:
		return MismatchScope
	case FieldNegativeScope:
		return MismatchNegativeScope
	case FieldUseInstruction:
		return MismatchUseInstruction
	case FieldMisuseWarning:
		return MismatchMisuseWarning
	case FieldSourceType:
		return MismatchProvenance
	case FieldAuthorityBasis:
		return MismatchAuthorityBasis
	case FieldUtilityScore:
		return MismatchUtilityScore
	case FieldQualityState:
		return MismatchQualityState
	case FieldQualityScore:
		return MismatchQualityScore
	case FieldSupersededBy:
		return MismatchSupersededBy
	default:
		return MismatchFieldValue
	}
}

func restFieldValue(it recall.MemoryItem, field string) (any, bool) {
	switch field {
	case FieldMemoryID:
		return it.ID, it.ID != ""
	case FieldStatement:
		return it.Statement, it.Statement != ""
	case FieldSchemaType:
		return it.SchemaType, it.SchemaType != ""
	case FieldLifecycleRole:
		return it.LifecycleRole, it.LifecycleRole != ""
	case FieldStatus:
		return it.Status, it.Status != ""
	case FieldApplicability:
		return string(it.Applicability), it.Applicability != ""
	case FieldScope:
		return it.Scope, it.Scope != ""
	case FieldNegativeScope:
		return copyStrings(it.NegativeScope), len(it.NegativeScope) > 0
	case FieldUseInstruction:
		return it.UseInstruction, it.UseInstruction != ""
	case FieldMisuseWarning:
		return it.MisuseWarning, it.MisuseWarning != ""
	case FieldSourceType:
		return it.SourceType, it.SourceType != ""
	case FieldAuthorityBasis:
		return it.AuthorityBasis, it.AuthorityBasis != ""
	case FieldAuthority:
		return it.Authority, true
	case FieldUtilityScore:
		return it.UtilityScore, it.UtilityScore != nil
	case FieldQualityState:
		return it.QualityState, it.QualityState != ""
	case FieldQualityScore:
		return it.QualityScore, it.QualityScore != nil
	case FieldSupersededBy:
		return it.SupersededBy, it.SupersededBy != ""
	default:
		return nil, false
	}
}

func mcpFieldValue(it recall.MemoryItem, field string) (any, bool) {
	return restFieldValue(it, field)
}

func copyStrings(xs []string) []string {
	if len(xs) == 0 {
		return nil
	}
	out := make([]string, len(xs))
	copy(out, xs)
	sort.Strings(out)
	return out
}

func floatEqual(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return math.Abs(*a-*b) < 1e-9
}

func valuesEqual(field string, a, b any) bool {
	switch field {
	case FieldNegativeScope:
		as, okA := a.([]string)
		bs, okB := b.([]string)
		if !okA || !okB {
			return false
		}
		if len(as) != len(bs) {
			return false
		}
		for i := range as {
			if as[i] != bs[i] {
				return false
			}
		}
		return true
	case FieldUtilityScore, FieldQualityScore:
		ap, okA := a.(*float64)
		bp, okB := b.(*float64)
		if !okA || !okB {
			return false
		}
		return floatEqual(ap, bp)
	case FieldAuthority:
		ai, okA := a.(int)
		bi, okB := b.(int)
		return okA && okB && ai == bi
	default:
		return strings.TrimSpace(fmt.Sprint(a)) == strings.TrimSpace(fmt.Sprint(b))
	}
}
