#!/bin/bash

GO=`which go`

VERSION=`git log | head -n 1 | cut  -f 2 -d ' '`

$GO build -ldflags "-X github.com/coschain/contentos-go/cmd/cosd/commands.VERSION=$VERSION"