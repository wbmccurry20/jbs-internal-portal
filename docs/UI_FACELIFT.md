# JBS Internal Portal UI Face Lift

## Brand tokens

- Primary: `#00A0E0`
- Primary hover: `#0088CC`
- Dark: `#1A1A1A`
- Charcoal: `#3E3832`
- Cream: `#F0E8E0`
- Gold: `#C0A870`
- Sage: `#788078`
- Brown: `#885830`
- Light blue: `#C0D8F0`
- Canvas: `#F7F6F4`

## Shared components

- `.btn-primary`
- `.btn-secondary`
- `.btn-ghost`
- `.btn-danger`
- `.page-header`
- `.card`
- `.kpi-card`
- `.chip`
- `.table-shell`
- `.empty-state`
- `.field-label`
- `.form-input`

These are defined in the shared styles layer and intended for reuse across authenticated pages without rewriting behavior.

## Intentionally not changed

- No licensing feature logic changes
- No reimbursement / Concur conversion refactor
- No Tailwind 4 migration
- No backend or schema changes
- No new product modules or product feature work

## Notes

- The login experience uses the dark JBS brand treatment and condensed heading language.
- Authenticated pages keep the tool layout on a warm off-white canvas with light cards and dark navigation chrome.
- The active licensing map uses blue for active / licensed, gold for expiring, red for expired, and a dark slate background with dark strokes.
