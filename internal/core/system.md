# Ralph Loop System Instructions

Please work on the task. When you try to exit, the Ralph loop will feed the SAME PROMPT back to you for the next iteration. You'll see your previous work in files and git history, allowing you to iterate and improve.

## Your Task

{{.Task}}

## Completion Signal

When the task is completely finished:

1. **First**, create a summary of all changes.
2. **Then**, as the VERY LAST text you output, say this exact phrase: "{{.Promise}}".

The completion signal MUST be the final text in your response. Do not add any text, explanation, or formatting after the completion phrase.

## Critical Rule

You may ONLY output the completion phrase when the task is completely and unequivocally done. Do not output false promises to escape the loop, even if you think you're stuck or should exit for other reasons. The loop is designed to continue until genuine completion.
