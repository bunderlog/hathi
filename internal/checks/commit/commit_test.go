package commit

import (
	"strings"
	"testing"

	"github.com/bunderlog/hathi/internal/checks"
)

const notOneOf = "is not one of " +
	"build, chore, ci, docs, feat, fix, perf, refactor, revert, style, test"

// reason is why conventional-commits with `rules` fails `message`, "" when it passes.
func reason(t *testing.T, message string, rules Rules) string {
	t.Helper()
	found, err := Check(rules).Find(checks.Input{Message: message})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) > 1 {
		t.Fatalf("%q: %d findings, want at most one", message, len(found))
	}
	if len(found) == 0 {
		return ""
	}
	return found[0].Reason
}

func expect(t *testing.T, message string, rules Rules, want string) {
	t.Helper()
	if got := reason(t, message, rules); got != want {
		t.Errorf("%q: got %q, want %q", message, got, want)
	}
}

// baloo: test_commit.py test_accepts
func TestCheck_Accepts(t *testing.T) {
	for _, message := range []string{
		"feat: add login",
		"fix(adr): correct a date",
		"feat!: drop node",
		"refactor(api/auth)!: split parse\n\nBody.",
		"feat: x\n\nBreaking change: a body line, not a footer.\n\nRefs: #1",
		"feat(multi word): a scope with spaces",
		"fix(ZrOpticsTab): a scope in any case",
		"docs: explain `README` and the API",
		"feat: 2 new checks",
		"feat: " + strings.Repeat("x", 94),
		"fix: x\n\n" + strings.Repeat("y", 100),
		"feat: é" + strings.Repeat("x", 93),
		"feat: x\n# ------------------------ >8 ------------------------\n" + strings.Repeat("z", 200),
		"# Please enter the commit message\n\ndocs: explain config",
		"Merge branch 'main' into feature",
		`Revert "feat: add login"`,
		`Reapply "feat: add login"`,
		"Merge pull request #7 from a/b",
		"Merge remote-tracking branch 'origin/main'",
		"fixup! feat: add login",
		"squash! feat: add login",
		"feat: x\n# a comment\n\nBody.",
		"feat: x\n# ------------------------ >8 ------------------------\ndiff --git a b",
		"feat: x\n\nBody.\n\nBREAKING CHANGE: drops node",
		"feat: x\n\nBREAKING-CHANGE: drops node\nRefs: #12",
		"feat: x\n\nBreaking change: not a token, so a body line",
		"feat: x\n\nRefs: #1\nbreaking change: part of the Refs value",
		"feat: x\n\nReviewed-by: Z\nCloses #42",
		"",
	} {
		t.Run(message, func(t *testing.T) { expect(t, message, Default, "") })
	}
}

// The examples in the Conventional Commits 1.0.0 specification, with LF and with CRLF.
// baloo: test_commit.py test_accepts_the_specs_examples
func TestCheck_AcceptsTheSpecsExamples(t *testing.T) {
	for _, message := range []string{
		"feat: allow provided config object to extend other configs\n\nBREAKING CHANGE: " +
			"`extends` key in config file is now used for extending other config files",
		"feat!: send an email to the customer when a product is shipped",
		"feat(api)!: send an email to the customer when a product is shipped",
		"feat!: drop support for Node 6\n\nBREAKING CHANGE: use JavaScript features not " +
			"available in Node 6.",
		"docs: correct spelling of CHANGELOG",
		"feat(lang): add Polish language",
		"fix: prevent racing of requests\n\nIntroduce a request id and a reference to latest " +
			"request. Dismiss\nincoming responses other than from latest request.\n\nRemove " +
			"timeouts which were used to mitigate the racing issue but are\nobsolete now.\n\n" +
			"Reviewed-by: Z\nRefs: #123",
		"revert: let us never again speak of the noodle incident\n\nRefs: 676104e, a215868",
	} {
		crlf := strings.ReplaceAll(message, "\n", "\r\n") + "\r\n"
		for _, m := range []string{message, crlf} {
			t.Run(m, func(t *testing.T) { expect(t, m, Default, "") })
		}
	}
}

// baloo: test_commit.py test_rejects_a_subject_of_another_shape
func TestCheck_RejectsASubjectOfAnotherShape(t *testing.T) {
	for _, message := range []string{
		"add login", "fix bug: x", "feat:add login", "feat:  x", "feat(): x", "feat( ): x",
		"feat (a): x", "feat: ", "Merge sort for the index",
	} {
		t.Run(message, func(t *testing.T) {
			expect(t, message, Default, `"`+message+"\" is not `type(scope): description`")
		})
	}
}

// commitlint's config-conventional error rules.
// baloo: test_commit.py test_rejects_what_commitlint_does
func TestCheck_RejectsWhatCommitlintDoes(t *testing.T) {
	x, y := strings.Repeat("x", 34), strings.Repeat("y", 40)
	for _, c := range []struct{ message, want string }{
		{"wip: any type", `"wip" ` + notOneOf},
		{"i18n: digits in the type", `"i18n" ` + notOneOf},
		{"Feat: any case", `"Feat" ` + notOneOf},
		{"fix: Add Thing", `"Add Thing" starts with an uppercase letter`},
		{"fix: API timeout", `"API timeout" starts with an uppercase letter`},
		{"fix: add thing.", `"add thing." ends with a full stop`},
		{"fix: add thing ", `"fix: add thing " has trailing spaces`},
		{"feat: x\nBody.", `leave a blank line between "feat: x" and the body`},
		{"feat: " + strings.Repeat("x", 95), `"feat: ` + x + `…" is 101 characters, over 100`},
		{"fix: x\n\n" + strings.Repeat("y", 101), `"` + y + `…" is 101 characters, over 100`},
		{
			"fix: x\n\nRefs: " + strings.Repeat("1", 100),
			`"Refs: ` + strings.Repeat("1", 34) + `…" is 106 characters, over 100`,
		},
	} {
		t.Run(c.message, func(t *testing.T) { expect(t, c.message, Default, c.want) })
	}
}

// baloo: test_commit.py test_rejects_a_breaking_change_footer_of_another_shape
func TestCheck_RejectsABreakingChangeFooterOfAnotherShape(t *testing.T) {
	for _, footer := range []string{
		"Breaking-Change: x", "breaking-change: x", "BREAKING CHANGE #12", "BREAKING-CHANGE #12",
	} {
		t.Run(footer, func(t *testing.T) {
			expect(t, "feat: x\n\nRefs: #1\n"+footer, Default,
				`"`+footer+"\" is not `BREAKING CHANGE: description`")
		})
	}
	// Footers are the trailing paragraphs that open with one, across blank lines.
	expect(t, "feat: x\n\nBody.\n\nbreaking-change: y\n\nRefs: #1", Default,
		"\"breaking-change: y\" is not `BREAKING CHANGE: description`")
}

// baloo: test_commit.py test_takes_its_own_types_or_any
func TestCheck_TakesItsOwnTypesOrAny(t *testing.T) {
	own := Rules{Types: []string{"wip", "feat"}, MaxLength: 100}
	anyType := Rules{MaxLength: 100}
	expect(t, "wip: x", own, "")
	expect(t, "fix: x", own, `"fix" is not one of wip, feat`)
	expect(t, "wip: x", anyType, "")
	expect(t, "add x", anyType, "\"add x\" is not `type(scope): description`")
}

// baloo: test_commit.py test_takes_its_own_line_length_or_none
func TestCheck_TakesItsOwnLineLengthOrNone(t *testing.T) {
	at := func(n int) Rules { return Rules{Types: Types, MaxLength: n} }
	expect(t, "feat: "+strings.Repeat("x", 67), at(72),
		`"feat: `+strings.Repeat("x", 34)+`…" is 73 characters, over 72`)
	expect(t, "feat: x\n\n"+strings.Repeat("y", 73), at(72),
		`"`+strings.Repeat("y", 40)+`…" is 73 characters, over 72`)
	expect(t, "feat: "+strings.Repeat("x", 500), at(0), "")
}

// No test in `baloo`, which has no scopes: they match as commitlint's scope-enum matches them.
func TestCheck_TakesItsOwnScopesOrAny(t *testing.T) {
	own := Rules{Types: Types, Scopes: []string{"api", "ui", "a,b"}, MaxLength: 100}
	for _, message := range []string{
		"feat: no scope", "feat(api): x", "feat(api,ui): x", "feat(api, ui): x", "feat(api/ui): x",
		`feat(ui\api): x`, "feat(a,b): x",
	} {
		t.Run(message, func(t *testing.T) { expect(t, message, own, "") })
	}
	for _, c := range []struct{ message, want string }{
		{"feat(web): x", `"web" is not one of api, ui, a,b`},
		{"feat(API): x", `"API" is not one of api, ui, a,b`},
		{"feat(api,web): x", `"web" is not one of api, ui, a,b`},
		{"feat(api / ui): x", `"api " is not one of api, ui, a,b`},
	} {
		t.Run(c.message, func(t *testing.T) { expect(t, c.message, own, c.want) })
	}
	expect(t, "feat(web): x", Default, "")
}
