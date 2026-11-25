# Security Policy

## Supported versions

dirstato has no tagged releases yet. Security fixes land on the `main` branch.

## Reporting a vulnerability

Please **do not** open a public issue for a security problem.

Report it privately through GitHub's
[private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability):
open the repository's **Security** tab and choose **Report a vulnerability**.

Include a description of the problem, steps to reproduce it and the impact you
expect. Once a fix is ready, the advisory is published with credit to the
reporter, unless you prefer to stay anonymous.

## Scope

dirstato is a read-only tool: it reads file metadata and copies a path to the
clipboard with `pbcopy`. Problems of particular interest include:

- a path copied with `c` that runs a command when pasted into a shell,
- a scan that follows a symlink or crosses a mount point,
- crashes or unbounded resource use caused by a crafted folder tree.
