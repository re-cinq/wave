package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/recinq/wave/internal/forge"
	"github.com/spf13/cobra"
)

// MergeOptions holds options for the merge command.
type MergeOptions struct {
	All bool
	Yes bool
}

// NewMergeCmd creates the merge command.
func NewMergeCmd() *cobra.Command {
	var opts MergeOptions

	cmd := &cobra.Command{
		Use:   "merge <PR-URL-or-number>",
		Short: "Merge a pull request using forge CLI with API fallback",
		Long: `Merge a pull request by number or URL. Detects the forge type
(GitHub, GitLab, Gitea, Forgejo, Codeberg) and uses the appropriate CLI tool.
Falls back to raw API calls when CLI merge fails.

With --all, merges all open PRs that have approved reviews, oldest first.`,
		Example: `  wave merge 123
  wave merge https://github.com/owner/repo/pull/123
  wave merge --all
  wave merge --all --yes`,
		Args: func(cmd *cobra.Command, args []string) error {
			if opts.All {
				if len(args) > 0 {
					return fmt.Errorf("--all does not accept arguments")
				}
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("requires exactly 1 argument (PR number or URL)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fi, err := forge.DetectFromGitRemotes()
			if err != nil {
				return NewCLIError(CodeInternalError, "failed to detect forge: "+err.Error(), "Ensure you are in a git repository with remotes")
			}
			if fi.Type == forge.ForgeLocal || fi.Type == forge.ForgeUnknown {
				return NewCLIError(CodeInternalError, "no forge detected (type: "+string(fi.Type)+")", "Ensure this repo has a remote pointing to a supported forge")
			}

			if opts.All {
				return runMergeAll(fi, opts)
			}

			prNumber, err := parsePRInput(args[0], fi)
			if err != nil {
				return NewCLIError(CodeInvalidArgs, err.Error(), "Pass a PR number (123) or URL (https://github.com/owner/repo/pull/123)")
			}

			if err := mergePR(fi, prNumber); err != nil {
				return NewCLIError(CodeInternalError, fmt.Sprintf("failed to merge PR #%d: %s", prNumber, err), "Check PR status and permissions")
			}
			fmt.Fprintf(os.Stderr, "Merged %s #%d\n", fi.PRTerm, prNumber)
			return nil
		},
	}

	cmd.Flags().BoolVar(&opts.All, "all", false, "Merge all approved open PRs (oldest first)")
	cmd.Flags().BoolVar(&opts.Yes, "yes", false, "Skip confirmation prompt (use with --all)")

	return cmd
}

// prURLPattern matches forge PR/MR URLs and extracts the number.
var prURLPattern = regexp.MustCompile(`https?://[^/]+/[^/]+/[^/]+/(?:pull|merge_requests|pulls)/(\d+)`)

// parsePRInput extracts a PR number from a URL or plain number string.
func parsePRInput(input string, _ forge.ForgeInfo) (int, error) {
	input = strings.TrimSpace(input)

	// Try plain number first.
	if n, err := strconv.Atoi(input); err == nil && n > 0 {
		return n, nil
	}

	// Try URL pattern.
	m := prURLPattern.FindStringSubmatch(input)
	if m != nil {
		n, err := strconv.Atoi(m[1])
		if err == nil && n > 0 {
			return n, nil
		}
	}

	return 0, fmt.Errorf("cannot parse PR identifier %q", input)
}

// mergePR attempts to merge a PR using the forge CLI, falling back to raw API
// if the CLI merge fails.
func mergePR(fi forge.ForgeInfo, number int) error {
	err := mergePRViaCLI(fi, number)
	if err == nil {
		return nil
	}

	// CLI failed — try API fallback for forges that support it.
	apiErr := mergePRViaAPI(fi, number)
	if apiErr == nil {
		return nil
	}

	return fmt.Errorf("CLI merge failed: %w; API fallback also failed: %v", err, apiErr)
}

// mergePRViaCLI runs the appropriate forge CLI to merge a PR.
func mergePRViaCLI(fi forge.ForgeInfo, number int) error {
	numStr := strconv.Itoa(number)
	var cmd *exec.Cmd

	switch fi.Type {
	case forge.ForgeGitHub:
		cmd = exec.Command("gh", "pr", "merge", numStr, "--merge")
	case forge.ForgeGitLab:
		cmd = exec.Command("glab", "mr", "merge", numStr)
	case forge.ForgeGitea, forge.ForgeForgejo, forge.ForgeCodeberg:
		cmd = exec.Command("tea", "pulls", "merge", numStr)
	case forge.ForgeBitbucket:
		cmd = exec.Command("bb", "pr", "merge", numStr)
	default:
		return fmt.Errorf("unsupported forge type: %s", fi.Type)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// mergePRViaAPI attempts to merge a PR using the forge's REST API directly.
func mergePRViaAPI(fi forge.ForgeInfo, number int) error {
	token := forge.ResolveToken(fi.Type)
	if token == "" {
		return fmt.Errorf("no API token found for forge %s", fi.Type)
	}

	switch fi.Type {
	case forge.ForgeGitHub:
		return mergeGitHubAPI(fi, number, token)
	case forge.ForgeGitea, forge.ForgeForgejo, forge.ForgeCodeberg:
		return mergeGiteaAPI(fi, number, token)
	case forge.ForgeGitLab:
		return mergeGitLabAPI(fi, number, token)
	default:
		return fmt.Errorf("no API fallback for forge type: %s", fi.Type)
	}
}

// mergeGitHubAPI merges a PR via the GitHub REST API.
func mergeGitHubAPI(fi forge.ForgeInfo, number int, token string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d/merge", fi.Owner, fi.Repo, number)
	body := []byte(`{"merge_method":"merge"}`)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}
	return nil
}

// mergeGiteaAPI merges a PR via the Gitea/Forgejo REST API.
func mergeGiteaAPI(fi forge.ForgeInfo, number int, token string) error {
	url := fmt.Sprintf("https://%s/api/v1/repos/%s/%s/pulls/%d/merge", fi.Host, fi.Owner, fi.Repo, number)
	body := []byte(`{"Do":"merge"}`)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gitea API returned %d", resp.StatusCode)
	}
	return nil
}

// mergeGitLabAPI merges an MR via the GitLab REST API.
func mergeGitLabAPI(fi forge.ForgeInfo, number int, token string) error {
	// GitLab API uses url-encoded project path.
	projectPath := fi.Owner + "%2F" + fi.Repo
	host := fi.Host
	if host == "" {
		host = "gitlab.com"
	}
	url := fmt.Sprintf("https://%s/api/v4/projects/%s/merge_requests/%d/merge", host, projectPath, number)

	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitLab API returned %d", resp.StatusCode)
	}
	return nil
}

// ghPRListEntry represents a PR from `gh pr list --json`.
type ghPRListEntry struct {
	Number         int    `json:"number"`
	Title          string `json:"title"`
	ReviewDecision string `json:"reviewDecision"`
}

// runMergeAll merges all approved open PRs, oldest first.
func runMergeAll(fi forge.ForgeInfo, opts MergeOptions) error {
	prs, err := listApprovedPRs(fi)
	if err != nil {
		return NewCLIError(CodeInternalError, "failed to list PRs: "+err.Error(), "Ensure forge CLI is available and authenticated")
	}

	if len(prs) == 0 {
		fmt.Fprintf(os.Stderr, "No approved open %ss found\n", fi.PRTerm)
		return nil
	}

	// Sort by number ascending (oldest first).
	sort.Slice(prs, func(i, j int) bool {
		return prs[i].Number < prs[j].Number
	})

	// Print dry-run summary unless --yes.
	if !opts.Yes {
		fmt.Fprintf(os.Stderr, "Will merge %d approved %s(s):\n", len(prs), fi.PRTerm)
		for _, pr := range prs {
			fmt.Fprintf(os.Stderr, "  #%d  %s\n", pr.Number, pr.Title)
		}

		if !isTTY() {
			fmt.Fprintf(os.Stderr, "\nUse --yes to proceed. Stdin is not a TTY.\n")
			return nil
		}

		fmt.Fprintf(os.Stderr, "\nProceed? [y/N] ")
		var response string
		_, _ = fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Fprintf(os.Stderr, "Aborted\n")
			return nil
		}
	}

	merged := 0
	conflicting := 0
	for _, pr := range prs {
		fmt.Fprintf(os.Stderr, "Merging #%d %s... ", pr.Number, pr.Title)
		if err := mergePR(fi, pr.Number); err != nil {
			fmt.Fprintf(os.Stderr, "FAILED: %s\n", err)
			conflicting++
			continue
		}
		fmt.Fprintf(os.Stderr, "OK\n")
		merged++
	}

	fmt.Fprintf(os.Stderr, "\nMerged %d PRs, %d conflicting\n", merged, conflicting)
	return nil
}

// listApprovedPRs returns open PRs with approved reviews using the forge CLI.
func listApprovedPRs(fi forge.ForgeInfo) ([]ghPRListEntry, error) {
	switch fi.Type {
	case forge.ForgeGitHub:
		return listApprovedPRsGitHub()
	case forge.ForgeGitLab:
		return listApprovedPRsGitLab()
	case forge.ForgeGitea, forge.ForgeForgejo, forge.ForgeCodeberg:
		return listApprovedPRsGitea()
	default:
		return nil, fmt.Errorf("listing approved PRs not supported for forge: %s", fi.Type)
	}
}

// listApprovedPRsGitHub uses gh CLI to list approved PRs.
func listApprovedPRsGitHub() ([]ghPRListEntry, error) {
	out, err := exec.Command("gh", "pr", "list", "--json", "number,title,reviewDecision").Output()
	if err != nil {
		return nil, fmt.Errorf("gh pr list failed: %w", err)
	}

	var prs []ghPRListEntry
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil, fmt.Errorf("failed to parse gh pr list output: %w", err)
	}

	var approved []ghPRListEntry
	for _, pr := range prs {
		if pr.ReviewDecision == "APPROVED" {
			approved = append(approved, pr)
		}
	}
	return approved, nil
}

// listApprovedPRsGitLab uses glab CLI to list merged-ready MRs.
func listApprovedPRsGitLab() ([]ghPRListEntry, error) {
	out, err := exec.Command("glab", "mr", "list", "--state", "opened", "--json", "iid,title").Output()
	if err != nil {
		return nil, fmt.Errorf("glab mr list failed: %w", err)
	}

	var raw []struct {
		IID   int    `json:"iid"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse glab mr list output: %w", err)
	}

	var entries []ghPRListEntry
	for _, mr := range raw {
		entries = append(entries, ghPRListEntry{
			Number:         mr.IID,
			Title:          mr.Title,
			ReviewDecision: "APPROVED", // glab doesn't expose review state in list; include all
		})
	}
	return entries, nil
}

// listApprovedPRsGitea uses tea CLI to list open PRs.
func listApprovedPRsGitea() ([]ghPRListEntry, error) {
	out, err := exec.Command("tea", "pulls", "list", "--state", "open", "--output", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("tea pulls list failed: %w", err)
	}

	var raw []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse tea pulls list output: %w", err)
	}

	var entries []ghPRListEntry
	for _, pr := range raw {
		entries = append(entries, ghPRListEntry{
			Number:         pr.Number,
			Title:          pr.Title,
			ReviewDecision: "APPROVED", // tea doesn't expose review decision; include all
		})
	}
	return entries, nil
}
