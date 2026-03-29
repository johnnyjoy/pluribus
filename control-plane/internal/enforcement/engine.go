package enforcement

// EvaluationEngine identifies the enforcement implementation (stable wire value).
const EvaluationEngineRuleBasedHeuristicV1 = "rule_based_heuristic_v1"

// EvaluationNoteRuleBased is returned on every EvaluateResponse so clients do not assume general NL reasoning.
const EvaluationNoteRuleBased = "Rule-based gate only (reason codes on triggered_memories); not unrestricted natural-language reasoning. Unmodelled constraints do not produce hits."
