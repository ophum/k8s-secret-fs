package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"path/filepath"
	"syscall"

	myfs "github.com/ophum/k8s-secret-fs/pkg/fs"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	clientset  kubernetes.Interface
	namespace  string
	secretName string
)

func init() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.StringVar(&namespace, "namespace", "default", "namespace")
	flag.StringVar(&secretName, "secret", "default", "secret")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
}

func getSecret(ctx context.Context) *v1.Secret {
	secretClient := clientset.CoreV1().Secrets(namespace)
	secret, err := secretClient.Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	return secret
}

type k8sSecretRoot struct {
	fs.Inode
}

var _ = (fs.NodeOnAdder)((*k8sSecretRoot)(nil))
var _ = (fs.NodeReaddirer)((*k8sSecretRoot)(nil))

func (r *k8sSecretRoot) OnAdd(ctx context.Context) {
	secret := getSecret(ctx)
	for name, _ := range secret.Data {
		ch := r.Inode.NewPersistentInode(ctx, &k8sSecretFile{key: name}, fs.StableAttr{Mode: syscall.S_IFREG})
		r.Inode.AddChild(name, ch, true)
	}
}

func (r *k8sSecretRoot) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	secret := getSecret(ctx)
	var entries []fuse.DirEntry
	for name := range secret.Data {
		ch := r.Inode.NewPersistentInode(ctx, &k8sSecretFile{key: name}, fs.StableAttr{Mode: syscall.S_IFREG})
		r.Inode.AddChild(name, ch, true)
		entries = append(entries, fuse.DirEntry{
			Name: name,
			Mode: syscall.S_IFREG,
			Ino:  ch.StableAttr().Ino,
		})
	}
	return fs.NewListDirStream(entries), 0
}

type k8sSecretFile struct {
	fs.Inode
	key string
}

var _ = (fs.NodeOpener)((*k8sSecretFile)(nil))

func (f *k8sSecretFile) Open(ctx context.Context, openFlags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	if fuseFlags&(syscall.O_RDWR|syscall.O_WRONLY) != 0 {
		return nil, 0, syscall.EROFS
	}

	secret := getSecret(ctx)
	fh = myfs.NewBytesFileHandle(secret.Data[f.key])
	return fh, fuse.FOPEN_DIRECT_IO, 0
}

// This demonstrates how to build a file system in memory. The
// read/write logic for the file is provided by the MemRegularFile type.
func main() {
	// This is where we'll mount the FS
	mntDir, _ := ioutil.TempDir("", "")

	root := &k8sSecretRoot{}
	server, err := fs.Mount(mntDir, root, &fs.Options{
		MountOptions: fuse.MountOptions{Debug: true},
	})
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Mounted on %s", mntDir)
	log.Printf("Unmount by calling 'fusermount -u %s'", mntDir)

	// Wait until unmount before exiting
	server.Wait()
}
