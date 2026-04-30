//go:build webui_preview

package webui

// Phase B service interfaces for /preview/* routes. Each interface returns a
// typed view-model so future Phase C real implementations can drop in without
// touching handler code. View-model structs stay sparse in Phase B — they are
// the interface-shape sketches that Phase C grows.

// PreviewLandingView is the view-model for /preview/.
type PreviewLandingView struct {
	Headline string
	Tagline  string
}

// PreviewOnboardView is the view-model for /preview/onboard.
type PreviewOnboardView struct {
	StepLabel string
	StepIndex int
	StepTotal int
}

// PreviewWorkView is the view-model for /preview/work.
type PreviewWorkView struct {
	BoardTitle string
	ItemCount  int
}

// PreviewWorkItemView is the view-model for /preview/work-item.
type PreviewWorkItemView struct {
	Title  string
	Status string
}

// PreviewProposalView is the view-model for /preview/proposal.
type PreviewProposalView struct {
	Title   string
	Summary string
}

// PreviewLandingSource yields the landing-page view-model.
type PreviewLandingSource interface {
	Landing() (PreviewLandingView, error)
}

// PreviewOnboardSession yields the onboarding view-model.
type PreviewOnboardSession interface {
	Onboard() (PreviewOnboardView, error)
}

// PreviewWorkSource yields the work-board view-model.
type PreviewWorkSource interface {
	Work() (PreviewWorkView, error)
}

// PreviewWorkItemSource yields a single work-item view-model.
type PreviewWorkItemSource interface {
	WorkItem() (PreviewWorkItemView, error)
}

// PreviewProposalSource yields the proposal view-model.
type PreviewProposalSource interface {
	Proposal() (PreviewProposalView, error)
}

// previewServiceRegistry holds one source per preview route. Phase B
// initialises every field with a stub; Phase C swaps individual fields for
// real implementations as they land.
type previewServiceRegistry struct {
	Landing  PreviewLandingSource
	Onboard  PreviewOnboardSession
	Work     PreviewWorkSource
	WorkItem PreviewWorkItemSource
	Proposal PreviewProposalSource
}
