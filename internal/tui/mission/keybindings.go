package mission

// ViewID identifies which view is active.
type ViewID int

const (
	ViewHealthPhase ViewID = iota // Startup health check progress
	ViewProposals                 // Pipeline proposal selection
	ViewFleet                     // Two-pane list + preview (fleet monitoring)
	ViewAttached                  // Fullscreen single-run view
)

// OverlayID identifies which overlay is shown (if any).
type OverlayID int

const (
	OverlayNone   OverlayID = iota
	OverlayHealth           // Read-only health overlay from fleet view
	OverlayForm             // Embedded huh form (pipeline selector, modify input)
	OverlayHelp
)

// viewHelp returns help text for a view.
func viewHelp(view ViewID) string {
	switch view {
	case ViewHealthPhase:
		return "q:quit"
	case ViewProposals:
		return "j/k:navigate  Enter:launch  Space:toggle  s:skip  m:modify  n:new  Tab:fleet  ?:help  q:quit"
	case ViewFleet:
		return "j/k:navigate  Enter:attach  n:new  c:cancel  r:retry  o:chat  /:filter  p/Tab:proposals  h:health  ?:help  q:quit"
	case ViewAttached:
		return "Esc:detach  c:cancel  o:chat  ?:help  q:quit"
	default:
		return ""
	}
}

// overlayHelp returns help text for overlays.
func overlayHelp(id OverlayID) string {
	switch id {
	case OverlayHealth:
		return "j/k:scroll  R:refresh  Esc:close"
	case OverlayForm:
		return "Enter:confirm  Esc:cancel"
	case OverlayHelp:
		return "Esc:close  q:quit"
	default:
		return ""
	}
}
