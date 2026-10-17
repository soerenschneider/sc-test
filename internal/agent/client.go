package agent

import (
	"context"
	"errors"
	"net/http"

	"github.com/soerenschneider/sc-agent/pkg/api"
)

var _ api.ClientInterface = &ScAgentClient{}

type ScAgentClient struct {
	client api.ClientInterface
}

func NewClient(client api.ClientInterface) (*ScAgentClient, error) {
	if client == nil {
		return nil, errors.New("nil client passed")
	}

	return &ScAgentClient{
		client: client,
	}, nil
}

func (a ScAgentClient) CertsAcmeGetCertificates(ctx context.Context, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.CertsAcmeGetCertificates(ctx, reqEditors...)
}

func (a ScAgentClient) CertsAcmeGetCertificate(ctx context.Context, id string, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.CertsAcmeGetCertificate(ctx, id, reqEditors...)
}

func (a ScAgentClient) CertsSshGetCertificates(ctx context.Context, params *api.CertsSshGetCertificatesParams, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.CertsSshGetCertificates(ctx, params, reqEditors...)
}

func (a ScAgentClient) CertsSshGetCertificate(ctx context.Context, id string, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.CertsSshGetCertificate(ctx, id, reqEditors...)
}

func (a ScAgentClient) InfoGetComponents(ctx context.Context, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.InfoGetComponents(ctx, reqEditors...)
}

func (a ScAgentClient) CertsSshPostIssueRequests(ctx context.Context, params *api.CertsSshPostIssueRequestsParams, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.CertsSshPostIssueRequests(ctx, params, reqEditors...)
}

func (a ScAgentClient) CertsX509GetCertificatesList(ctx context.Context, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.CertsX509GetCertificatesList(ctx, reqEditors...)
}

func (a ScAgentClient) CertsX509PostIssueRequests(ctx context.Context, params *api.CertsX509PostIssueRequestsParams, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.CertsX509PostIssueRequests(ctx, params, reqEditors...)
}

func (a ScAgentClient) CertsX509GetCertificate(ctx context.Context, id string, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.CertsX509GetCertificate(ctx, id, reqEditors...)
}

func (a ScAgentClient) K0sPostAction(ctx context.Context, params *api.K0sPostActionParams, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.K0sPostAction(ctx, params, reqEditors...)
}

func (a ScAgentClient) LibvirtPostDomainAction(ctx context.Context, domain string, params *api.LibvirtPostDomainActionParams, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.LibvirtPostDomainAction(ctx, domain, params, reqEditors...)
}

func (a ScAgentClient) PowerPostAction(ctx context.Context, params *api.PowerPostActionParams, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.PowerPostAction(ctx, params, reqEditors...)
}

func (a ScAgentClient) PowerRebootManagerPostStatus(ctx context.Context, params *api.PowerRebootManagerPostStatusParams, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.PowerRebootManagerPostStatus(ctx, params, reqEditors...)
}

func (a ScAgentClient) PowerRebootManagerGetStatus(ctx context.Context, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.PowerRebootManagerGetStatus(ctx, reqEditors...)
}

func (a ScAgentClient) ReplicationGetHttpItemsList(ctx context.Context, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.ReplicationGetHttpItemsList(ctx, reqEditors...)
}

func (a ScAgentClient) ReplicationGetHttpItem(ctx context.Context, id string, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.ReplicationGetHttpItem(ctx, id, reqEditors...)
}

func (a ScAgentClient) ReplicationGetSecretsItemsList(ctx context.Context, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.ReplicationGetSecretsItemsList(ctx, reqEditors...)
}

func (a ScAgentClient) ReplicationGetSecretsItem(ctx context.Context, id string, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.ReplicationGetSecretsItem(ctx, id, reqEditors...)
}

func (a ScAgentClient) ReplicationPostSecretsRequests(ctx context.Context, params *api.ReplicationPostSecretsRequestsParams, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.ReplicationPostSecretsRequests(ctx, params, reqEditors...)
}

func (a ScAgentClient) ServicesUnitStatusPut(ctx context.Context, unit string, params *api.ServicesUnitStatusPutParams, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.ServicesUnitStatusPut(ctx, unit, params, reqEditors...)
}

func (a ScAgentClient) PackagesInstalledGet(ctx context.Context, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.PackagesInstalledGet(ctx, reqEditors...)
}

func (a ScAgentClient) PackagesUpdatesGet(ctx context.Context, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.PackagesUpdatesGet(ctx, reqEditors...)
}

func (a ScAgentClient) PackagesUpgradeRequestsPost(ctx context.Context, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.PackagesUpgradeRequestsPost(ctx, reqEditors...)
}

func (a ScAgentClient) ServicesUnitLogsGet(ctx context.Context, unit string, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.ServicesUnitLogsGet(ctx, unit, reqEditors...)
}

func (a ScAgentClient) WolPostMessage(ctx context.Context, alias string, reqEditors ...api.RequestEditorFn) (*http.Response, error) {
	return a.client.WolPostMessage(ctx, alias, reqEditors...)
}
