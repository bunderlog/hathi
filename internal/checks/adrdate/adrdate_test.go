package adrdate

import (
	"os"
	"slices"
	"testing"

	"github.com/bunderlog/hathi/internal/checks"
	"github.com/bunderlog/hathi/internal/gittest"
)

const today = "2026-10-01"

// adr is an ADR dated `date` with the decision `decision`.
func adr(date, decision string) string {
	return "# Billing\n\nDate: " + date + "\n\n" + decision + "\n"
}

// repo is a repo with the ADR billing.md dated 2026-09-30, committed.
func repo(t *testing.T) string {
	dir := gittest.Repo(t)
	gittest.Stage(t, dir, map[string]string{".about/adr/billing.md": adr("2026-09-30", "Invoices are monthly.")})
	gittest.Commit(t, dir, "docs: adr")
	return dir
}

func find(dir string) ([]checks.Finding, error) {
	return Check(today).Find(checks.Input{Changes: checks.Changes{Repo: checks.Repo{Dir: dir}}})
}

func expect(t *testing.T, dir string, want ...checks.Finding) {
	t.Helper()
	got, err := find(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// baloo: test_adr.py test_finds_an_adr_changed_under_an_old_date
func TestCheck_FindsAnADRChangedUnderAnOldDate(t *testing.T) {
	dir := repo(t)
	gittest.Stage(t, dir, map[string]string{".about/adr/billing.md": adr("2026-09-30", "Invoices are weekly.")})
	expect(t, dir, checks.Finding{
		File: ".about/adr/billing.md", Line: 3,
		Reason: "changed, but its Date is 2026-09-30; set it to 2026-10-01",
	})
}

// An ADR with CRLF line endings is read as with LF.
// baloo: test_adr.py test_reads_crlf
func TestCheck_ReadsCRLF(t *testing.T) {
	dir := repo(t)
	gittest.Stage(t, dir, map[string]string{
		".about/adr/billing.md": "# Billing\r\n\r\nDate: " + today + "\r\n\r\nWeekly.\r\n",
	})
	expect(t, dir)
}

// An ADR changed again on the day of its Date passes as well.
// baloo: test_adr.py test_passes_an_adr_dated_today
func TestCheck_PassesAnADRDatedToday(t *testing.T) {
	dir := repo(t)
	gittest.Stage(t, dir, map[string]string{".about/adr/billing.md": adr(today, "Invoices are weekly.")})
	expect(t, dir)
	gittest.Commit(t, dir, "docs: weekly")
	gittest.Stage(t, dir, map[string]string{".about/adr/billing.md": adr(today, "Invoices are daily.")})
	expect(t, dir)
}

// baloo: test_adr.py test_finds_an_adr_without_a_date
func TestCheck_FindsAnADRWithoutADate(t *testing.T) {
	dir := repo(t)
	gittest.Stage(t, dir, map[string]string{".about/adr/billing.md": "# Billing\n\nInvoices are weekly.\n"})
	expect(t, dir, checks.Finding{
		File:   ".about/adr/billing.md",
		Reason: "has no Date line; add Date: 2026-10-01 under its title",
	})
}

// A new ADR, a deleted or renamed one, and a file beside the ADRs pass.
// baloo: test_adr.py test_passes_what_is_not_a_changed_adr
func TestCheck_PassesWhatIsNotAChangedADR(t *testing.T) {
	dir := repo(t)
	gittest.Stage(t, dir, map[string]string{
		".about/adr/tax.md":    adr("2026-09-30", "Prices include tax."),
		".about/adr/notes.txt": "Date: 2026-09-30\n",
		".about/glossary.md":   "Date: 2026-09-30\n",
		"docs/adr/billing.md":  adr("2026-09-30", "Elsewhere."),
	})
	expect(t, dir)
	gittest.Commit(t, dir, "docs: more")
	gittest.Stage(t, dir, map[string]string{
		".about/adr/notes.txt": "Date: 2026-09-29\n",
		".about/glossary.md":   "Date: 2026-09-29\n",
		"docs/adr/billing.md":  adr("2026-09-30", "Still elsewhere."),
	})
	expect(t, dir)
	gittest.Commit(t, dir, "docs: beside")
	gittest.Git(t, dir, "mv", ".about/adr/billing.md", ".about/adr/invoices.md")
	gittest.Git(t, dir, "rm", "-q", ".about/adr/tax.md")
	expect(t, dir)
}

// A commit in a range is held to its own committer date, not to today.
func TestCheck_HoldsACommitToItsCommitterDate(t *testing.T) {
	dir := repo(t)
	gittest.Stage(t, dir, map[string]string{".about/adr/billing.md": adr("2026-09-30", "Invoices are weekly.")})
	t.Setenv("GIT_COMMITTER_DATE", "2026-09-30T12:00:00Z")
	stale := gittest.Commit(t, dir, "docs: weekly")
	gittest.Stage(t, dir, map[string]string{".about/adr/billing.md": adr("2026-09-30", "Invoices are daily.")})
	t.Setenv("GIT_COMMITTER_DATE", "2026-10-02T12:00:00Z")
	later := gittest.Commit(t, dir, "docs: daily")
	for commit, want := range map[string][]checks.Finding{
		stale: nil,
		later: {{File: ".about/adr/billing.md", Line: 3,
			Reason: "changed, but its Date is 2026-09-30; set it to 2026-10-02"}},
	} {
		got, err := Check(today).Find(checks.Input{
			Changes: checks.Changes{Repo: checks.Repo{Dir: dir}, Commit: commit},
		})
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("%s: got %v and %v, want %v", commit, got, err, want)
		}
	}
}

// baloo: test_adr.py test_fails_outside_a_repo
func TestCheck_FailsOutsideARepo(t *testing.T) {
	gittest.Isolate(t)
	t.Setenv("GIT_CEILING_DIRECTORIES", os.TempDir())
	if _, err := find(t.TempDir()); err == nil {
		t.Error("got no error outside a repo")
	}
}
