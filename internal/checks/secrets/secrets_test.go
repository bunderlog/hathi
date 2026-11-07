package secrets

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/bunderlog/hathi/internal/checks"
	"github.com/bunderlog/hathi/internal/gittest"
)

// A fake Secret of each kind, built at run time, so this file holds none and passes the Check
// itself.
var fakes = []struct{ what, secret string }{
	{"a private key", "-----BEGIN RSA " + "PRIVATE KEY-----\\n" + body},
	{"a private key", "-----BEGIN " + "PRIVATE KEY-----\\n" + body},
	{"an AWS access key", "AKIA" + strings.Repeat("Z7Q2", 4)},
	// The last 6 characters are the base62 CRC32 of the 30 before them, worked out apart.
	{"a GitHub token", "ghp_" + "Z7q2W9x4K1m8R3t6Y5u0P2s7D4f9G1" + "0F19eW"},
	{"a GitHub token", "github_pat_" + strings.Repeat("A_b1", 16)},
	// Cloudflare's last 8 characters are the hex CRC32 of the 40 before them, worked out apart.
	{"a Cloudflare token", "cfut_" + "Q8wE2rT6yU1iO5pA9sD3fG7hJ4kL0zX2cV6bN8mQ" + "a25c625c"},
	{"a Cloudflare token", "cfk_" + "M3nB7vC1xZ5lK9jH2gF6dS0aP4oI8uY3tR7eW1qA" + "9e0665fc"},
	// Atlassian's last 8 characters are the hex CRC32 of all before them, worked out apart.
	{"an Atlassian API token", "ATATT3" + atlassian + "B0BA09DD"},
	{"a GitLab token", "glpat-" + strings.Repeat("a1B2c", 4)},
	{"a Slack token", "xoxb-" + "1234567890-abc"},
	{"a Slack token", "xapp-" + "1-A0B1C2D3E4F-123"},
	{"a Slack webhook URL", "https://hooks.slack" + ".com/services/T0B1C2D3E/B4F5G6H7I/" +
		strings.Repeat("a1B2", 6)},
	{"a Stripe live key", "sk_live_" + strings.Repeat("a1B2", 6)},
	{"an Anthropic API key", "sk-ant-" + "api03-" + strings.Repeat("a1B2c3", 8)},
	{"an OpenAI API key", "sk-proj-" + strings.Repeat("a1B2", 6) + "T3Blbk" + "FJ" +
		strings.Repeat("c3D4", 6)},
	{"a Google API key", "AIza" + strings.Repeat("Sy", 17) + "x"},
	{"an npm token", "npm_" + "Pq3Rs7Tu1Vw5Xy9Za2Bc6De0Fg4Hi8" + "2Cbhyj"}, // checksum as GitHub's
	{"a PyPI token", "pypi-AgEIcHlw" + "aS5vcmc" + strings.Repeat("a1B2c3", 9)},
	{"a Docker Hub token", "dckr_pat_" + strings.Repeat("a1B2c3", 5)},
	{"a Hugging Face token", "hf_" + strings.Repeat("qWeRtYzU", 5)[:34]},
	{"a SendGrid API key", "SG." + strings.Repeat("a1B2c3D", 4)[:22] + "." +
		strings.Repeat("e5F6g7H", 7)[:43]},
	{"a DigitalOcean token", "dop_v1_" + strings.Repeat("a1b2c3d4", 8)},
	{"a Shopify token", "shpat_" + strings.Repeat("a1b2c3d4", 4)},
	{"a Vault token", "hvs." + strings.Repeat("a1B2c3", 16)},
	{"an age secret key", "AGE-SECRET-" + "KEY-1" +
		strings.Repeat("QPZRY9X8GF2TVDW0S3JN54KHCE6MUA7L", 2)[:58]},
	{"a Telegram bot token", "123456789" + ":A" + strings.Repeat("a1B2c3D4e5", 4)[:34]},
	{"a private key", "-----BEGIN PGP " + "PRIVATE KEY BLOCK-----\\n" + body},
	{"a private key", "LS0tLS1CRUdJTiBPUEVOU1NI" +
		"IFBSSVZBVEUgS0VZLS0tLS0KYjNCbGJuTnphQzFyWlhrdGRqRUFBQUFBQkc1dmJtVUFBQUFF" +
		"Ym05dVpRQUFBQUFBQUFBQkFBQUFNd0FBQUF0emMyZ3RaVwotLS0tLUVORCBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0K"},
	{"an OpenRouter API key", "sk-or-" + "v1-" + strings.Repeat("a1b2c3d4", 8)},
	{"a Groq API key", "gsk_" + strings.Repeat("a1B2", 13)},
	{"an xAI API key", "xai-" + strings.Repeat("a1B2", 20)},
	{"a Replicate token", "r8_" + strings.Repeat("a1B2", 10)[:37]},
	{"a Perplexity API key", "pplx-" + strings.Repeat("a1B2", 12)},
	{"a LangSmith API key", "lsv2_pt_" + strings.Repeat("a1b2", 8) + "_" + "c3d4e5f6a7"},
	{"a Pinecone API key", "pcsk_" + "a1B2c_" + strings.Repeat("a1B2c3D4e", 7)},
	{"a webhook signing secret", "whsec_" + strings.Repeat("a1B2", 8)},
	{"a Supabase token", "sbp_" + strings.Repeat("a1b2c3d4", 5)},
	{"a Sentry token", "sntryu_" + strings.Repeat("a1b2c3d4", 8)},
	{"a Fly.io token", "fo1_" + strings.Repeat("a1B2", 11)[:43]},
	{"a Linear API key", "lin_api_" + strings.Repeat("a1B2", 10)},
	{"a Notion token", "ntn_" + "48151623420" + strings.Repeat("a1B2", 8) + "c3D"},
	{"a Figma token", "figd_" + strings.Repeat("a1B2", 10)},
	{"a Doppler token", "dp.pt." + strings.Repeat("a1B2", 11)[:43]},
	{"a 1Password token", "ops_eyJ" + strings.Repeat("a1B2", 64)},
	{"a Heroku API key", "HRKU-AA" + strings.Repeat("a1B2", 15)[:58]},
	{"a Terraform Cloud token", "a1b2c3d" + "4e5f6a7" + ".atlasv1." + strings.Repeat("c3d4", 16)},
	{"a Pulumi token", "pul-" + strings.Repeat("a1b2c3d4", 5)},
	{"a Postman API key", "PMAK-" + strings.Repeat("a1b2", 6) + "-" + strings.Repeat("c3d4", 9)[:34]},
	{"a New Relic key", "NRAK-" + strings.Repeat("A1B2", 7)[:27]},
	{"a Databricks token", "dapi" + strings.Repeat("a1b2c3d4", 4)},
	{"a Discord webhook URL", "https://discord.com/api/" + "webhooks/123456789012345678/" +
		strings.Repeat("a1B2", 17)},
	{"a Microsoft Teams webhook URL", "https://acme.webhook.office.com/" + "webhookb2/" + uuid +
		"@" + uuid + "/IncomingWebhook/" + strings.Repeat("a1b2", 8) + "/" + uuid},
	{"a Grafana token", "glsa_" + strings.Repeat("a1B2", 8) + "_" + "c3d4e5f6"},
	{"a Grafana token", "glc_" + strings.Repeat("a1B2", 10)},
	{"a Grafana token", "eyJrIjoi" + strings.Repeat("a1B2", 18)},
	{"a Tailscale key", "tskey-api-" + "kQx7Ld2CNTRL-" + strings.Repeat("a1B2", 8)},
	{"a CircleCI token", "CCIPAT_" + strings.Repeat("a1B2", 6)[:22] + "_" + strings.Repeat("a1b2c3d4", 5)},
	{"a Buildkite token", "bkua_" + strings.Repeat("a1b2", 10)},
	{"a Docker Swarm token", "SWMTKN-1-" + strings.Repeat("a1b2c", 10) + "-" + strings.Repeat("d3e4f", 5)},
	{"a Docker Swarm token", "SWMKEY-1-" + strings.Repeat("a1B2", 11)[:43]},
	{"a Doppler token", "dp.st.dev." + strings.Repeat("a1B2", 11)[:43]},
	{"a Dynatrace token", "dt0c01." + strings.Repeat("A1B2", 6) + "." + strings.Repeat("C3D4", 16)},
	{"a RubyGems key", "rubygems_" + strings.Repeat("a1b2c3", 8)},
	{"a Clojars token", "CLOJARS_" + strings.Repeat("a1b2c3", 10)},
	{"a PostHog personal API key", "phx_" + strings.Repeat("a1B2", 11)[:43]},
	{"a Sourcegraph token", "sgp_" + strings.Repeat("a1b2", 4) + "_" + strings.Repeat("c3d4", 10)},
	{"a Mapbox secret token", "sk.eyJ" + strings.Repeat("a1B2", 22)},
	{"a PlanetScale token", "pscale_tkn_" + strings.Repeat("a1B2", 10)},
	{"a Prefect key", "pnu_" + strings.Repeat("a1B2", 9)},
	{"a ReadMe API key", "rdme_" + strings.Repeat("a1b2c3d4e5", 7)},
	{"a Contentful token", "CFPAT-" + strings.Repeat("a1B2", 11)[:43]},
	{"a Typeform token", "tfp_" + strings.Repeat("a1B2", 11)},
	{"a Resend API key", "re_" + "Ab3Cd5Ef" + "_" + strings.Repeat("Gh7Jk9Mn", 3)},
	{"a Shippo live token", "shippo_live_" + strings.Repeat("a1b2c3d4", 5)},
	{"a JWT", "eyJ" + "hbGciOiJIUzI1NiJ9.eyJ" + "zdWIiOiIxMjM0NTY3ODkwIn0." + strings.Repeat("Sf", 20)},
	{"a password in a URL", "ftp://dm:" + "hunter2-Pass" + "@10.0.0.1/devices"},
	{"a password in a URL", "redis://:" + "hunter2" + "@cache:6379"},
	{"a password in a URL", "cfg://u:" + "hunter2" + "@h"},
}

// An Atlassian API token's body between its prefix and its checksum.
var atlassian = strings.Repeat("Kp8Wq2Lr6Tz1Vb5Xc9Md3Gf7Js0Hn4", 6)[:177] + "="

// A UUID, as webhook URLs hold them.
const uuid = "1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d"

// A private key's body, base64 as PEM has it.
var body = strings.Repeat("MIIEvQIBADANBgkqhkiG9w0B", 3)

var aws = fakes[2].secret

const unmarked = "an Env file; keep it out of Git, or put hathi:allow-secret on its first line"

func find(t *testing.T, dir string) []checks.Finding {
	t.Helper()
	found, err := Check.Find(checks.Input{Changes: checks.Changes{Repo: checks.Repo{Dir: dir}}})
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func expect(t *testing.T, got []checks.Finding, want ...checks.Finding) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// baloo: test_secrets.py test_finds_each_kind
func TestCheck_FindsEachKind(t *testing.T) {
	for _, f := range fakes {
		t.Run(f.what+" "+f.secret[:12], func(t *testing.T) {
			dir := gittest.Repo(t)
			gittest.Stage(t, dir, map[string]string{"src/a.go": "x := 1\nkey := \"" + f.secret + "\"\n"})
			expect(t, find(t, dir), checks.Finding{File: "src/a.go", Line: 2, Reason: f.what})
		})
	}
}

// baloo: test_secrets.py test_passes_what_is_no_secret
func TestCheck_PassesWhatIsNoSecret(t *testing.T) {
	dir := gittest.Repo(t)
	lines := []string{
		"sk-learn is a library",
		"re := regexp.MustCompile(`AKIA[0-9A-Z]{16}`)",
		"-----BEGIN PUBLIC KEY-----",
		"-----BEGIN " + "PRIVATE KEY-----",
		"ghp_short",
		// Placeholders, and a match without its kind's marker.
		"AKIA" + "IOSFODNN7EXAMPLE",
		"ghp_" + strings.Repeat("x", 36),
		"ghp_" + "Hn4Kp8Wq2Lr6Tz1Vb5Xc9Md3Gf7Js0" + "09TywX",              // its checksum is 09TywE
		"cfut_" + "Q8wE2rT6yU1iO5pA9sD3fG7hJ4kL0zX2cV6bN8mQ" + "a25c625d", // a25c625c
		"npm_" + strings.Repeat("0", 36),
		"npm_" + "Pq3Rs7Tu1Vw5Xy9Za2Bc6De0Fg4Hi8" + "2Cbhyk", // its checksum is 2Cbhyj
		"ATATT3" + atlassian + "B0BA09DE",                    // B0BA09DD
		"sk-proj-" + strings.Repeat("a1B2", 10),
		"AKIA" + strings.Repeat("Z7Q8", 4),
		// A certificate in base64, as a kubeconfig's certificate-authority-data holds it.
		"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJzekNDQVZtZ0F3SUJBZ0lVWTJWeWRHbG1hV05oZEdVZ1ptOXlJ" +
			"SFJsYzNSek1Bb0dDQ3FHU000OUJBTUMKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=",
		"sk_test_" + strings.Repeat("a1B2", 6),
		"eyJhbGciOiJIUzI1NiJ9 alone",
		"postgres://user:${DB_PASSWORD}@db/app",
		"https://x:$TOKEN@github.com/org/repo",
		"https://<user>:<password>@host",
		"https://user:****@host",
		"https://user@host:8080/path",
		"http://localhost:8080/a@b",
		"git@github.com:org/repo.git",
	}
	gittest.Stage(t, dir, map[string]string{"a.go": strings.Join(lines, "\n")})
	expect(t, find(t, dir))
}

// A private key counts by its body: on the lines after its BEGIN line, PEM's headers between, or
// after it on the same line, as a string in code holds it; its BEGIN line alone passes, and so does
// one a long name follows.
func TestCheck_FindsAPrivateKeyByItsBody(t *testing.T) {
	dir := gittest.Repo(t)
	begin := "-----BEGIN " + "PRIVATE KEY-----"
	gittest.Stage(t, dir, map[string]string{
		"key.pem": begin + "\n" + body + "\n" + body + "\n-----END PRIVATE KEY-----\n",
		"old.pem": "-----BEGIN RSA " + "PRIVATE KEY-----\nProc-Type: 4,ENCRYPTED\n" +
			"DEK-Info: AES-128-CBC,0A1B\n\n" + body + "\n",
		"pgp.asc":  "-----BEGIN PGP " + "PRIVATE KEY BLOCK-----\nVersion: GnuPG v2\n\n" + body + "\n",
		"parse.go": "const begin = \"" + begin + "\"\nfunc parse() {}\n",
		"read.go": "const begin = \"" + begin + "\"\n" +
			"func readThePrivateKeyFromTheGivenBlockAndReturnIt() {}\n",
	})
	expect(t, find(t, dir),
		checks.Finding{File: "key.pem", Line: 1, Reason: "a private key"},
		checks.Finding{File: "old.pem", Line: 1, Reason: "a private key"},
		checks.Finding{File: "pgp.asc", Line: 1, Reason: "a private key"})
}

// No output holds the Secret (PRD, criterion 10): a Finding has its place and its kind only.
// baloo: test_secrets.py test_never_reports_the_secret
func TestCheck_NeverReportsTheSecret(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Stage(t, dir, map[string]string{"a.go": aws})
	for _, f := range find(t, dir) {
		if strings.Contains(f.File+f.Reason, aws) {
			t.Errorf("%v holds the Secret", f)
		}
	}
}

// baloo: test_secrets.py test_passes_a_line_marked_allow_secret
func TestCheck_PassesALineMarkedAllowSecret(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Stage(t, dir, map[string]string{
		"a.go": aws + " // hathi:allow-secret\n" + aws + "\n" + aws + " // baloo:allow-secret\n",
	})
	expect(t, find(t, dir), checks.Finding{File: "a.go", Line: 2, Reason: "an AWS access key"})
}

// gitleaks' mark passes too, on a line and on an Env file's first line.
func TestCheck_PassesWhatGitleaksAllowMarks(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Stage(t, dir, map[string]string{
		"a.go": aws + " // gitleaks:allow\n",
		".env": "# gitleaks:allow\nPORT=3000\n",
	})
	expect(t, find(t, dir))
}

// Only lines the staged changes add count, by their line in the staged file.
// baloo: test_secrets.py test_finds_only_added_lines
func TestCheck_FindsOnlyAddedLines(t *testing.T) {
	dir := gittest.Repo(t)
	lines := strings.Split(strings.Repeat("ok\n", 12), "\n")
	lines[2] = aws
	gittest.Stage(t, dir, map[string]string{"a.go": strings.Join(lines, "\n")})
	gittest.Commit(t, dir, "x")
	lines[2] = "ok"
	lines[10] = "++ " + aws // an added line that looks like a diff's header
	gittest.Stage(t, dir, map[string]string{"a.go": strings.Join(lines, "\n")})
	gittest.Write(t, dir, "unstaged.go", aws)
	expect(t, find(t, dir), checks.Finding{File: "a.go", Line: 11, Reason: "an AWS access key"})
	gittest.Commit(t, dir, "y")
	gittest.Git(t, dir, "rm", "-q", "a.go")
	expect(t, find(t, dir))
}

// baloo: test_secrets.py test_finds_env_files_but_not_their_templates
func TestCheck_FindsEnvFilesButNotTheirTemplates(t *testing.T) {
	dir := gittest.Repo(t)
	files := map[string]string{}
	for _, path := range []string{
		".env", "app/.env.local", "prod.env", ".envrc", ".env.example", "a.env.sample", "env.go",
	} {
		files[path] = ""
	}
	gittest.Stage(t, dir, files)
	var want []checks.Finding
	for _, path := range []string{".env", ".envrc", "app/.env.local", "prod.env"} {
		want = append(want, checks.Finding{File: path, Reason: unmarked})
	}
	expect(t, find(t, dir), want...)
}

// An Env file marked on its first line passes, and its lines are still looked through.
// baloo: test_secrets.py test_passes_an_env_file_marked_on_its_first_line
func TestCheck_PassesAnEnvFileMarkedOnItsFirstLine(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Stage(t, dir, map[string]string{
		".env":  "# public defaults, hathi:allow-secret\nPORT=3000\n",
		"a.env": "# baloo:allow-secret\nPORT=3000\n",
		"b.env": "PORT=3000\n# hathi:allow-secret\n",
	})
	expect(t, find(t, dir), checks.Finding{File: "b.env", Reason: unmarked})
	gittest.Stage(t, dir, map[string]string{".env": "# hathi:allow-secret\nKEY=" + aws + "\n"})
	found := find(t, dir)
	if !slices.Contains(found, checks.Finding{File: ".env", Line: 2, Reason: "an AWS access key"}) {
		t.Errorf("got %v, want .env:2 among them", found)
	}
}

// The Check's own files hold no Secret, so a repo that commits them passes: `hathi`'s files here,
// where `baloo`'s test stages `baloo`'s.
// baloo: test_secrets.py test_passes_its_own_files
func TestCheck_PassesItsOwnFiles(t *testing.T) {
	dir := gittest.Repo(t)
	files := map[string]string{}
	for _, name := range []string{"secrets.go", "secrets_test.go"} {
		text, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		files[filepath.Base(name)] = string(text)
	}
	gittest.Stage(t, dir, files)
	expect(t, find(t, dir))
}

// baloo: test_secrets.py test_fails_outside_a_repo
func TestCheck_FailsOutsideARepo(t *testing.T) {
	gittest.Isolate(t)
	t.Setenv("GIT_CEILING_DIRECTORIES", os.TempDir())
	_, err := Check.Find(checks.Input{Changes: checks.Changes{Repo: checks.Repo{Dir: t.TempDir()}}})
	if err == nil {
		t.Error("got no error outside a repo")
	}
}

// baloo: test_secrets.py test_masks_each_secret_and_nothing_else — left out: `hathi` has no
// mask_secrets, which only `baloo`'s Hooks use, to keep Secrets out of Claude's context.

// A path with a space, or one Git quotes, is named as it is.
func TestCheck_NamesAPathAsItIs(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Stage(t, dir, map[string]string{"a b.txt": aws + "\n", `a"b.txt`: aws + "\n"})
	expect(t, find(t, dir),
		checks.Finding{File: "a b.txt", Line: 1, Reason: "an AWS access key"},
		checks.Finding{File: `a"b.txt`, Line: 1, Reason: "an AWS access key"})
}
