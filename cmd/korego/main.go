// korego is the korego multicall binary.
// It dispatches to registered commands by argv[0] (symlink mode) or argv[1]
// (subcommand mode).
package main

import (
	"os"

	"github.com/ramayac/korego"

	// Import all utilities to trigger their init() registrations.
	_ "github.com/ramayac/korego/pkg/basename"
	_ "github.com/ramayac/korego/pkg/cat"
	_ "github.com/ramayac/korego/pkg/chgrp"
	_ "github.com/ramayac/korego/pkg/chmod"
	_ "github.com/ramayac/korego/pkg/chown"
	_ "github.com/ramayac/korego/pkg/cp"
	_ "github.com/ramayac/korego/pkg/cut"
	_ "github.com/ramayac/korego/pkg/daemon"
	_ "github.com/ramayac/korego/pkg/date"
	_ "github.com/ramayac/korego/pkg/df"
	_ "github.com/ramayac/korego/pkg/dirname"
	_ "github.com/ramayac/korego/pkg/du"
	_ "github.com/ramayac/korego/pkg/echo"
	_ "github.com/ramayac/korego/pkg/env"
	_ "github.com/ramayac/korego/pkg/find"
	_ "github.com/ramayac/korego/pkg/grep"
	_ "github.com/ramayac/korego/pkg/head"
	_ "github.com/ramayac/korego/pkg/hostname"
	_ "github.com/ramayac/korego/pkg/id"
	_ "github.com/ramayac/korego/pkg/kill"
	_ "github.com/ramayac/korego/pkg/ln"
	_ "github.com/ramayac/korego/pkg/ls"
	_ "github.com/ramayac/korego/pkg/mkdir"
	_ "github.com/ramayac/korego/pkg/mv"
	_ "github.com/ramayac/korego/pkg/printenv"
	_ "github.com/ramayac/korego/pkg/ps"
	_ "github.com/ramayac/korego/pkg/pwd"
	_ "github.com/ramayac/korego/pkg/readlink"
	_ "github.com/ramayac/korego/pkg/rm"
	_ "github.com/ramayac/korego/pkg/rmdir"
	_ "github.com/ramayac/korego/pkg/sed"
	_ "github.com/ramayac/korego/pkg/shell"
	_ "github.com/ramayac/korego/pkg/sha256sum"
	_ "github.com/ramayac/korego/pkg/sleep"
	_ "github.com/ramayac/korego/pkg/sort"
	_ "github.com/ramayac/korego/pkg/stat"
	_ "github.com/ramayac/korego/pkg/tail"
	_ "github.com/ramayac/korego/pkg/tar"
	_ "github.com/ramayac/korego/pkg/tee"
	_ "github.com/ramayac/korego/pkg/touch"
	_ "github.com/ramayac/korego/pkg/tr"
	_ "github.com/ramayac/korego/pkg/truefalse"
	_ "github.com/ramayac/korego/pkg/uname"
	_ "github.com/ramayac/korego/pkg/uniq"
	_ "github.com/ramayac/korego/pkg/wc"
	_ "github.com/ramayac/korego/pkg/whoami"
	_ "github.com/ramayac/korego/pkg/xargs"
	_ "github.com/ramayac/korego/pkg/yes"
	_ "github.com/ramayac/korego/pkg/printf"
	_ "github.com/ramayac/korego/pkg/expr"
	_ "github.com/ramayac/korego/pkg/gzip"
	_ "github.com/ramayac/korego/pkg/diff"
	_ "github.com/ramayac/korego/pkg/testcmd"
	_ "github.com/ramayac/korego/pkg/md5sum"
)

func main() {
	os.Exit(korego.Main())
}
