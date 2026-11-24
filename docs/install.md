# Installation

gellee is installed on every machine you work on: the scripts, the
`/handoff` and `/memories` skills, the session hook and the collector.
The status files live in one state repository, a private git repository
whose remote carries them from one machine to the next.

## What it needs

- `git`, `bash`, `jq` and `column` (from util-linux).
- [mise](https://mise.jdx.dev), which installs the pinned Go and Node to
  build the collector and the dashboard.
- Claude Code.
- A private remote for the state repository, reachable from every
  machine.

## The layout

Every machine keeps its projects under one projects root, in one
directory per root named in the configuration, with each project
directly beneath:

```text
~/projects/                  GELLEE_PROJECTS_ROOT
├── public-github/           a root named in config
│   ├── webapp/              project public-github/webapp
│   └── gellee/
└── public-gitlab/
    └── tooling/
~/state/                     GELLEE_STATE, outside the roots
```

A project's name is its path under the projects root, and its status
file is that path in the state repository, so the directory names have
to be the same on every machine; the projects root itself may differ.
Paths are resolved before they are compared: a root has to be a real
directory under the projects root, not a symbolic link pointing
elsewhere. The state repository stays outside the roots, or it would be
counted as a project.

## The state repository

On the first machine, create it with its configuration, then give it a
remote and push it:

```bash
mkdir -p ~/state && cd ~/state && git init
printf 'roots=public-github public-gitlab\nlimit=3\nactive_days=10\n' > config
git add config && git commit -s -m "chore: add the configuration"
git remote add origin git@git.internal:you/state.git
git push -u origin main
```

On every other machine, clone it:

```bash
git clone git@git.internal:you/state.git ~/state
```

## On each machine

1. Clone this repository next to your projects, then build gellee and
   put it on your `PATH` with the skills:

   ```bash
   git clone https://github.com/aimd54/gellee.git ~/projects/public-github/gellee
   cd ~/projects/public-github/gellee
   mise trust && mise install
   mise exec -- make build
   mkdir -p ~/.local/bin ~/.claude/skills
   ln -s "$PWD"/bin/gellee-* ~/.local/bin/
   ln -s "$PWD/bin/gellee" ~/.local/bin/gellee
   ln -s "$PWD/skills/handoff" ~/.claude/skills/handoff
   ln -s "$PWD/skills/memories" ~/.claude/skills/memories
   ```

   `bin/gellee-*` are the scripts; `bin/gellee`, the collector and the
   dashboard, exists once `make build` has run.

2. Export the variables from your shell profile:

   ```bash
   export GELLEE_STATE="$HOME/state"
   export GELLEE_PROJECTS_ROOT="$HOME/projects"
   ```

   The collector names its files after the short host name. Two machines
   with the same host name would overwrite each other's figures: give one
   of them `GELLEE_MACHINE`.

3. Register the hook and the variables in `~/.claude/settings.json`,
   merging with what is already there:

   ```json
   {
     "env": {
       "GELLEE_STATE": "/home/you/state",
       "GELLEE_PROJECTS_ROOT": "/home/you/projects"
     },
     "hooks": {
       "SessionStart": [
         {
           "hooks": [
             {
               "type": "command",
               "command": "/home/you/.local/bin/gellee-session-start"
             }
           ]
         }
       ]
     }
   }
   ```

4. On the first machine only, create a stub for every project and choose
   which to park:

   ```bash
   gellee-init
   gellee-status
   ```

   `gellee-init` never touches an existing file, so it is harmless on a
   later machine, but the stubs are already there.

## The server

The server runs the released image, pinned by digest:

```text
ghcr.io/aimd54/gellee:vX.Y.Z@sha256:<digest>
```

The digest is in the release notes. The image is signed by the release
workflow; check it before deploying:

```bash
cosign verify ghcr.io/aimd54/gellee@sha256:<digest> \
  --certificate-identity-regexp '^https://github.com/aimd54/gellee/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

It serves a checkout of the state repository mounted at `/state` on
port 8080; see [the dashboard guide](dashboard.md).

## Checking a machine

```bash
cd ~/projects/public-github/gellee && gellee-path
gellee-status
gellee collect --no-commit
git -C "$GELLEE_STATE" status -s
```

`gellee-path` prints the project's status file in the state repository.
The collector writes `git/<machine>.json`, and `usage/<machine>/` once
Claude Code has run there; `git status` lists those files and nothing
else. A Claude Code session started in a project then opens with
`gellee status for <root>/<name>`.

`gellee collect --push` commits and pushes them. `/handoff` runs the
collector itself, and [the dashboard guide](dashboard.md#collect) has a
timer that runs it every hour.

## The memory store

Memory files and the global instructions file travel through a second
private repository, described in [the memories guide](memories.md).
Create it empty on the forge, then on each machine:

1. Clone it outside the roots:

   ```bash
   git clone git@git.internal:you/memories.git ~/memories
   ```

2. Export the variable from your shell profile, and give it and the
   directory access to Claude Code in `~/.claude/settings.json`, merging
   with what is already there:

   ```bash
   export GELLEE_MEMORIES="$HOME/memories"
   ```

   ```json
   {
     "env": {
       "GELLEE_MEMORIES": "/home/you/memories"
     },
     "permissions": {
       "additionalDirectories": ["/home/you/memories"]
     }
   }
   ```

   The hook reads the variable to sync at session start, and `/memories`
   edits the checkout through the directory access.

3. Join the machine, after reading what it would do:

   ```bash
   gellee memories adopt
   gellee memories adopt --apply
   ```

   Before the second machine joins, run `/memories review all` on the
   first, so the second receives the reviewed set.

4. Install the timer that syncs every ten minutes:

   ```ini
   # ~/.config/systemd/user/gellee-memories.service
   [Service]
   Type=oneshot
   Environment=GELLEE_STATE=%h/state
   Environment=GELLEE_PROJECTS_ROOT=%h/projects
   Environment=GELLEE_MEMORIES=%h/memories
   Environment=GELLEE_MACHINE=m1
   ExecStart=%h/.local/bin/gellee memories sync
   ```

   Keep the `GELLEE_MACHINE` line only when the shell profile or the
   settings set that variable, and give it the same value there: without
   it the timer names the machine after its host and the session hook by
   the variable, so the notices miss the versions the timer set aside.

   ```ini
   # ~/.config/systemd/user/gellee-memories.timer
   [Timer]
   OnCalendar=*:0/10
   Persistent=true

   [Install]
   WantedBy=timers.target
   ```

   ```bash
   systemctl --user daemon-reload
   systemctl --user start gellee-memories.service
   journalctl --user -u gellee-memories -n 5
   systemctl --user enable --now gellee-memories.timer
   ```

   The service runs without a terminal: the journal has to show the
   `sync:` summary line and no push error before the timer is enabled.
   A remote that asks for a passphrase or a password here never gets
   the push.

## Private files

A file whose name matches `*.private.*` and that git ignores is a
private file: notes and plans kept out of a public repository. gellee
keeps them in the state repository, under
`private/<root>/<name>/<path in the checkout>`, and puts a symbolic
link at the usual path in each checkout, so they reach every machine
the state repository reaches.

| Command | Effect |
| --- | --- |
| `gellee private link` | Makes the missing links in the checkout holding `--dir` (default: the current directory) |
| `gellee private sweep` | Moves the private files that are still regular files into the state repository and links them |
| `gellee private status` | Lists what is unswept, in the way, missing, broken, or not ignored by git; `--json` for the skill |

The session-start hook runs `link`, `/handoff` runs `sweep` and commits
`private/<root>/<name>` with the status file. A regular file where a
link belongs is left alone and listed as in the way; `sweep` replaces it
only when its bytes equal the stored copy, or when the stored copy is
the one last committed and the file is newer, as after an editor saved
over the link. To remove a private file,
remove it from the state repository: a link deleted alone comes back at
the next session start. To bring existing files in, run
`gellee private sweep` in each checkout of the machine that holds them,
then commit and push the state repository.

## Moving a project to another root

A project's key is its path under the projects root, so moving it from
one root to another, for instance from `public-gitlab` to a root for a
private forge, changes its key from `old` to `new`:

1. In the state repository, `git mv` the status file to `new`'s path,
   add `old` to its `formerly`, which keeps its usage figures attached,
   and `git mv` its directory under `private/`. Commit and push. The
   links in each checkout then point at the old place; the next session
   start, or `gellee private link`, points them at the new one.
2. In the memory store, `git mv` the project's directory. Commit and
   push.
3. On each machine, move the checkout, then rename the project's
   directory under `~/.claude/projects/` to the name Claude Code gives
   the new path: the resolved path with every `/` and `.` turned into
   `-`. Without the rename, the next memory sync counts the old
   directory as an orphan.
4. On a machine whose checkouts come from its provisioning, move the
   clone there.

## Working on two machines

Start with `git -C "$GELLEE_STATE" pull --rebase`, so the status files
are the ones the other machine pushed; `/handoff` pushes when you stop.
Claude Code's memory files travel through [the memory
store](#the-memory-store); user settings and skills do not travel yet:
see [the roadmap](roadmap.md#continuity-between-machines).
