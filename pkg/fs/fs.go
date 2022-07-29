package fs

import (
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/ophum/k8s-secret-fs/pkg/k8s"
)

type SecretFS struct {
	root *SecretRoot
}

func NewSecretFS(client *k8s.Client) *SecretFS {
	return &SecretFS{
		root: &SecretRoot{
			client: client,
		},
	}
}

func (f *SecretFS) Mount(mountPoint string) (*fuse.Server, error) {
	return fs.Mount(mountPoint, f.root, &fs.Options{
		MountOptions: fuse.MountOptions{
			FsName: "k8s-secret-fs",
			Name:   "k8s-secret-fs",
		},
	})
}
