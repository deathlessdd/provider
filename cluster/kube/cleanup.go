package kube

import (
	"context"

	mani "github.com/akash-network/akash-api/go/manifest/v2beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes"

	mtypes "github.com/akash-network/akash-api/go/node/market/v1beta4"

	"github.com/akash-network/provider/cluster/kube/builder"
)

func cleanupStaleResources(ctx context.Context, kc kubernetes.Interface, lid mtypes.LeaseID, group *mani.Group) error {
	ns := builder.LidNS(lid)

	// build label selector for objects not in current manifest group
	svcnames := make([]string, 0, len(group.Services))
	for _, svc := range group.Services {
		svcnames = append(svcnames, svc.Name)
	}

	req1, err := labels.NewRequirement(builder.AkashManifestServiceLabelName, selection.NotIn, svcnames)
	if err != nil {
		return err
	}
	req2, err := labels.NewRequirement(builder.AkashServiceTarget, selection.Equals, []string{"true"})
	if err != nil {
		return err
	}

	req3, err := labels.NewRequirement(builder.AkashManagedLabelName, selection.NotIn, []string{builder.AkashMetalLB})
	if err != nil {
		return err
	}

	selector := labels.NewSelector().Add(*req1).Add(*req2).Add(*req3).String()

	// delete stale deployments
	if err := kc.AppsV1().Deployments(ns).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		return err
	}

	// delete stale deployments
	if err := kc.AppsV1().StatefulSets(ns).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		return err
	}

	// delete stale services (no DeleteCollection)
	services, err := kc.CoreV1().Services(ns).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return err
	}
	for _, svc := range services.Items {
		if err := kc.CoreV1().Services(ns).Delete(ctx, svc.Name, metav1.DeleteOptions{}); err != nil {
			return err
		}
	}

	return nil
}
