package fs

import (
	"context"
	"log"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/ophum/k8s-secret-fs/pkg/k8s"
)

type SecretFile struct {
	fs.Inode
	key       string
	client    *k8s.Client
	namespace string
	name      string
}

var _ = (fs.NodeOpener)((*SecretFile)(nil))

func (f *SecretFile) Open(ctx context.Context, openFlags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	if fuseFlags&(syscall.O_RDWR|syscall.O_WRONLY) != 0 {
		return nil, 0, syscall.EROFS
	}

	secret, err := f.client.GetSecret(ctx)
	if err != nil {
		log.Panic(err)
	}

	fh = NewBytesFileHandle(secret.Data[f.key])
	return fh, fuse.FOPEN_DIRECT_IO, 0
}
