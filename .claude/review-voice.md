# Review voice

How to write anything a reviewer will read: PR review bodies, line comments, and
replies to review comments.

Not PR descriptions — those carry reasoning a reviewer needs up front and are
deliberately out of scope.

## Length follows the diff

- Scale the review to the PR's impact, not to how much there is to say. Internal
  docs, cleanup, dep bumps: shorter than normal.
- A review longer than the diff it reviews is almost always wasted effort.
- Bullets, fragments, abbreviations. No essays, no preamble, no closing paragraph
  restating the change.

## Say it once

- A point goes in the main body **or** a line comment, never both.
- Don't list what changed since the last review.
- Don't recap previous points that are now resolved. Raise a previous point only if
  it's unresolved or resolved badly.
- If you're asking for a block to be rewritten, don't also leave nits on code inside
  it — they'll never be actioned.

## Earn each request

- Weigh the cost of a change against its benefit before asking. If rewording a
  comment costs more than it's worth, don't raise it at all.
- Good code is the bar, not perfect code.
- Reviewing prose (docs, comments): correctness and clarity only. Style varies
  between people; leave it alone.
- Praise: one short sentence at most, only for something genuinely worth noting. No
  flattery, no opening compliment.

## Replies to review comments

- One or two lines. What was done, or why the current approach stands.
- Don't restate the reviewer's point back at them. No thanks-for-catching-this, no
  summary of the wider change.
- Disagreeing is fine — give the concrete reason in a sentence.

Source: review feedback from @hexaglow, August 2026.
