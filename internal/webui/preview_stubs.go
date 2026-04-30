//go:build webui_preview

package webui

// Phase B deterministic stubs. Each method returns constant literals — no
// time.Now, no rand, no map iteration. Determinism is contract: tests assert
// byte-equality across repeated calls so Phase C can swap real implementations
// behind feature flags without test churn.

type previewLandingStub struct{}

func (previewLandingStub) Landing() (PreviewLandingView, error) {
	return PreviewLandingView{
		Headline: "Wave preview",
		Tagline:  "phase B stub services",
	}, nil
}

type previewOnboardStub struct{}

func (previewOnboardStub) Onboard() (PreviewOnboardView, error) {
	return PreviewOnboardView{
		StepLabel: "Connect repository",
		StepIndex: 1,
		StepTotal: 4,
	}, nil
}

type previewWorkStub struct{}

func (previewWorkStub) Work() (PreviewWorkView, error) {
	return PreviewWorkView{
		BoardTitle: "Active work",
		ItemCount:  3,
	}, nil
}

type previewWorkItemStub struct{}

func (previewWorkItemStub) WorkItem() (PreviewWorkItemView, error) {
	return PreviewWorkItemView{
		Title:  "Sample issue",
		Status: "in-progress",
	}, nil
}

type previewProposalStub struct{}

func (previewProposalStub) Proposal() (PreviewProposalView, error) {
	return PreviewProposalView{
		Title:   "Sample proposal",
		Summary: "Stub proposal body for designer review.",
	}, nil
}

// defaultPreviewServices returns a registry populated with stub sources for
// every route. Called once from preview.go init().
func defaultPreviewServices() *previewServiceRegistry {
	return &previewServiceRegistry{
		Landing:  previewLandingStub{},
		Onboard:  previewOnboardStub{},
		Work:     previewWorkStub{},
		WorkItem: previewWorkItemStub{},
		Proposal: previewProposalStub{},
	}
}
