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
	"gopkg.in/yaml.v3"

	"k8s.io/client-go/util/homedir"
)

var (
	client *k8s.Client
	config *Config
)

type Config struct {
	Kubeconfig string `yaml:"kubeconfig"`
	Namespace  string `yaml:"namespace"`
	SecretName string `yaml:"secretName"`
	MountPoint string `yaml:"mountPoint"`
}

func init() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "-config config.yaml")

	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config = &Config{}
	if configPath == "" {
		config.Kubeconfig = *kubeconfig
		namespacedSecretName := flag.Arg(0)
		splited := strings.Split(namespacedSecretName, "/")
		switch len(splited) {
		case 1:
			config.Namespace = "default"
			config.SecretName = splited[0]
		case 2:
			config.Namespace = splited[0]
			config.SecretName = splited[1]
		default:
			log.Panic("invalid namespace/secretName")
		}
		config.MountPoint = flag.Arg(1)
	} else {
		f, err := os.Open(configPath)
		if err != nil {
			log.Panic(err)
		}
		defer f.Close()

		if err := yaml.NewDecoder(f).Decode(config); err != nil {
			log.Panic(err)
		}
	}

	var err error
	client, err = k8s.NewClient(config.Kubeconfig, config.Namespace, config.SecretName)
	if err != nil {
		log.Panic(err)
	}
}

func main() {
	secretFS := fs.NewSecretFS(client)
	server, err := secretFS.Mount(config.MountPoint)
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
