# Raven

This is a command-line mod installer for Death's Door.

## Installation

To install this tool, grab a binary release, or if you have a [Go][] toolchain installed, run this command to compile it from source:

    $ go install github.com/dpinela/Raven/cmd/raven@latest

[Go]: https://go.dev

## Commands

There are two ways of invoking Raven's commands;
either run the command directly from the shell
(as in the examples below), or start Raven without
arguments. Doing the latter opens a console that can
be used to enter the same commands (without the
`raven` prefix), which makes it so that you can launch
the executable directly on Windows and use it.

### setup

The setup command installs BepInEx onto your game
and records the install location for future commands.
In its simplest form, available only on Windows,
it tries to find the game in the default
locations for Steam and GoG:

    raven setup

If you don't have the game at one of these locations,
you must specify its path explicitly:

    raven setup /location/of/Deaths_Door.exe

or using quotes if the path contains spaces:

    raven setup "/location/of/Death's Door/Deaths_Door.exe"

Backslashes are accepted too on Windows, but make sure to double them
if using the quoted form:

    raven setup "C:\\location\\of\\Death's Door\\Deaths_Door.exe"

Either the path to the game executable itself or to
its parent directory are acceptable.

### list

The list command serves to look up information about any mod listed on [modlinks][].
In its simplest form, it prints a list of all available mods:

    $ raven list
    AlternativeGameModes
    ItemChanger
    MagicUI
    Plando
    Randemo
    RecentItemsDisplay
    ...

Using the `-s` option, you can reduce the list to mods containing a particular string
(case-insensitive) in their names:

    $ raven list -s rand
    Randemo
    ...

The `-i` option reduces the list to mods that are currently installed, and also adds
any mods you have installed that aren't listed on modlinks.

The `-d` option adds more detailed information about each mod:

    $ raven list -d -s randemo
    Randemo
        Repository: https://github.com/dpinela/DeathsDoor.Plando
        Dependencies: Plando
        A plando that served as a demo for the randomizer

`-d` can technically be used without `-s` as well, but there is usually little reason
to do that.

[modlinks]: https://github.com/dd-modding/modlinks

## install

The install command downloads and installs one or more mods that are listed on
modlinks, along with any necessary dependencies. It takes as arguments the list of
mods to install:

    $ raven install randemo

The arguments usually do not need to match mod names exactly; for each one, the mod
to install is selected by the first of the following that matches one and only one mod:

- A partial case-insensitive match
- A full case-insensitive match
- A full case-sensitive match

If Raven can't disambiguate which mod you want, it will print an error message
explaining why, and skip installing that mod:

    $ raven install item
    "item" is ambiguous: matches ItemChanger, RecentItemsDisplay

    $ raven install modthatdoesnotexistatallandneverwill
    "modthatdoesnotexistatallandneverwill" matches no mods

Once it resolves which mods to get, Raven installs the latest available version of
each of them, **irrespective of which, if any, version you had installed before.**
It makes no attempt to keep track of which mod versions are currently installed in
any way; to save time and bandwidth, it instead caches downloads and relies on the
hash listed in modlinks to check whether the cached files are still valid and
up-to-date.

For most mods, installing a new version **entirely removes** the previously
installed one, so any custom files added to that mod's folder will be deleted as
well.

### yeet

The yeet command fully removes the named mods. It uses the same matching algorithm
as the install command, so partial matches will work when unambiguous:

    $ raven yeet randemo
    Yeeted Randemo

This command can target any mod you have installed, regardless of source, including mods that do not
exist on modlinks or were installed by a different tool.