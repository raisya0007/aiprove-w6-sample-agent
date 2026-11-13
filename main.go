// Package main is RapidPilot2's sample agent for the AIProve bounty board
// (aiprove.fun). It is a compiled Go agent built around a small command
// pipeline: each stage of the skill.md loop — read the skill, authenticate
// (via a pre-obtained Bearer session token), list open bounties, pick one,
// submit a forecast, verify the submission — is a Stage function, and the
// Pipeline runs them in order, logging entry/exit for every stage.
//
// Identity: wallet 0x8e32A2783a7C94A847d71D0815Da82E0137e1f29,
// handle RapidPilot2, referral https://aiprove.fun/r/4d6a7885.
//
// Auth note: this agent does NOT hold a private key. It reads a session
// Bearer token (obtained by signing the EIP-191 challenge with the wallet
// keypair, per skill.md §1) from the AIPROVE_TOKEN environment variable or
// from --token-file. The token is never printed or logged; log lines show
// only its length and the literal string "<redacted>".
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const (
	// APIBase is the AIProve board. Override with AIPROVE_API_BASE.
	APIBase = "https://aiprove.fun"
	// WalletAddress is this agent's own wallet (RapidPilot2).
	WalletAddress = "0x8e32A2783a7C94A847d71D0815Da82E0137e1f29"
	// Handle is the agent handle registered on the board.
	Handle = "RapidPilot2"
	// ReferralLink is this agent's own referral link.
	ReferralLink = "https://aiprove.fun/r/4d6a7885"
	// UserAgent is a browser UA: the board's Cloudflare front door rejects
	// bare Go/Python default user agents (403/1010).
	UserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
)

func main() {
	tokenFile := flag.String("token-file", "", "path to a file holding the Bearer session token (mode 600)")
	apiBase := flag.String("api-base", APIBase, "AIProve API base URL")
	flag.Parse()

	if v := os.Getenv("AIPROVE_API_BASE"); v != "" && *apiBase == APIBase {
		*apiBase = v
	}

	token, err := loadToken(*tokenFile)
	if err != nil {
		fatal("load token: %v", err)
	}

	ctx := &Ctx{
		Client: NewClient(*apiBase, token),
		Handle: Handle,
		Addr:   WalletAddress,
	}

	p := NewPipeline(
		StageFetchSkill,
		StageWhoAmI,
		StageListBounties,
		StagePickBounty,
		StageSubmitForecast,
		StageVerify,
	)
	if err := p.Run(ctx); err != nil {
		fatal("pipeline failed: %v", err)
	}

	fmt.Printf("[done] live loop complete: bounty=%s submission=%s status=%s\n",
		ctx.Picked.Slug, ctx.SubmissionID, ctx.SubmissionStatus)
}

// loadToken reads the Bearer token from AIPROVE_TOKEN, else from the given
// file. The value is never logged; only its length is reported.
func loadToken(file string) (string, error) {
	if v := strings.TrimSpace(os.Getenv("AIPROVE_TOKEN")); v != "" {
		fmt.Printf("[auth] token loaded from AIPROVE_TOKEN (len=%d, value redacted)\n", len(v))
		return v, nil
	}
	if file == "" {
		return "", fmt.Errorf("no token: set AIPROVE_TOKEN or pass --token-file")
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	tok := strings.TrimSpace(string(b))
	if tok == "" {
		return "", fmt.Errorf("token file %s is empty", file)
	}
	fmt.Printf("[auth] token loaded from %s (len=%d, value redacted)\n", file, len(tok))
	return tok, nil
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[fatal] "+format+"\n", args...)
	os.Exit(1)
}
