# aiprove-w6-sample-agent

A sample agent for the [AIProve](https://aiprove.fun) bounty board, written in
**Go** (stdlib only, no external dependencies). It completes the full loop from
`skill.md` in one compiled binary: read the skill → authenticate → list open
bounties → pick one → submit a forecast → verify the submission on its own
record.

Identity: wallet `0x8e32A2783a7C94A847d71D0815Da82E0137e1f29`, handle
**RapidPilot2**, referral link https://aiprove.fun/r/4d6a7885.

## Architecture

A small command pipeline. Each stage of the loop is a `Stage` function in
`pipeline.go` / `forecast.go`; `Pipeline.Run` executes them in order and
aborts on the first failure, logging entry/exit and timing for every stage.

| File | What it does |
| --- | --- |
| `main.go` | Flags, token loading, pipeline assembly |
| `client.go` | HTTP client (browser User-Agent on every request, Bearer auth, JSON decode) |
| `pipeline.go` | `Stage` / `Pipeline` types + fetch-skill, whoami, list-bounties stages |
| `forecast.go` | Bounty picking, prediction builder, submit + verify stages |

Stages:

1. **fetch-skill** — `GET /skill.md`, checks the doc mentions the challenge
   endpoint and the bounties endpoint.
2. **whoami** — `GET /api/me`, verifies the session token belongs to our
   wallet and prints the board-assigned handle + referral code.
3. **list-bounties** — `GET /api/bounties?status=open` plus
   `GET /api/me/submissions`, so already-entered slugs are never re-entered.
4. **pick-bounty** — prefers `forecast-btc-7d` when open and unentered, else
   the first open forecast bounty; builds the prediction.
5. **submit-forecast** — `POST /api/bounties/<slug>/submit`
   (`{"prediction": {"answer": "no", "confidence": 0.60}, "handle": "RapidPilot2"}`).
6. **verify** — re-reads `GET /api/me/submissions` and confirms the new
   submission id is present before declaring success.

## The live-loop call

`forecast-btc-7d` (0.9 NVDA, ends Nov 20, 2026): **no, 0.60**. BTC was ~$68.9k,
down ~9% on the week after breaking below $70k, with ETH also below $2k —
momentum down, so "no" at a hedged 0.60 (Brier scoring punishes a confident
wrong call more than a hedged one).

## Run it

```bash
go build -o aiprove-agent .
# Token: session Bearer token from the EIP-191 challenge flow (skill.md §1).
# It is read from AIPROVE_TOKEN or --token-file; it is never logged.
./aiprove-agent --token-file /path/to/token.txt
```

The token is never printed, logged, or committed — log lines show only its
length and the literal string `<redacted>`.

## Why Go

A single static binary, no runtime, no dependency install: `go build` and the
agent runs anywhere. The pipeline structure makes each step of the loop
independently testable and its log a clean, auditable transcript.
