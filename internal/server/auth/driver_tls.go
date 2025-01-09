package auth

import (
	"context"
	"net/http"

	"github.com/lxc/incus/v6/shared/api"
	"github.com/lxc/incus/v6/shared/logger"
)

// TLS represents a TLS authorizer.
type TLS struct {
	commonAuthorizer

	certificateFingerprints []string
}

func (t *TLS) load(ctx context.Context, certificateFingerprints []string, opts map[string]any) error {
	t.certificateFingerprints = certificateFingerprints
	return nil
}

// CheckPermission returns an error if the user does not have the given Entitlement on the given Object.
func (t *TLS) CheckPermission(ctx context.Context, r *http.Request, object Object, entitlement Entitlement) error {
	details, err := t.requestDetails(r)
	if err != nil {
		return api.StatusErrorf(http.StatusForbidden, "Failed to extract request details: %v", err)
	}

	// Always allow full access via local unix socket.
	if details.Protocol == "unix" {
		return nil
	}

	if details.Protocol != api.AuthenticationMethodTLS {
		t.logger.Warn("Authentication protocol is not compatible with authorization driver", logger.Ctx{"protocol": details.Protocol})
		// Return nil. If the server has been configured with an authentication method but no associated authorization driver,
		// the default is to give these authenticated users admin privileges.
		return nil
	}

	for _, cert := range t.certificateFingerprints {
		if cert == details.Username {
			// Authentication via TLS gives full, unrestricted access.
			return nil
		}
	}

	return api.StatusErrorf(http.StatusForbidden, "Client certificate not found")
}
