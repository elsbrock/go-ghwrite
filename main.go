package main

import "os"
import "flag"
import "strings"
import "fmt"
import "io/ioutil"
import "context"
import "encoding/base64"
import "github.com/google/go-github/v32/github"
import "golang.org/x/oauth2"

var (
	name      = flag.String("name", "Max Mustermann", "the author name")
	email     = flag.String("email", "me@example.com", "the author email")
	branch    = flag.String("branch", "master", "the git branch")
	commitMsg = flag.String("commit-msg", "update submitted via go-ghwrite", "the commit message")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of: %s [opts] repo/slug:filename\n\nParameters:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, ("\nA valid Github token with scope `repo` is required in GOGHWRITE_TOKEN.\n"))
}

func main() {
	flag.Usage = usage
	flag.Parse()

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

	repoSHA, _, err := client.Repositories.GetCommitSHA1(ctx, owner, repo, *branch, "")
	if err != nil {
		fmt.Println(err)
		return
	}

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println(err)
		return
	}

	fblob := &github.Blob{}
	b64c := "base64"
	fblob.Encoding = &b64c
	b64data := base64.StdEncoding.EncodeToString(data)
	fblob.Content = &b64data

	blob, _, err := client.Git.CreateBlob(ctx, owner, repo, fblob)
	if err != nil {
		fmt.Println(err)
		return
	}

	entries := make([]*github.TreeEntry, 0, 0)
	fileSHA := blob.GetSHA()
	mode := "100644"
	blobType := "blob"
	entry := github.TreeEntry{
		SHA:  &fileSHA,
		Path: &destpath,
		Mode: &mode,
		Type: &blobType,
	}
	entries = append(entries, &entry)

	tree, _, err := client.Git.CreateTree(ctx, owner, repo, repoSHA, entries)
	if err != nil {
		fmt.Println(err)
		return
	}

	commit := &github.Commit{}
	author := &github.CommitAuthor{}
	author.Name = name
	author.Email = email
	commit.Author = author
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
