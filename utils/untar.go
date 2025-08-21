package utils

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
)

func UnTar(
	basepath string,
	tarbuf []byte,
) error {
	if err := os.MkdirAll(basepath, 0700); err != nil && !os.IsExist(err) {
		return fmt.Errorf("make directory failed, %w", err)
	}

	if err := os.RemoveAll(basepath); err != nil {
		return fmt.Errorf("remove files from previous run failed, %w", err)
	}

	buf := bytes.NewBuffer(tarbuf)
	tr := tar.NewReader(buf)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		b, err := io.ReadAll(tr)
		if err != nil {
			return fmt.Errorf("read file from tar failed, %w", err)
		}

		filepath := path.Join(basepath, hdr.Name)
		err = os.MkdirAll(path.Dir(filepath), 0700)
		if err != nil && !os.IsExist(err) {
			return fmt.Errorf("make directory for file failed, %w", err)
		}

		if err = os.WriteFile(filepath, b, os.FileMode(hdr.Mode)); err != nil {
			return fmt.Errorf("write to file failed, %w", err)
		}
	}

	return nil
}
