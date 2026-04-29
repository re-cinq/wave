package worksource

import "path"

// matches reports whether rec applies to ref. Inactive bindings never match.
//
// Semantics:
//   - Forge: exact equality (case-sensitive, matching the storage layer).
//   - Repo: path.Match(rec.RepoPattern, ref.Repo).
//   - LabelFilter: any-of. Empty filter means "all labels accepted".
//   - State: subset. Empty rec.State means "any state". A rec.State of "any"
//     also accepts every state.
//   - Kinds: subset. Empty means "any kind".
func matches(rec BindingRecord, ref WorkItemRef) bool {
	if !rec.Active {
		return false
	}
	if rec.Forge != ref.Forge {
		return false
	}
	ok, err := path.Match(rec.RepoPattern, ref.Repo)
	if err != nil || !ok {
		return false
	}
	if !stateAccepts(rec.State, ref.State) {
		return false
	}
	if !kindAccepts(rec.Kinds, ref.Kind) {
		return false
	}
	if !labelsAccept(rec.LabelFilter, ref.Labels) {
		return false
	}
	return true
}

// stateAccepts is true when the binding's state filter admits the work-item's
// state. Empty filter or "any" accept everything.
func stateAccepts(filter, refState string) bool {
	if filter == "" || filter == "any" {
		return true
	}
	return filter == refState
}

// kindAccepts is true when the binding's kind filter admits the work-item's
// kind. Empty filter accepts everything.
func kindAccepts(filter []string, refKind string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, k := range filter {
		if k == refKind {
			return true
		}
	}
	return false
}

// labelsAccept is true when the binding's label filter admits the work-item.
// any-of semantics: at least one filter label must appear on the work-item.
// An empty filter accepts everything.
func labelsAccept(filter, refLabels []string) bool {
	if len(filter) == 0 {
		return true
	}
	want := make(map[string]struct{}, len(filter))
	for _, l := range filter {
		want[l] = struct{}{}
	}
	for _, l := range refLabels {
		if _, ok := want[l]; ok {
			return true
		}
	}
	return false
}
