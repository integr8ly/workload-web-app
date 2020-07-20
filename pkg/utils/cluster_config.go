package utils

import (
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetClusterConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		if err == rest.ErrNotInCluster {
			// fall back to kubeconfig
			kubeconfig := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
			if kubeconfig == "" {
				// fall back to recomaned kubeconfig location
				kubeconfig = clientcmd.RecommendedHomeFile
			}

			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return config, nil
}
