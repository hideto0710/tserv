/*
Copyright © 2020 HIDETO INAMURA <h.inamura0710@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package action

import (
	"archive/tar"
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	torchstandTypes "github.com/hideto0710/torchstand/pkg/types"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type Archiver struct {
	ref          *torchstandTypes.Ref
	registryPath string
}

func NewArchiver(ref *torchstandTypes.Ref, registryPath string) *Archiver {
	return &Archiver{
		ref: ref, registryPath: registryPath,
	}
}

func (a *Archiver) Archive(dest string) error {
	zipFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer zipFile.Close()
	w := zip.NewWriter(zipFile)
	defer w.Close()

	if err := a.copyConfig(w); err != nil {
		return err
	}
	if err := a.copyPyTorchModel(w); err != nil {
		return err
	}
	if err := a.copyContents(w); err != nil {
		return err
	}
	return nil
}

func (a *Archiver) copyConfig(w *zip.Writer) error {
	return addFileToZip(w, a.blobPath(a.ref.Config), marFilePath)
}

func (a *Archiver) copyPyTorchModel(w *zip.Writer) error {
	configFile, err := os.Open(a.blobPath(a.ref.Config))
	if err != nil {
		return err
	}
	m := &torchstandTypes.Manifest{}
	if err := json.NewDecoder(configFile).Decode(&m); err != nil {
		return err
	}
	return addFileToZip(w, a.blobPath(a.ref.PyTorchModel), m.Model.SerializedFile)
}

func (a *Archiver) copyContents(w *zip.Writer) error {
	contentFile, err := os.Open(a.blobPath(a.ref.Content))
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(contentFile)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := header.Name
		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			f, err := w.Create(name)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tarReader); err != nil {
				return err
			}
		default:
			continue
		}
	}
	return nil
}

func (a *Archiver) blobPath(desc v1.Descriptor) string {
	return filepath.Join(a.registryPath, "blobs", desc.Digest.Algorithm().String(), desc.Digest.Hex())
}
