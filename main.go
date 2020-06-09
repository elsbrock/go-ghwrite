package main

import "os"
import "flag"
import "strings"
import "fmt"
import "io/ioutil"
import "context"
import "github.com/google/go-github/v32/github"
import "golang.org/x/oauth2"

var (
	name      = flag.String("name", "Max Mustermann", "the author name")
	email     = flag.String("email", "me@example.com", "the author email")
	branch    = flag.String("branch", "master", "the git branch")
	commitmsg = flag.String("commit-msg", "update submitted via go-ghwrite", "the commit message")
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

	cfg, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println(err)
		return
	}

	copts := &github.RepositoryContentGetOptions{}
	fileContent, _, resp, err := client.Repositories.GetContents(ctx, owner, repo, destpath, copts)
	if err != nil {
		if resp == nil || (resp != nil && resp.StatusCode != 404) {
			fmt.Println(err)
			return
		}
	}

	sha := fileContent.GetSHA()

	fopts := &github.RepositoryContentFileOptions{
		Message:   commitmsg,
		Content:   cfg,
		SHA:       &sha,
		Branch:    github.String(*branch),
		Committer: &github.CommitAuthor{Name: github.String(*name), Email: github.String(*email)},
	}

	_, _, err = client.Repositories.CreateFile(ctx, owner, repo, destpath, fopts)
	if err != nil {
		fmt.Println(err)
		return
	}
}
