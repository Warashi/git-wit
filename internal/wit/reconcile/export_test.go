package reconcile

import "context"

func FindBrokenSymlinks(root string) ([]string, error) {
	return findBrokenSymlinks(root)
}

func ResolveWorktreeOwner(ctx context.Context, worktreePath string) error {
	return resolveWorktreeOwner(ctx, worktreePath)
}
