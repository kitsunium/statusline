# ADR 0004 — stdin wins, and an absent quota stays absent

- **Status**: Accepted (carried over from kodflow/status-line, frozen at 207dfe2)
- **Date**: 2026-09-28

## Decision

The host pipes `rate_limits` on every redraw: free, synchronous, current. The
usage API only adds what stdin cannot know (quotas scoped to a model family,
the credit balance) and the buckets a host build did not send; its `limits[]`
array is read first, the legacy `five_hour` / `seven_day` fields are a
fallback. A bucket absent from both sources stays absent: plans differ, and a
0 % bar would be a lie about the account. A scoped quota shows only while its
model is in use.
