package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/ophum/k8s-secret-fs/pkg/fs"
	"github.com/ophum/k8s-secret-fs/pkg/k8s"

	"k8s.io/client-go/util/homedir"
)

var (
	client     *k8s.Client
	namespace  string
	secretName string
	mountPoint string
)

func init() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	namespacedSecretName := flag.Arg(0)
	splited := strings.Split(namespacedSecretName, "/")
	switch len(splited) {
	case 1:
		namespace = "default"
		secretName = splited[0]
	case 2:
		namespace = splited[0]
		secretName = splited[1]
	default:
		log.Panic("invalid namespace/secretName")
	}

	mountPoint = flag.Arg(1)

	var err error
	client, err = k8s.NewClient(*kubeconfig, namespace, secretName)
	if err != nil {
		log.Panic(err)
	}
}

func main() {
	secretFS := fs.NewSecretFS(client)
	server, err := secretFS.Mount(mountPoint)
	if err != nil {
		log.Panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		server.Unmount()
	}()

	server.Wait()
}
