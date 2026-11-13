package main

import (
	"fmt"
	"strings"
	"time"
)

// Ctx carries the shared state threaded through the pipeline stages.
type Ctx struct {
	Client           *Client
	Handle           string
	Addr             string
	SkillBytes       int
	Me               MeResponse
	Open             []Bounty
	Entered          map[string]bool
	Picked           Bounty
	Prediction       map[string]any
	SubmissionID     string
	SubmissionStatus string
}

// Stage is one step of the skill.md loop. It logs its own entry and exit;
// the pipeline adds timing and aborts on the first error.
type Stage struct {
	Name string
	Run  func(*Ctx) error
}

func S(name string, fn func(*Ctx) error) Stage { return Stage{Name: name, Run: fn} }

// Pipeline is the ordered list of stages. Small on purpose: the whole
// agent is "read skill → prove identity → list → pick → submit → verify".
type Pipeline struct{ stages []Stage }

func NewPipeline(stages ...Stage) *Pipeline { return &Pipeline{stages: stages} }

func (p *Pipeline) Run(ctx *Ctx) error {
	for _, st := range p.stages {
		start := time.Now()
		fmt.Printf("[stage] >>> %s\n", st.Name)
		if err := st.Run(ctx); err != nil {
			fmt.Printf("[stage] !!! %s failed after %s: %v\n", st.Name, time.Since(start).Round(time.Millisecond), err)
			return fmt.Errorf("stage %s: %w", st.Name, err)
		}
		fmt.Printf("[stage] <<< %s ok (%s)\n", st.Name, time.Since(start).Round(time.Millisecond))
	}
	return nil
}

// StageFetchSkill — skill.md §0: read the skill doc and confirm it is the
// real board doc (challenge endpoint + submit shape present).
var StageFetchSkill = S("fetch-skill", func(ctx *Ctx) error {
	body, status, err := ctx.Client.getPublic("/skill.md")
	if err != nil {
		return err
	}
	text := string(body)
	if !strings.Contains(text, "/api/auth/challenge") || !strings.Contains(text, "/api/bounties") {
		return fmt.Errorf("skill.md does not look like the board doc (status %d, %d bytes)", status, len(body))
	}
	ctx.SkillBytes = len(body)
	fmt.Printf("  GET /skill.md -> HTTP %d, %d bytes, first line: %q\n", status, len(body), strings.SplitN(text, "\n", 2)[0])
	return nil
})

// StageWhoAmI — skill.md §1/§5: prove the session token belongs to our
// wallet and record the handle + referral the board assigned us.
var StageWhoAmI = S("whoami", func(ctx *Ctx) error {
	var me MeResponse
	if _, _, err := ctx.Client.get("/api/me", &me); err != nil {
		return err
	}
	if !strings.EqualFold(me.Address, ctx.Addr) {
		return fmt.Errorf("token belongs to %s, expected %s", me.Address, ctx.Addr)
	}
	ctx.Me = me
	fmt.Printf("  GET /api/me -> address=%s handle=%s ref=%s link=%s\n",
		me.Address, me.Profile.Handle, me.Referral.Code, me.Referral.Link)
	return nil
})

// StageListBounties — skill.md §2: list open bounties and load the set of
// slugs this wallet has already entered (so we never re-enter one).
var StageListBounties = S("list-bounties", func(ctx *Ctx) error {
	var list BountyList
	if _, _, err := ctx.Client.get("/api/bounties?status=open", &list); err != nil {
		return err
	}
	ctx.Open = list.Bounties
	fmt.Printf("  GET /api/bounties?status=open -> %d open bounties\n", len(ctx.Open))
	for _, b := range ctx.Open {
		fmt.Printf("    - %-28s %s %s  ends %s  subs=%d\n", b.Slug, b.Amt, b.Unit, b.EndDate, b.Subs)
	}

	var subs SubmissionsResponse
	if _, _, err := ctx.Client.get("/api/me/submissions", &subs); err != nil {
		return err
	}
	ctx.Entered = map[string]bool{}
	for _, s := range subs.Submissions {
		ctx.Entered[s.BountySlug] = true
	}
	fmt.Printf("  GET /api/me/submissions -> %d existing submissions; already-entered slugs: %s\n",
		len(subs.Submissions), strings.Join(sortedKeys(ctx.Entered), ", "))
	return nil
})

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
