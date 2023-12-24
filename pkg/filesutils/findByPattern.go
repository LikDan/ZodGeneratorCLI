package filesutils

import (
	"os"
)

func FindByFunc(dirPath string, recursive bool, match func(path string) bool) ([]string, error) {
	dir, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var filenames []string
	for _, file := range dir {
		fullPath := dirPath + "/" + file.Name()

		if file.IsDir() {
			if recursive {
				filenames_, err := FindByFunc(fullPath, true, match)
				if err != nil {
					return nil, err
				}

				filenames = append(filenames, filenames_...)
			}
			continue
		}

		if match(fullPath) {
			filenames = append(filenames, fullPath)
		}
	}

	return filenames, err
}
