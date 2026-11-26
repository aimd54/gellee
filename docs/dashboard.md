# Dashboard

A read-only web view over the state repository: every project with its
state and next action, git activity, and token usage per project.

It has two halves. `gellee collect` runs on each machine you work on and
writes summaries into the state repository. `gellee serve` reads a
checkout of that repository and serves the interface. Session
transcripts never leave the machine that wrote them.

## Build

`make build` writes the binary to `bin/gellee`; [the installation
guide](install.md) builds it and puts it on the `PATH`.

## Collect

```bash
gellee collect
```

It reads `~/.claude/projects`, the working trees under the configured
roots, and writes into the state repository named by `GELLEE_STATE`:

| File | Content |
| --- | --- |
| `usage/<machine>/<YYYY-MM>.jsonl` | Tokens per day, session, project and model |
| `git/<machine>.json` | Branch, last commit, unpushed and uncommitted work per project |
| `machines/<machine>.json` | Claude Code version, memory store health and the sessions of the last 7 days |

It commits those files and nothing else. Add `--push` to push the state
repository as well, or `--no-commit` to only write the files.

A session's title and branch are written only for sessions inside the
configured roots; elsewhere the file keeps its ID, its link and its
times. No prompt text and no account identifier is written.

| Setting | Meaning | Default |
| --- | --- | --- |
| `GELLEE_STATE` or `--state` | State repository | required |
| `GELLEE_PROJECTS_ROOT` | Directory that holds the roots | `projects_root` in `config` |
| `GELLEE_MACHINE` or `--machine` | Name the files are written under | short host name |
| `GELLEE_MEMORIES` or `--memories` | Checkout of the memory store, read for this machine's last push and the versions set aside | none |
| `--transcripts` | Transcript directory | `~/.claude/projects` |

The versions set aside are counted under the machine's name in the
store, so `--machine` has to be the name the machine joined the memory
store with. Both default to `GELLEE_MACHINE`, else the short host
name.

`/handoff` runs the collector when it is on the `PATH`, so the figures
are as fresh as the last handoff. To refresh them on a schedule, add a
user-level timer:

```ini
# ~/.config/systemd/user/gellee-collect.service
[Service]
Type=oneshot
Environment=GELLEE_STATE=%h/state
Environment=GELLEE_PROJECTS_ROOT=%h/projects
ExecStart=%h/.local/bin/gellee collect --push
```

```ini
# ~/.config/systemd/user/gellee-collect.timer
[Timer]
OnCalendar=hourly
Persistent=true

[Install]
WantedBy=timers.target
```

```bash
systemctl --user enable --now gellee-collect.timer
```

### What is counted

- Every assistant message once, whatever the number of lines it was
  streamed over, with subagent runs counted under their session.
- Each message under the project of the directory it ran in. Work
  outside the configured roots is summed under `other`.
- Figures already collected are kept when Claude Code later removes the
  transcript they came from.
- A renamed project keeps its history when its status file lists the old
  name in `formerly`.

## Serve

```bash
gellee serve --state ~/state
```

Open <http://127.0.0.1:8080>.

| Flag | Meaning | Default |
| --- | --- | --- |
| `--state` | State repository checkout | `GELLEE_STATE` |
| `--listen` | Address | `127.0.0.1:8080` |
| `--refresh` | How long files are served before being read again | `1m` |
| `--demo` | Serve invented data from a temporary directory, ignoring `--state` | off |

Outside the demo, the server reads files and writes nothing. It has no
authentication of its own: it listens on loopback unless told
otherwise, and on a server it belongs behind a reverse proxy that
authenticates.

### On a server

The server needs a checkout of the state repository and something that
keeps it current. A read-only deploy token is enough.

```bash
git clone https://git.internal/you/state.git /srv/gellee/state
make image
docker run -d --name gellee --read-only \
  -v /srv/gellee/state:/state:ro -p 127.0.0.1:8080:8080 \
  gellee:"$(git rev-parse --short HEAD)"
```

`make image` builds the current commit and tags the image with its short
hash; it stops when the files that go into the image have uncommitted
changes, so a tag always names what is inside. To run the image
somewhere else, push it to a registry and deploy the digest it prints:

```bash
make image-push IMAGE=registry.internal/tools/gellee
```

Each release publishes a signed image to `ghcr.io/aimd54/gellee`,
pinned by digest in its notes; [the installation
guide](install.md#the-server) shows how to verify and use it.

The container runs as user 65532, so the checkout has to be readable by
others: clone it with a umask of `022`, or pass `--user` with the owner
of the checkout. A directory the server may not list is shown as a file
problem and its content is left out.

Keep the checkout current with a timer that runs
`git -C /srv/gellee/state pull --ff-only`, or with a sync sidecar in a
cluster. The dashboard picks up new files within the refresh interval.

## Demo

`gellee serve --demo` serves the dashboard over invented projects,
without a state repository:

```bash
gellee serve --demo
docker run --rm -p 8080:8080 ghcr.io/aimd54/gellee:vX.Y.Z --demo
```

The demo is written into a temporary directory when the server starts
and removed when it stops; `--state` and `GELLEE_STATE` are ignored.
Every date is placed relative to the start, or to `--today` when it is
given, so the same `--today` always shows the same pages. A badge in
the rail says the data is invented.

The demo writes to a temporary directory, so with `--read-only` the
container needs a writable `/tmp` (`--tmpfs /tmp`). Without `--today`,
the dates stay where they were written at start, so after about two
hours the machines read as not reporting.

| Project | State | What it shows |
| --- | --- | --- |
| `studio/loom` | active | The focus project: a session running, every section filled, usage written under its former name `studio/shuttle` |
| `studio/loom-site` | active | A group shared with `studio/loom`; uncommitted work |
| `studio/pathfinder` | active | A commit newer than its status |
| `garden/seedbank` | active | Unpushed commits; the fourth active project against a limit of 3 |
| `garden/trellis` | parked | A parked reason; usage on a model with no price |
| `garden/almanac` | done | A finished project |
| `studio/weft` | done | A project replaced by `studio/loom` |

Of the three machines, `laptop` is current, `desktop` last reported
three hours ago with an older Claude Code and two memory versions set
aside, and `worker-1` has not pushed its memories for two days.

## Views

| View | Shows |
| --- | --- |
| Overview | The bench: one card per active project, most recent handoff first, with its next action, warnings, usage over 7 days and the machine and time of its last session. Then four figures, the workbench (what needs attention, what is parked, or everything), usage over the last 30 days and the first ideas |
| Projects | Every project as a row under its group, filtered by state and root, sorted by last handoff or by usage |
| A project | Its own page at `/projects/<root>/<name>`: the next action, where work stopped, what remains, its ideas, git facts per machine, its sessions on every worker with the ended ones folded apart, and usage, with the table per day and per model folded |
| Usage | Usage per project and per model over a chosen period, with usage per day |
| Ideas | The *Ideas* section of every status file as notes, filtered by project |
| Workers | One card per machine that reports, the most recently seen first: when it was last seen, its Claude Code version, its memory sync, its sessions of the last 7 days with the ended ones folded apart and, folded, its checkouts with their warnings |

A project is named by the part of its key after the root, and dates are
written "today", "yesterday", then "9 November".

The workbench states each warning as a sentence. *Stale status* means a
commit is more recent than the last handoff, *unpushed work* that
commits are missing from the upstream or that there is no remote,
*uncommitted work* that the working tree has changes. When several
machines report a project, the sentence names the machine its figures
come from.

### Workers

A worker is up when it collected within the last two hours. Its
Claude Code version is marked behind when it is older than the newest
version among the workers seen in the last 7 days, and its memory sync
stale when it has not pushed for a day before its last collection.

*Open* goes to `https://claude.ai/code/session_<id>`. The `<id>` comes
from the ID a session connected through Remote Control records, or,
for a session Remote Control spawned into a worktree, from the
worktree's name, `.claude/worktrees/bridge-cse_<id>`. The Claude Code
documentation does not describe this correspondence; if it changes,
the link may lead nowhere while the session's row stays right.

Sessions started from the phone into a spawned worktree count for the
project the worktree belongs to. They have a title once they are
renamed with `/rename`.

A session has ended when its process has, for instance when Claude
Code is closed or a session started from the phone is archived. A
resumed session counts as running again, and an ended session can
still be resumed. A session whose process stopped without recording
its end stays among the running ones until its last activity is 7 days
old.

## Estimated cost

The headline figure is tokens without cache reads. To show an estimated
cost instead, add a `prices` file to the state repository, one model per
line with the price of a million tokens of each kind:

```text
# model            input  output  cache_write  cache_read
example-model-1    3      15      3.75         0.30
```

A sum that includes a model missing from the file is shown in tokens.
Figures that sit side by side share one unit: the rows of a list, the
cards of the bench and the three periods of a project are all shown in
tokens as soon as one of them includes such a model.

## Development

```bash
make check      # every gate
make e2e-web    # browser tests, in a container
```

For work on the interface, run the server and the Vite development
server side by side:

```bash
bin/gellee serve --state tests/fixtures/state --today 2026-11-11
cd web && npm run dev
```
