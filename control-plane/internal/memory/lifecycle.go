package memory

// AuthorityScale is the integer scale for stored authority (0–10 => logical 0.0–1.0).
const AuthorityScale = 10

// ApplyAuthorityEvent computes the new authority after a validation or contradiction/failure event.
// currentAuthority is the stored int (0–AuthorityScale). deltaPos used for validation, deltaNeg for contradiction/failure.
// Returns new authority in 0–AuthorityScale.
func ApplyAuthorityEvent(currentAuthority int, eventType string, deltaPos, deltaNeg float64) int {
	if currentAuthority < 0 {
		currentAuthority = 0
	}
	if currentAuthority > AuthorityScale {
		currentAuthority = AuthorityScale
	}
	current := float64(currentAuthority) / float64(AuthorityScale)
	var new float64
	switch eventType {
	case "validation":
		new = current + deltaPos*(1.0-current)
	case "contradiction", "failure":
		new = current - deltaNeg*current
	default:
		return currentAuthority
	}
	if new < 0 {
		new = 0
	}
	if new > 1 {
		new = 1
	}
	out := int(new*float64(AuthorityScale) + 0.5)
	if out < 0 {
		return 0
	}
	if out > AuthorityScale {
		return AuthorityScale
	}
	return out
}
