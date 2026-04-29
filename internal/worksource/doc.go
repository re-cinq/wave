// Package worksource is the domain service over the work-source binding table
// introduced by PRE-5 (state.WorksourceStore). It maps forge queries (forge +
// repo glob + label any-of + work-item state/kind filters) onto a pipeline +
// trigger mode, and exposes a MatchBindings query that selects bindings
// applicable to a given WorkItemRef.
//
// The service is the only translator between the typed domain shape
// (BindingSpec / BindingRecord) and the storage shape (state.WorksourceBindingRecord
// with opaque JSON selector). Trigger names use dashed form externally
// (on-demand|on-label|on-open|scheduled) and are normalised to underscored
// form (state.WorksourceTrigger) before persistence.
//
// DeleteBinding is a soft-delete: it maps to state.DeactivateBinding so run
// history retains the binding context. Glob syntax is path.Match — *, ?, and
// [class]; no double-star.
package worksource
