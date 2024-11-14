# walk

Walk is a lightweight alternative to the find(1) command, it's main
purpose is to walk directories recursively and print the name of
each line on a separate line. It is very much inspired by the 9front
command walk(1).

	Usage: walk [ -dfx ] [ -n min,max ] [ -e "fmt" ] [ name ... ] [ ! cmd ]
	  !	Run cmd in a sub shell with sh(1) for each match.
	  	If an unescaped % occurs in the command list it will
	  	be replaced with the file name.
	  -d	Print only directories.
	  -e string
	    	Specifies the output format.
	    	The attributes are automatically separated with a space.
	    	The following characters are accepted:
	    	U	Owner name
	    	G	Group name
	    	M	name of the last user to modify the file
	    	a	last access time
	    	m	last modification time
	    	n	final path element (name)
	    	p	path
	    	s	size (bytes)
	    	x	permissions (default "p")
	  -f	Print only non-directories.
	  -n string
	    	Sets the inclusive range for depth filtering.
	    	The expected format is "min,max" and both are optional.
	    	An argument of n with no comma is equivalent to 0,n.
	  -x	Print only if the executable bit is set.

## Installation

	go build

## Examples

This example is straight from 9fronts man page and is adjusted to
work with this version of walk. It will walk the `~/bin/` directory,
list its files sorted by the modification date:

	walk -f -e mp ~/bin | sort -n | sed -E 's/^[^ ]+ //'

Here is another example where walk will only print files or directories
that are deeper than 2 levels or more:

	walk -n 2, ~/bin

# License
MIT
