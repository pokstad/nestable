# Nestable

A terminal based app for taking nested notes.

## Overview

Nestable is a terminal based app for nesting notes within each other. It is delivered as a single executable binary with batteries included. It stores notes locally in a single SQLite file. It has subcommands to help edit notes, find notes, and export notes.

## Features

- Single file executable - easy to install and remove, Nestable is self contained in a single file executable (thanks Go!)
- Single file for all notes - all notes and files are stored in a single file database (thanks SQLite!). Easy to back up and transport.
- Use any editor that can be launched from the commandline (vi/mate/vscode)
- Easy to find notes with fuzzy find feature (thanks Go FuzzyFinder!)

## Installation

Download Nestable from the releases. Place it in a directory in your `PATH` environment variable, such as `/usr/local/bin`.

Want to build it from source? If you have all the build dependencies, you can run `make` inside the repo.

## Setup

### Customize your text editor

Nestable can be configured to work with any text editor launchable from the command line. Nestable keeps track of your desired editor by looking up a config value in the nest database. You can view the current value with:

`nst get-config -key editor`

By default, the editor is set to `vi`.

You can change it with `nst set-config -key editor -val $MYEDITOR`

If the database doesn't have a value for the editor, or it is set to an empty string, Nestable will defer to the `EDITOR` environment variable for which editor to run.

To see which editor is currently configured

Note: Nestable requires that the command used to launch the editor must wait to exit until the file is closed. For example, TextMate has a CLI command `mate` that accepts an argument `-w` that enables this behavior. Without `-w`, `mate` will return immediately before the file is done being edited and fail to capture the changes in the database. Nestable is aware of this nuance and appends the necessary args for the following supported editors:

1. `mate` (Textmate)

## Nest (Database File)

The database file that stores notes is called a "nest". The nest will have a `.nest` suffix. The nest is actually a SQLite3 database that stores all notes and attachments in a single file.

Nestable will look in the current directory for the hidden file `.notebook.nest`. If it cannot find this file in the current directory, it will default to the file `~/.notebook.nest`. If this file does not exist, an error will be returned.

You can override this behavior by providing one of these options:

- `nst -n [path/to/my.nest]` - provide this top level CLI argument to specify a location. If the location is omitted, an interactive fuzzy finder will be displayed to allow selecting all detected nest files. Overrides `NESTABLE_NEST` environment variable when provided.
- `NESTABLE_NEST` - set this environment variable to the path to the desired nest

## Usage

Cheatsheet for common usage:

| Short Command | Description |
|---------------|-------------|
| `nst`   | view usage help |
| `nst i` | initialize nest |
| `nst n` | create a new note |
| `nst e` | select a note to edit |
| `nst v` | select a note to view |
| `nst b` | browse all notes |
| `nst gc -key <key>` | get a configuration value |
| `nst sc -key <key> -value <value>` | set a configuration value |

### Explore commands

Don't know what Nestable can do yet? Run it without a subcommand to see a list of all possible subcommands:

`nst`

### Initialize nest

Before you can start writing notes, you need to initialize your nest:

`nst i(nit)`

This will create an empty nest at `~/.notebook.nest`.

### Creating and editing a note

To quickly add a new note: `nst n(ew)`

To quickly edit an existing note: `nst e(dit) [id]`

If the optional `id` value is not specified, Nestable will display an interactive list to select the desired note.
By default, the last modified note will be selected.

### Viewing a note

The view subcommand allows you to view a note without leaving the terminal.

To select a note to view, `nst view` opens a fuzzy finder to locate a note for viewing.
By default, the last modified note will be selected.

To view a specific note, `nst v(iew) [<note-ish>]`

### Browsing notes

To browse all notes: `nst b(rowse)` displays all notes in a outline in the terminal.

### Configuration

Each nest can track configuation values in a key-value database. To select a config for viewing:

`nst g(et-)c(onfig)`

You can also specify a config:

`nst g(et-)c(onfig) -key <config-key>`

You can set a config with:

`nst s(et-)c(onfig) -key <config-key> -value <config-value>`

## FAQ

### What is `<note-ish>`?

Note-ish is inspired by [Git's "commit-ish" or "tree-ish"](https://stackoverflow.com/questions/23303549/what-are-commit-ish-and-tree-ish-in-git), Note-ish is a notation to specify a note one of several ways. The following list of notations are in order of precedence:

1. note ID - the auto incrementing note ID assigned to each note. This is typically displayed within brackets `[1]` when viewing notes.
1. note revision sha256 hash - the partial or full hexadecimal SHA256 digest for the revision of the desired note (see note revisions). This allows you to refer to an older revision of a note. You can specify as little of the digest prefix as you want, but it must be enough of the prefix to guarantee a unique revision.
1. fuzzy match heading - the first result for the given pattern that matches a note's heading (first 80 chars) using the internal fuzzy finder.