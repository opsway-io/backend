package k8s

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Service interface {
	UpsertIngress(ctx context.Context, domain string) error
	DeleteIngress(ctx context.Context, domain string) error
}

type ServiceImpl struct {
	client *kubernetes.Clientset
	logger *logrus.Entry
}

func NewService(logger *logrus.Logger) Service {
	l := logger.WithField("module", "k8s")

	config, err := rest.InClusterConfig()
	if err != nil {
		l.WithError(err).Warn("Running outside of Kubernetes cluster; dynamic auto-TLS ingress generation will be skipped.")
		return &ServiceImpl{
			logger: l,
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		l.WithError(err).Warn("Failed to create Kubernetes clientset. Dynamic auto-TLS ingress generation will be disabled.")
		return &ServiceImpl{
			logger: l,
		}
	}

	return &ServiceImpl{
		client: clientset,
		logger: l,
	}
}

func (s *ServiceImpl) UpsertIngress(ctx context.Context, domain string) error {
	if s.client == nil {
		return nil
	}

	ingressName := generateIngressName(domain)
	namespace := "status-page"

	pathType := networkingv1.PathTypePrefix

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressName,
			Namespace: namespace,
			Annotations: map[string]string{
				"cert-manager.io/cluster-issuer":       "letsencrypt",
				"konghq.com/protocol":                  "https",
				"konghq.com/http-redirect-status-code": "308",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: stringPtr("kong"),
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{domain},
					SecretName: fmt.Sprintf("%s-tls", ingressName),
				},
			},
			Rules: []networkingv1.IngressRule{
				{
					Host: domain,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/v1",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "backend-api",
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "status-page",
											Port: networkingv1.ServiceBackendPort{
												Name: "http",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ingresses := s.client.NetworkingV1().Ingresses(namespace)

	existing, err := ingresses.Get(ctx, ingressName, metav1.GetOptions{})
	if err == nil && existing != nil {
		ingress.ResourceVersion = existing.ResourceVersion
		_, err = ingresses.Update(ctx, ingress, metav1.UpdateOptions{})
		if err != nil {
			s.logger.WithError(err).Error("Failed to update ingress")
			return err
		}
		s.logger.Infof("Updated ingress for custom domain %s", domain)
		return nil
	}

	_, err = ingresses.Create(ctx, ingress, metav1.CreateOptions{})
	if err != nil {
		s.logger.WithError(err).Error("Failed to create ingress")
		return err
	}
	s.logger.Infof("Created ingress for custom domain %s", domain)

	return nil
}

func (s *ServiceImpl) DeleteIngress(ctx context.Context, domain string) error {
	if s.client == nil {
		return nil
	}

	ingressName := generateIngressName(domain)
	namespace := "status-page"

	ingresses := s.client.NetworkingV1().Ingresses(namespace)
	err := ingresses.Delete(ctx, ingressName, metav1.DeleteOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		s.logger.WithError(err).Error("Failed to delete ingress")
		return err
	}

	s.logger.Infof("Deleted ingress for custom domain %s", domain)
	return nil
}

func generateIngressName(domain string) string {
	clean := strings.ReplaceAll(domain, ".", "-")
	return fmt.Sprintf("status-page-custom-%s", clean)
}

func stringPtr(s string) *string {
	return &s
}
