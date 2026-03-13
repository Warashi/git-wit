package git

// WithRepoDir returns a copy of the runner using repoDir.
func (r Runner) WithRepoDir(repoDir string) Runner {
	r.repoDir = repoDir

	return r
}
