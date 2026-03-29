package recall

import "strings"

// enrichSituationQueryWithProposal appends proposal text so lexical and semantic retrieval
// share the same expanded situation string (charter: situation > words).
func enrichSituationQueryWithProposal(situationQuery, proposalText string) string {
	sq := strings.TrimSpace(situationQuery)
	pt := strings.TrimSpace(proposalText)
	if pt == "" {
		return sq
	}
	if sq == "" {
		return pt
	}
	return strings.TrimSpace(sq + " " + pt)
}
