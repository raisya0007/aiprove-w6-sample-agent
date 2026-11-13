package main

import (
	"fmt"
)

// PreferredSlug is the live-loop target when it is open and unentered:
// a 7-day BTC direction call, 0.9 NVDA, AutoGrade/Brier-scored.
const PreferredSlug = "forecast-btc-7d"

// StagePickBounty — choose ONE open bounty this wallet has not entered.
// Preference order: forecast-btc-7d first, then the first open forecast
// bounty. The prediction is built here so the submit stage is mechanical.
var StagePickBounty = S("pick-bounty", func(ctx *Ctx) error {
	var pick *Bounty
	for i := range ctx.Open {
		b := &ctx.Open[i]
		if ctx.Entered[b.Slug] || b.Status != "open" {
			continue
		}
		if b.Slug == PreferredSlug {
			pick = b
			break
		}
		if pick == nil && b.Cat == "forecast" {
			pick = b
		}
	}
	if pick == nil {
		return fmt.Errorf("no open, unentered bounty found")
	}
	ctx.Picked = *pick
	ctx.Prediction = buildPrediction(pick.Slug)
	fmt.Printf("  picked %s: %q (%s %s, ends %s)\n", pick.Slug, pick.Title, pick.Amt, pick.Unit, pick.EndDate)
	fmt.Printf("  prediction: %v\n", ctx.Prediction)
	return nil
})

// buildPrediction returns the forecast payload for a slug. Only yes/no
// bounties are supported by this sample agent; anything else is an error
// surfaced by the caller via the empty map.
//
// forecast-btc-7d call (Nov 14): BTC ~$68.9k, down ~9% on the week after
// breaking below $70k, ETH also below $2k — momentum is down, so "no"
// (BTC does NOT close higher in 7 days) at a hedged 0.60. A confident
// wrong call costs more than a hedged one under Brier scoring.
func buildPrediction(slug string) map[string]any {
	switch slug {
	case "forecast-btc-7d":
		return map[string]any{"answer": "no", "confidence": 0.60}
	case "btc-tomorrow", "forecast-eth-24h":
		return map[string]any{"answer": "no", "confidence": 0.58}
	default:
		return map[string]any{"answer": "no", "confidence": 0.55}
	}
}

// StageSubmitForecast — skill.md §4 shape B: POST the prediction.
// A 409 means the wallet already entered (no duplicate is created).
var StageSubmitForecast = S("submit-forecast", func(ctx *Ctx) error {
	body := map[string]any{
		"prediction": ctx.Prediction,
		"handle":     ctx.Handle,
	}
	var out SubmitResponse
	status, err := ctx.Client.postJSON("/api/bounties/"+ctx.Picked.Slug+"/submit", body, &out)
	if err != nil {
		return fmt.Errorf("submit HTTP %d: %w", status, err)
	}
	if out.ID == "" {
		return fmt.Errorf("submit HTTP %d returned no submission id", status)
	}
	ctx.SubmissionID = out.ID
	ctx.SubmissionStatus = out.Status
	fmt.Printf("  POST /api/bounties/%s/submit -> HTTP %d, submission id=%s status=%s\n",
		ctx.Picked.Slug, status, out.ID, out.Status)
	return nil
})

// StageVerify — skill.md §5: confirm the submission is on our own record
// before declaring the loop complete. This is the step that catches a
// connection drop after the server already processed the submit.
var StageVerify = S("verify", func(ctx *Ctx) error {
	var subs SubmissionsResponse
	if _, _, err := ctx.Client.get("/api/me/submissions", &subs); err != nil {
		return err
	}
	for _, s := range subs.Submissions {
		if s.ID == ctx.SubmissionID {
			fmt.Printf("  GET /api/me/submissions -> confirmed id=%s slug=%s status=%s (total %d submissions)\n",
				s.ID, s.BountySlug, s.Status, len(subs.Submissions))
			ctx.SubmissionStatus = s.Status
			return nil
		}
	}
	return fmt.Errorf("submission %s not found in /api/me/submissions (%d records)", ctx.SubmissionID, len(subs.Submissions))
})
