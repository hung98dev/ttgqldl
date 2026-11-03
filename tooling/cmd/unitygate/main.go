// Command unitygate decides whether a manual Unity acceptance run may proceed.
//
// It is the single, testable implementation of the acceptance preconditions:
// every input is supplied explicitly so the rules can be exercised without a
// live repository. The workflow calls it; it never inspects the network itself.
//
// Rules (all must hold, fail-closed):
//  1. the owner-enabled runner switch is "true"
//  2. the protected environment has been confirmed by the owner
//  3. the workflow ref is refs/heads/main
//  4. the repository identity matches the expected repository
//  5. the requested SHA is a full 40-character lowercase hex commit id
//  6. the checked-out SHA equals the requested SHA
//  7. the freshly fetched origin/main tip equals the checked-out SHA
//
// Rule 7 is exact equality, not ancestry: a stale ancestor must never be
// accepted as the current main revision.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"time"
)

var shaPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Input is everything the decision depends on.
type Input struct {
	RunnerEnabled      string `json:"runnerEnabled"`
	EnvironmentReady   string `json:"environmentReady"`
	Ref                string `json:"ref"`
	Repository         string `json:"repository"`
	ExpectedRepository string `json:"expectedRepository"`
	RequestedSHA       string `json:"requestedSha"`
	CheckoutSHA        string `json:"checkoutSha"`
	RemoteMainSHA      string `json:"remoteMainSha"`
}

// Result is the machine-readable verdict.
type Result struct {
	OK        bool     `json:"ok"`
	SHA       string   `json:"sha,omitempty"`
	Reasons   []string `json:"reasons,omitempty"`
	CheckedAt string   `json:"checkedAt"`
}

func evaluate(in Input) Result {
	reasons := make([]string, 0, 8)

	if in.RunnerEnabled != "true" {
		reasons = append(reasons, "TTG_UNITY_RUNNER_ENABLED is not 'true': no licensed Unity runner is authorised for this repository")
	}
	if in.EnvironmentReady != "true" {
		reasons = append(reasons, "TTG_UNITY_ENV_CONFIRMED is not 'true': the protected 'unity-licensed' environment has not been confirmed by the owner")
	}
	if in.Ref != "refs/heads/main" {
		reasons = append(reasons, fmt.Sprintf("workflow ref %q is not refs/heads/main; only main can be accepted", in.Ref))
	}
	if in.ExpectedRepository == "" {
		reasons = append(reasons, "expected repository is empty")
	} else if in.Repository != in.ExpectedRepository {
		reasons = append(reasons, fmt.Sprintf("repository %q does not match expected %q", in.Repository, in.ExpectedRepository))
	}
	if !shaPattern.MatchString(in.RequestedSHA) {
		reasons = append(reasons, fmt.Sprintf("requested SHA %q is not a full 40-character lowercase hex commit id", in.RequestedSHA))
	}
	if in.RequestedSHA != in.CheckoutSHA {
		reasons = append(reasons, fmt.Sprintf("checked-out SHA %q is not the requested SHA %q", in.CheckoutSHA, in.RequestedSHA))
	}
	if in.CheckoutSHA != in.RemoteMainSHA {
		reasons = append(reasons,
			fmt.Sprintf("checked-out SHA %q is not the current origin/main tip %q (stale ancestors are rejected)", in.CheckoutSHA, in.RemoteMainSHA))
	}

	result := Result{
		OK:        len(reasons) == 0,
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
		Reasons:   reasons,
	}
	if result.OK {
		result.SHA = in.CheckoutSHA
	}
	return result
}

func main() {
	var in Input
	flag.StringVar(&in.RunnerEnabled, "runner-enabled", "", "value of the TTG_UNITY_RUNNER_ENABLED repository variable")
	flag.StringVar(&in.EnvironmentReady, "environment-ready", "", "value of the TTG_UNITY_ENV_CONFIRMED repository variable")
	flag.StringVar(&in.Ref, "ref", "", "GITHUB_REF of the workflow run")
	flag.StringVar(&in.Repository, "repository", "", "GITHUB_REPOSITORY of the workflow run")
	flag.StringVar(&in.ExpectedRepository, "expected-repository", "", "repository that may be accepted")
	flag.StringVar(&in.RequestedSHA, "requested-sha", "", "SHA typed by the operator")
	flag.StringVar(&in.CheckoutSHA, "checkout-sha", "", "SHA actually checked out")
	flag.StringVar(&in.RemoteMainSHA, "remote-main-sha", "", "freshly fetched origin/main tip")
	asJSON := flag.Bool("json", false, "emit the result as JSON")
	flag.Parse()

	result := evaluate(in)

	if *asJSON {
		payload, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, "unitygate:", err)
			os.Exit(2)
		}
		fmt.Println(string(payload))
	} else {
		if result.OK {
			fmt.Printf("unitygate: accepted revision %s\n", result.SHA)
		} else {
			for _, reason := range result.Reasons {
				fmt.Fprintf(os.Stderr, "unitygate: %s\n", reason)
			}
		}
	}

	if !result.OK {
		os.Exit(1)
	}
}
