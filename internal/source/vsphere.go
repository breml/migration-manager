/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0

Code originally copied from https://github.com/vmware-tanzu/sources-for-knative/blob/main/pkg/vsphere/client.go,
but has been subsequently modified to extend capabilities.
*/

package source

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/url"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/session/keepalive"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/soap"
)

const keepaliveInterval = 5 * time.Minute // vCenter APIs keep-alive

func soapWithKeepalive(ctx context.Context, clientURL *url.URL, insecure bool, additionalRootCert *tls.Certificate) (*govmomi.Client, error) {
	soapClient := soap.NewClient(clientURL, insecure)

	if additionalRootCert != nil {
		x509cert, err := x509.ParseCertificate(additionalRootCert.Certificate[0])
		if err != nil {
			return nil, err
		}

		certpool := x509.NewCertPool()
		certpool.AddCert(x509cert)
		soapClient.DefaultTransport().TLSClientConfig.RootCAs = certpool
	}

	vimClient, err := vim25.NewClient(ctx, soapClient)
	if err != nil {
		return nil, err
	}

	vimClient.RoundTripper = keepalive.NewHandlerSOAP(vimClient.RoundTripper, keepaliveInterval, soapKeepAliveHandler(ctx, vimClient))

	// explicitly create session to activate keep-alive handler via Login
	m := session.NewManager(vimClient)
	err = m.Login(ctx, clientURL.User)
	if err != nil {
		return nil, err
	}

	c := govmomi.Client{
		Client:         vimClient,
		SessionManager: m,
	}

	return &c, nil
}

func soapKeepAliveHandler(ctx context.Context, c *vim25.Client) func() error {
	// logger := logging.FromContext(ctx).With("rpc", "keepalive")

	return func() error {
		// logger.Info("Executing SOAP keep-alive handler")
		_, err := methods.GetCurrentTime(ctx, c)
		if err != nil {
			return err
		}

		// logger.Infof("vCenter current time: %s", t.String())
		return nil
	}
}
