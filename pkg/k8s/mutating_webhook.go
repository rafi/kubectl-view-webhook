/*
Copyright © 2020 Trendyol Tech

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

package k8s

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"github.com/Trendyol/kubectl-view-webhook/pkg/printer"
	"k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"time"
)

type MutatingWebHookClient struct {
	client  *kubernetes.Clientset
	context context.Context
}

// NewMutatingWebHookClient constructs a new MutatingWebHookClient with the specified output
// of *kubernetes.Clientset
func NewMutatingWebHookClient(client *kubernetes.Clientset) *MutatingWebHookClient {
	return &MutatingWebHookClient{
		client:  client,
		context: context.Background(),
	}
}

func (w *MutatingWebHookClient) Run(args []string) (*printer.PrintModel, error) {
	c := w.client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations()
	var items []printer.PrintItem
	ncList, err := w.client.CoreV1().Namespaces().List(w.context, metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}
	if len(args) == 1 {
		mutatingWebhookConfigurationList, err := c.List(w.context, metaV1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for _, mwc := range mutatingWebhookConfigurationList.Items {
			items = w.fillPrintItems(mwc, items, ncList)
		}
	} else {
		mutatingWebhookConfiguration, err := c.Get(w.context, args[1], metaV1.GetOptions{})
		if err != nil {
			return nil, err
		}

		items = w.fillPrintItems(*mutatingWebhookConfiguration, items, ncList)
	}

	return &printer.PrintModel{
		Items: items,
	}, nil
}

func (w *MutatingWebHookClient) fillPrintItems(mwc v1beta1.MutatingWebhookConfiguration, items []printer.PrintItem, ncList *v1.NamespaceList) []printer.PrintItem {
	item := printer.PrintItem{
		Kind: "MutatingWebhookConfiguration",
		Name: mwc.Name, //TODO: typeMeta nil
	}
	for _, webhook := range mwc.Webhooks {
		var operations, resources, activeNamespaces []string

		if webhook.NamespaceSelector != nil {
			for _, ns := range ncList.Items {
				available := false
				for k, v := range webhook.NamespaceSelector.MatchLabels {
					if ns.Labels[k] == v {
						available = true
					}
				}
				if available {
					activeNamespaces = append(activeNamespaces, ns.Name)
				}
			}
		}

		item.WebhookName = webhook.Name
		for _, rule := range webhook.Rules {

			for _, op := range rule.Operations {
				operations = append(operations, string(op))
			}

			for _, r := range rule.Resources {
				resources = append(resources, r)
			}
		}

		item.Operations = operations
		item.Resources = resources
		item.ValidUntil = retrieveValidDateCount(webhook.ClientConfig.CABundle)
		items = append(items, item)
	}
	return items
}

func retrieveValidDateCount(certificate []byte) int64 {
	block, _ := pem.Decode(certificate)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Fatalf("x509.ParseCertificate - error occurred, detail: %v", err)
	}
	return int64(cert.NotAfter.Sub(time.Now()).Hours() / 24)
}
