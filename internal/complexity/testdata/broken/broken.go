// Deliberately broken Go source. Lives under testdata/ so Go tooling
// ignores it; the complexity analyzer is expected to surface a parse error
// when pointed at this file.
package broken

func Broken() int {
	return 1 +
}
