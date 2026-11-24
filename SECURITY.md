# Security policy

## Reporting a vulnerability

Report vulnerabilities privately through GitHub: open the Security tab
of this repository and choose "Report a vulnerability". Do not open a
public issue for a security problem.

## Supported versions

Before 1.0, only the latest minor release receives security fixes.

## Disclosure

A confirmed vulnerability is fixed in a patch release. The release notes
name the advisory.

## Scope

`gellee serve` reads a checkout of the state repository and serves a
read-only interface and API. It has no write path, no credentials and no
storage, and is meant to sit behind the operator's own authentication.

`gellee collect` reads local session transcripts and working trees and
writes summaries to the state repository.

Reports about content the operator put in the state repository are out
of scope.
