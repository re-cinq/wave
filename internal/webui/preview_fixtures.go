//go:build webui_preview

package webui

// Fixtures for the /preview/* routes. Phase A keeps these as empty typed
// structs so handler signatures and template binding contracts are explicit;
// the templates currently render hard-coded HTML inherited from the design
// mockups. Phase B will populate these structs from real services without
// changing handler signatures.

type previewLandingFixture struct{}

type previewOnboardFixture struct{}

type previewWorkFixture struct{}

type previewWorkItemFixture struct{}

type previewProposalFixture struct{}

var (
	landingFixture  = previewLandingFixture{}
	onboardFixture  = previewOnboardFixture{}
	workFixture     = previewWorkFixture{}
	workItemFixture = previewWorkItemFixture{}
	proposalFixture = previewProposalFixture{}
)
