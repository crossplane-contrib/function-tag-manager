package main

// Clone a Provider repo and extract the CRD manifests.
import (
	"os"

	"github.com/alecthomas/kong"
	"github.com/crossplane-contrib/function-tag-manager/cmd/generator/render"
	"github.com/crossplane/function-sdk-go"
	"github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/osfs"
	git "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/cache"
	"github.com/go-git/go-git/v6/storage"
	"github.com/go-git/go-git/v6/storage/filesystem"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
)

// CLI of this Utility.
type CLI struct {
	Debug bool `help:"Emit debug logs in addition to info logs." short:"d"`

	RepositoryDir           string `help:"local git repository cache" default:"_work/providers/provider-upjet-aws"`
	RepoURL                 string `help:"Git repo to clone" default:"https://github.com/crossplane-contrib/provider-upjet-aws.git"`
	CrossplanePackageCRDDir string `help:"Location of CRD files" default:"package/crds"`
	OutputFile              string `help:"file to output generated Go code"`
	GitBranchOriginMain     string `help:"Git branch to clone." default:"refs/remotes/origin/main"`
}

// Cloner clones Git repositories.
type Cloner struct {
	// Paths to extract from the repository
	Paths []string
	// Branch to clone
	Reference string
	// Storage backend to use
	Storage  storage.Storer
	RepoURL  string
	Worktree billy.Filesystem
}

// Generater generates filters from CRD directories.
type Generater struct {
	Cloner

	Logger        logging.Logger
	RepoDirectory string
}

func (c *CLI) Run() error {
	log, err := function.NewLogger(c.Debug)
	if err != nil {
		return err
	}

	log.Info("Generating resource filters from CRDs")
	g := Generater{
		Logger:        log,
		RepoDirectory: c.RepositoryDir,
	}

	var w *git.Worktree

	filesystemfs := osfs.New(c.RepositoryDir)

	_, err = os.Stat(c.RepositoryDir)
	if os.IsNotExist(err) {
		log.Debug("repo does not exist on the filesystem, cloning", "directory", c.RepositoryDir)

		storage := filesystem.NewStorage(filesystemfs, cache.NewObjectLRU(cache.DefaultMaxSize))
		g.Cloner = Cloner{
			Paths:     []string{c.CrossplanePackageCRDDir},
			Reference: c.GitBranchOriginMain,
			RepoURL:   c.RepoURL,
			Storage:   storage,
			Worktree:  filesystemfs,
		}

		w, err = g.Clone()
		if err != nil {
			return err
		}

		log.Debug("git clone complete")
	} else {
		log.Debug("using existing git repo", "directory", c.RepositoryDir)

		w = &git.Worktree{Filesystem: filesystemfs}
		g.Worktree = w.Filesystem
	}

	log.Debug("examining CRD files", "directory", c.CrossplanePackageCRDDir)

	filter, err := ExamineFieldFromCRDVersions(w.Filesystem)
	if err != nil {
		return err
	}

	var out *os.File
	if c.OutputFile == "" {
		out = os.Stdout
	} else {
		out, err = os.Create(c.OutputFile)
		if err != nil {
			return err
		}
	}

	log.Debug("rendering template", "location", out.Name())

	return render.Render(out, filter, render.AWSResourceFilterTemplate)
}

func main() {
	ctx := kong.Parse(&CLI{}, kong.Description("Generate filter lists from Provider CRDs"))
	ctx.FatalIfErrorf(ctx.Run())
}

func FetchRepository() {
}

// Clone performs a sparse checkout of a git repository.
func (g Generater) Clone() (*git.Worktree, error) {
	g.Logger.Info("cloning repo", "url", g.RepoURL)

	r, err := git.Clone(g.Storage, g.Worktree, &git.CloneOptions{
		NoCheckout: true,
		URL:        g.RepoURL,
	})
	if err != nil {
		return nil, err
	}

	wt, err := r.Worktree()
	if err != nil {
		g.Logger.Info("error: unable to get worktree", "error", err)
		return nil, errors.Wrapf(err, "unable to get worktree")
	}

	err = wt.Checkout(&git.CheckoutOptions{
		Branch:                    plumbing.ReferenceName(g.Reference),
		SparseCheckoutDirectories: g.Paths,
	})
	if err != nil {
		g.Logger.Info("error: unable to checkout paths", "error", err)
		return nil, errors.Wrapf(err, "unable to checkout paths")
	}

	return wt, nil
}
