package main

import (
	"archive/tar"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

var (
	name      = flag.String("name", "", "the author name, defaults to the owner name of the token")
	email     = flag.String("email", "", "the author email, defaults to the owner email of the token")
	branch    = flag.String("branch", "main", "the git branch")
	commitMsg = flag.String("commit-msg", "update submitted via go-ghwrite", "the commit message")
	readTar   = flag.Bool("read-tar", false, "interpret input as tarball and upload individual files")
)

func usage() {
	fmt.Fprintf(os.Stderr, `Usage of: %s [opts]

  # single file
  go-ghwrite [opts] repo/slug:targetfile < sourcefile

  # multiple files
  tar cvf - file1 file2 file3 | go-ghwrite -read-tar repo/slug:
  
Parameters:
`, os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, ("\nA valid Github token with scope `repo` is required in GOGHWRITE_TOKEN.\n"))
}

type GithubWriter struct {
	ctx    context.Context
	client *github.Client
	owner  string
	repo   string
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if (*name != "" && *email == "") || (*name == "" && *email != "") {
		flag.Usage()
		os.Exit(1)
	}

	var owner, repo, destpath string

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	wannabeSlug := flag.Arg(0)
	if !strings.Contains(wannabeSlug, ":") {
		flag.Usage()
		os.Exit(1)
	}

	split := strings.Split(wannabeSlug, ":")
	destpath = split[1]
	if !strings.Contains(split[0], "/") {
		flag.Usage()
		os.Exit(1)
	}

	reposlug := strings.Split(split[0], "/")
	owner = reposlug[0]
	repo = reposlug[1]

	var token string
	if v, ok := os.LookupEnv("GOGHWRITE_TOKEN"); !ok {
		flag.Usage()
		os.Exit(1)
	} else {
		token = v
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	writer := GithubWriter{
		ctx:    ctx,
		client: client,
		owner:  owner,
		repo:   repo,
	}

	repoSHA, _, err := client.Repositories.GetCommitSHA1(ctx, owner, repo, *branch, "")
	if err != nil {
		fmt.Println(err)
		return
	}

	entries := make([]*github.TreeEntry, 0)

	if *readTar {
		tr := tar.NewReader(os.Stdin)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Println(err)
				return
			}

			data, err := ioutil.ReadAll(tr)
			if err != nil {
				fmt.Println(err)
				return
			}

			path := hdr.Name
			if destpath != "" {
				path = destpath + "/" + path
			}
			fmt.Printf("staging %s… ", path)

			sha, err := writer.createBlob(data)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println(sha)

			entry := treeEntryBlob(path, sha)
			entries = append(entries, entry)
		}
	} else {
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("staging %s\n", destpath)
		sha, err := writer.createBlob(data)
		if err != nil {
			fmt.Println(err)
			return
		}
		entry := treeEntryBlob(destpath, sha)
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return
	}

	tree, _, err := client.Git.CreateTree(ctx, owner, repo, repoSHA, entries)
	if err != nil {
		fmt.Println(err)
		return
	}

	commit := &github.Commit{}
	if *name != "" {
		author := &github.CommitAuthor{}
		author.Name = name
		author.Email = email
		commit.Author = author
	}
	commit.Message = commitMsg
	commit.Tree = tree
	c := github.Commit{}
	c.SHA = &repoSHA
	commit.Parents = []*github.Commit{&c}

	newCommit, _, err := client.Git.CreateCommit(ctx, owner, repo, commit)
	if err != nil {
		fmt.Println(err)
		return
	}

	ref := &github.Reference{}
	branchRef := fmt.Sprintf("refs/heads/%s", *branch)
	ref.Ref = &branchRef
	obj := &github.GitObject{}
	commitType := "commit"
	obj.Type = &commitType
	commitSHA := newCommit.GetSHA()
	obj.SHA = &commitSHA
	ref.Object = obj
	_, _, err = client.Git.UpdateRef(ctx, owner, repo, ref, false)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Fprintf(os.Stderr, "committed as %s\n", commitSHA)
}

func (w GithubWriter) createBlob(data []byte) (string, error) {
	fblob := &github.Blob{}
	b64c := "base64"
	fblob.Encoding = &b64c
	b64data := base64.StdEncoding.EncodeToString(data)
	fblob.Content = &b64data

	blob, _, err := w.client.Git.CreateBlob(w.ctx, w.owner, w.repo, fblob)
	if err != nil {
		return "", err
	}
	return blob.GetSHA(), nil
}

func treeEntryBlob(path string, fileSHA string) *github.TreeEntry {
	mode := "100644"
	blobType := "blob"
	entry := github.TreeEntry{
		SHA:  &fileSHA,
		Path: &path,
		Mode: &mode,
		Type: &blobType,
	}
	return &entry
}
