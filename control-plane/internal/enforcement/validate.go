package enforcement

import (
	"errors"
	"fmt"
	"strings"
)

const maxProposalBytes = 32768
const maxAgentIDBytes = 512

// ErrDisabled is returned when enforcement is not enabled in config.
var ErrDisabled = errors.New("enforcement is disabled in config: set enforcement.enabled to true to use this endpoint")

// ValidateEvaluateRequest checks required fields and bounds.
func ValidateEvaluateRequest(req EvaluateRequest) error {
	if strings.TrimSpace(req.ProposalText) == "" {
		return errors.New("proposal_text is required")
	}
	if len(req.ProposalText) > maxProposalBytes {
		return fmt.Errorf("proposal_text exceeds maximum size (%d bytes)", maxProposalBytes)
	}
	if len(req.AgentID) > maxAgentIDBytes {
		return fmt.Errorf("agent_id exceeds maximum size (%d bytes)", maxAgentIDBytes)
	}
	return nil
}
