# go-ghwrite

![Build](https://github.com/elsbrock/go-ghwrite/workflows/Build/badge.svg)

Commit file(s) to Github repo using the GitHub API

## Synopsis

```
Usage of:

  # single file
  go-ghwrite [opts] repo/slug:targetfile < sourcefile

  # multiple files
  tar cvf - file1 file2 file3 | go-ghwrite -read-tar repo/slug:

Parameters:
  -branch string
        the git branch (default "main")
  -commit-msg string
        the commit message (default "update submitted via go-ghwrite")
  -email string
        the author email, defaults to the owner email of the token
  -name string
        the author name, defaults to the owner name of the token
  -read-tar                                                                                        
        interpret input as tarball and upload individual files
                                                                                                   
A valid Github token with scope `repo` is required in GOGHWRITE_TOKEN.
```

## Description

This is a small CLI tool that can be used to commit one or several files to a
GitHub repository using the GitHub API. Now typically you would of course do
that with `git` :-) and you are probably wondering how this can be useful:

There are certain scenarios where you do not have or want to install `git` and
configured to be able to push to the repository, and instead just be able
commit and push with a single call. In my case I would like to commit the
configuration of my router on each change but the router's default installation
does not have `git` installed nor do I want to configure and maintain the key.

## Usage

Usage is quite simple: specify the target repository (e.g. `elsbrock/testrepo`)
and the target file. If the target file does not exist it will be created,
updated otherwise. If the target file contains slashes `/` these will be
interpreted as directory.

You may also submit multiple files using `-read-tar`; in that case the input
must be an uncompressed tarball. An empty target file may be used when reading
from a tarball to represent the repository root, and all files of that tarball
will be extracted into the root.

Each successful call to the CLI will create a single commit, ie. when writing
multiple files at once using the tarball method a single commit will be created
for all of them.

If either `-name` or `-email` is given, both need to be provided. Otherwise the
author information of the token owner is used.

### Configuration

Create a new Personal Access Token with scope `repo` and export it into your
environment under the name `GOGHWRITE_TOKEN`.

> Beware: this token is fairly powerful and cannot be restricted to selected
> repositories only. Make sure it is stored securely.

### Limitations

The size of the files is limited by the GitHub API. Every file is read,
base64-encoded and submitted synchronously via single HTTP requests, so you
should not use this for large files.

### Examples

```sh
# configure the token
export GOGHWRITE_TOKEN=…
# commit and push a single file
go-ghwrite elsbrock/testrepo:targetfile < sourcefile
# commit and push a single file into the folder myfolder
go-ghwrite elsbrock/testrepo:myfolder/targetfile < sourcefile
# commit and push the contents of a tarball to the root of the repo
tar cvf - file1 file2 file3 | go-ghwrite -read-tar elsbrock/testrepo:
```

## License

MIT
