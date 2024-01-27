# walk

Walk utility inspired by Plan 9 / 9front's walk(1) command.

Walk is a lightweight alternative to the find(1) command, it's main
purpose is to walk directories recursively and print the name of
each line on a separate line.

## Installation

	go build -o walk .

## Examples

This example is straight from 9fronts man page and is adjusted to
work with this version of walk. It will walk the `~/bin/` directory,
list its files sorted by the modification date:

	walk -f -e mp ~/bin | sort -n | sed -E 's/^[^ ]+ //'

Here is another example where walk will only print files or directories
that are of depth 2 or more:

	walk -n 2, ~/bin

# License
MIT
