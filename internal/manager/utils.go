package manager

import (
	"fmt"
	"io"
	"os"
)

func copyFile(sourcePath, targetPath string) error {
	//nolint:gosec
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() {
		if cerr := source.Close(); cerr != nil {
			fmt.Printf("failed to close source file: %v\n", cerr)
		}
	}()

	//nolint:gosec
	target, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer func() {
		if cerr := target.Close(); cerr != nil {
			fmt.Printf("failed to close target file: %v\n", cerr)
		}
	}()

	_, err = io.Copy(target, source)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	return nil
}
