package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	kclient   kubernetes.Interface
	namespace string
	name      string
}

func NewClient(kubeconfig, namespace, name string) (*Client, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &Client{clientset, namespace, name}, nil
}

func (c *Client) GetSecret(ctx context.Context) (*v1.Secret, error) {
	secretClient := c.kclient.CoreV1().Secrets(c.namespace)
	secret, err := secretClient.Get(ctx, c.name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return secret, nil
}
