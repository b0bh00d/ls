# ls
Enahnced directory listing for Windows console

## Summary
The original **ls**, which I wrote in Python, started out as an attempt to provide
more mimicry of the UN*X command of the same name.  However, that Python script
diverged somewhat from the original goal, morphing into something with more features
targeting Windows-based systems.

This project is _not_ the orignal **ls** Python script I wrote.  Instead, I decided to
use that script as the blueprint for a learning challenge in Go, so this project
is a port of that Python script.  The result actually includes some minor
improvements over the original Python implemenation.

## Configuration

**ls** uses a JSON file to hold its configuration details.  This file, called
`ls.json`, is expected to be found in the %APPDATA% folder.  This project includes an
example `ls.json` file, the settings of which were used to create screen shots.

## Coloring

File/folder coloring is a basic feature of **ls**.  The ANSI color values used
are contained in the JSON configuration file.  This file can be modified to change
the default colors used, and it can be extended to colorize more file types than
are provided by default.

Windows does not by default enable ANSI color support in its cmd.exe console
windows.  **ls** relies on C++ code in a Windows DLL to manage color support within
a cmd.exe console.  The C++ code (along with the VS2019 project files) for this
shared library can be found in the `term/DLL` subfolder.

## SCM Status

**ls** has built-in support for detecting the presence of a source-control manager
for the folder being displayed.  At this time, the three most popular SCMs are
supported: Subversion, Mercurial and git.  If **ls** detects that the current folder
is under SCM control, it will attempt to retrieve the current state of the entries
within the context of that SCM.

For example, a Mercirual folder might display the following:

![hg](https://user-images.githubusercontent.com/4536448/109701790-aa517f80-7b50-11eb-83ca-7ba481b1331c.png)

## Metadata

Once of the more advanced features of **ls** is its display of entry metadata.
Such metadata can fall into one of three categories:

1. Symlinks (a.k.a. reparse points, in Microsoft not-invented-here speak)
1. Alternative Data Streams (in particular, Directory Opus)
1. Total Commander descript.ion file

Display of these metadata types with **ls** is illustrated in the following
screenshot:

![symlinks and metadata](https://user-images.githubusercontent.com/4536448/109701816-b3425100-7b50-11eb-8412-893a4094dcfa.png)


As with coloring, this feature of **ls** also relies upon C++ code that
interfaces directly with Win32 API calls.  Accessing ADSs is somewhat
acrobatic given the interfaces and casting required, so C++ was the best
choice to address this.  You can find the source code (along with VS2019
project files) in the `meta/DLL` subfolder.  Unlike coloring, this
DLL is optional, and **ls** will function without it (you just won't get
metadata displayed).

## Building

On Windows, compile with: `go build -ldflags "-s -w" .`

The Go system should pull down all modules that **ls** requires
in order to build the resulting executable.

I hope you find this useful.
