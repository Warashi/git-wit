package sync

import "io/fs"

type CopierForTest struct {
	copier copier
}

func NewCopierForTest(
	dirClone func(srcPath string, destPath string) (bool, error),
	fileClone func(srcPath string, destPath string, mode fs.FileMode) (bool, error),
) CopierForTest {
	return CopierForTest{
		copier: copier{
			cloneDir:  dirClone,
			cloneFile: fileClone,
		},
	}
}

func (c CopierForTest) CopyDir(srcPath string, destPath string, mode fs.FileMode) error {
	return c.copier.copyDir(srcPath, destPath, mode)
}

func (c CopierForTest) CopyFile(srcPath string, destPath string, mode fs.FileMode) error {
	return c.copier.copyFile(srcPath, destPath, mode)
}
