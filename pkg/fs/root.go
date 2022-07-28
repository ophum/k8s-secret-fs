package fs

import (
	"context"
	"log"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/ophum/k8s-secret-fs/pkg/k8s"
)

type SecretRoot struct {
	fs.Inode
	client *k8s.Client
}

var _ = (fs.NodeOnAdder)((*SecretRoot)(nil))
var _ = (fs.NodeReaddirer)((*SecretRoot)(nil))

func (r *SecretRoot) OnAdd(ctx context.Context) {
	secret, err := r.client.GetSecret(ctx)
	if err != nil {
		log.Panic(err)
	}

	for name, _ := range secret.Data {
		ch := r.Inode.NewPersistentInode(ctx, &SecretFile{key: name, client: r.client}, fs.StableAttr{Mode: syscall.S_IFREG})
		r.Inode.AddChild(name, ch, true)
	}
}

func (r *SecretRoot) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	secret, err := r.client.GetSecret(ctx)
	if err != nil {
		log.Panic(err)
	}

	var entries []fuse.DirEntry
	for name := range secret.Data {
		ch := r.Inode.NewPersistentInode(ctx, &SecretFile{key: name, client: r.client}, fs.StableAttr{Mode: syscall.S_IFREG})
		r.Inode.AddChild(name, ch, true)
		entries = append(entries, fuse.DirEntry{
			Name: name,
			Mode: syscall.S_IFREG,
			Ino:  ch.StableAttr().Ino,
		})
	}
	return fs.NewListDirStream(entries), 0
}
