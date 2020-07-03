package checks

import (
	"fmt"
	"os"
	"strings"
	"time"

	enmasseV1beta1 "github.com/enmasseproject/enmasse/pkg/apis/enmasse/v1beta1"
	enmasseClient "github.com/enmasseproject/enmasse/pkg/client/clientset/versioned"
	"github.com/integr8ly/workload-web-app/pkg/counters"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const amqCRUDService = "amq_crud"

type AMQCRUDChecks struct {
	namespace string
	client    *enmasseClient.Clientset
}

func NewAMQCRUDChecks(namespace string) (*AMQCRUDChecks, error) {
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

	enmasse, err := enmasseClient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &AMQCRUDChecks{namespace: namespace, client: enmasse}, nil
}

func (c *AMQCRUDChecks) RunForever() {
	counters.InitCounters(amqCRUDService, "")
	for {
		start := time.Now()
		err := c.run()
		counters.ServiceTotalRequestsCounter.WithLabelValues(amqCRUDService, "").Inc()
		if err != nil {
			counters.UpdateErrorMetricsForService(amqCRUDService, "", err.Error(), time.Now().Sub(start).Seconds())
		} else {
			counters.UpdateSuccessMetricsForService(amqCRUDService, "")
		}
	}
}

func (c *AMQCRUDChecks) run() error {

	addressSpaceName := "workload-app-crud"
	addressName := fmt.Sprintf("%s.queue-requests", addressSpaceName)

	errors := []string{}

	reportError := func(err error) {
		log.Error(err)
		errors = append(errors, err.Error())
	}

	// Create a new AddressSpace
	err := c.createAddressSpace(addressSpaceName)
	if err != nil {
		reportError(fmt.Errorf("create AddressSpace failed with error: %s", err))
	}

	// Create a new Address
	err = c.createAddress(addressName)
	if err != nil {
		reportError(fmt.Errorf("create Address failed with error: %s", err))
	}

	// Delete the Address
	err = c.deleteAddress(addressName)
	if err != nil {
		reportError(fmt.Errorf("delete Address failed with error: %s", err))
	}

	// Delete the AddressSpace
	err = c.deleteAddressSpace(addressSpaceName)
	if err != nil {
		reportError(fmt.Errorf("delete AddressSpace failed with error: %s", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "\n"))
	}
	return nil
}

func (c *AMQCRUDChecks) createAddressSpace(name string) error {

	log := log.WithField("name", name)
	log.Debug("create AddressSpace")

	a := &enmasseV1beta1.AddressSpace{
		TypeMeta: v1.TypeMeta{
			APIVersion: "enmasse.io/v1beta1",
			Kind:       "AddressSpace",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: enmasseV1beta1.AddressSpaceSpec{
			Type: "standard",
			Plan: "standard-unlimited",
			AuthenticationService: &enmasseV1beta1.AuthenticationService{
				Name: "none-authservice",
			},
		},
	}
	_, err := c.client.EnmasseV1beta1().
		AddressSpaces(c.namespace).
		Create(a)
	if err != nil {
		return err
	}

	err = wait.Poll(30*time.Second, 5*time.Minute, func() (bool, error) {
		a, err := c.client.EnmasseV1beta1().
			AddressSpaces(c.namespace).
			Get(name, v1.GetOptions{})
		if err != nil {
			return false, err
		}

		if !a.Status.IsReady {
			log.WithField("phase", a.Status.Phase).Debug("wating for AddressSpace to be ready")
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	log.Debug("created AddressSpace")
	return nil
}

func (c *AMQCRUDChecks) deleteAddressSpace(name string) error {

	log := log.WithField("name", name)
	log.Debug("delete AddressSpace")

	err := c.client.EnmasseV1beta1().
		AddressSpaces(c.namespace).
		Delete(name, &v1.DeleteOptions{})
	if err != nil {
		return err
	}

	err = wait.Poll(5*time.Second, time.Minute, func() (bool, error) {

		_, err := c.client.EnmasseV1beta1().
			AddressSpaces(c.namespace).
			Get(name, v1.GetOptions{})
		if err != nil {
			if isNotFound(err) {
				// addressspace deleted
				return true, nil
			}
			// failed to retrieve the addressspace for any other reason
			return false, err
		}

		// waiting for the addressspace to be deleted
		log.Debug("waiting for AddressSpace to be deleted")
		return false, nil
	})
	if err != nil {
		return err
	}

	log.Debug("deleted AddressSpace")
	return nil
}

func (c *AMQCRUDChecks) createAddress(name string) error {

	log := log.WithField("name", name)
	log.Debug("create Address")

	a := &enmasseV1beta1.Address{
		TypeMeta: v1.TypeMeta{
			APIVersion: "enmasse.io/v1beta1",
			Kind:       "Address",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: enmasseV1beta1.AddressSpec{
			Address: "queue-requests",
			Type:    "queue",
			Plan:    "standard-small-queue",
		},
	}
	_, err := c.client.EnmasseV1beta1().
		Addresses(c.namespace).
		Create(a)
	if err != nil {
		return err
	}

	err = wait.Poll(30*time.Second, 5*time.Minute, func() (bool, error) {
		a, err := c.client.EnmasseV1beta1().
			Addresses(c.namespace).
			Get(name, v1.GetOptions{})
		if err != nil {
			return false, err
		}

		if !a.Status.IsReady {
			log.WithField("phase", a.Status.Phase).Debug("wating for Address to be ready")
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	log.Debug("created Address")
	return nil

}

func (c *AMQCRUDChecks) deleteAddress(name string) error {

	log := log.WithField("name", name)
	log.Debug("delete Address")

	err := c.client.EnmasseV1beta1().
		Addresses(c.namespace).
		Delete(name, &v1.DeleteOptions{})
	if err != nil {
		return err
	}

	err = wait.Poll(5*time.Second, time.Minute, func() (bool, error) {

		_, err := c.client.EnmasseV1beta1().
			Addresses(c.namespace).
			Get(name, v1.GetOptions{})
		if err != nil {
			if isNotFound(err) {
				return true, nil
			}
			return false, err
		}

		log.Debug("waiting for Address to be deleted")
		return false, nil
	})
	if err != nil {
		return err
	}

	log.Debug("deleted Address")
	return nil
}

func isNotFound(err error) bool {
	if s, ok := err.(*errors.StatusError); ok {
		if s.ErrStatus.Reason == "NotFound" {
			return true
		}
	}
	return false
}
