package sqlite_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	dbschema "github.com/FuturFusion/migration-manager/internal/db"
	dbdriver "github.com/FuturFusion/migration-manager/internal/db/sqlite"
	"github.com/FuturFusion/migration-manager/internal/migration"
	"github.com/FuturFusion/migration-manager/internal/migration/repo/sqlite"
	"github.com/FuturFusion/migration-manager/shared/api"
)

func TestTargetDatabaseActions(t *testing.T) {
	incusTargetA := migration.Target{Name: "Target A", TargetType: api.TARGETTYPE_INCUS, Properties: []byte(`{"endpoint": "https://localhost:6443", "tls_client_key", "PRIVATE_KEY", "tls_client_cert": "PUBLIC_CERT"}`)}
	incusTargetB := migration.Target{Name: "Target B", TargetType: api.TARGETTYPE_INCUS, Properties: []byte(`{"endpoint": "https://incus.local:6443", "oidc_tokens": {"access_token":"encoded_content","token_type":"Bearer","refresh_token":"encoded_content","expiry":"2024-11-06T14:23:16.439206188Z","IDTokenClaims":null,"IDToken":"encoded_content"}}`)}
	incusTargetC := migration.Target{Name: "Target C", TargetType: api.TARGETTYPE_INCUS, Properties: []byte(`{"endpoint": "https://10.10.10.10:6443", "insecure": true}`)}

	ctx := context.Background()

	// Create a new temporary database.
	tmpDir := t.TempDir()
	db, err := dbdriver.Open(tmpDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		err = db.Close()
		require.NoError(t, err)
	})

	_, err = dbschema.EnsureSchema(db, tmpDir)
	require.NoError(t, err)

	target := sqlite.NewTarget(db)

	// Add incusTargetA.
	incusTargetA, err = target.Create(ctx, incusTargetA)
	require.NoError(t, err)

	// Add incusTargetB.
	incusTargetB, err = target.Create(ctx, incusTargetB)
	require.NoError(t, err)

	// Add incusTargetC.
	incusTargetC, err = target.Create(ctx, incusTargetC)
	require.NoError(t, err)

	// Ensure we have three entries
	targets, err := target.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, targets, 3)

	// Ensure we have three entries
	targetNames, err := target.GetAllNames(ctx)
	require.NoError(t, err)
	require.Len(t, targetNames, 3)
	require.ElementsMatch(t, []string{"Target A", "Target B", "Target C"}, targetNames)

	// Should get back incusTargetA unchanged.
	dbIncusTargetA, err := target.GetByName(ctx, incusTargetA.Name)
	require.NoError(t, err)
	require.Equal(t, incusTargetA, dbIncusTargetA)

	dbIncusTargetA, err = target.GetByID(ctx, incusTargetA.ID)
	require.NoError(t, err)
	require.Equal(t, incusTargetA, dbIncusTargetA)

	dbIncusTargetC, err := target.GetByName(ctx, incusTargetC.Name)
	require.NoError(t, err)
	require.Equal(t, incusTargetC, dbIncusTargetC)

	// Test updating a target.
	incusTargetC.Properties = []byte(`{"endpoint": "https://127.0.0.1:6443", "insecure": true, "connectivity_status": 1}`)
	dbIncusTargetC, err = target.UpdateByID(ctx, incusTargetC)
	require.Equal(t, incusTargetC, dbIncusTargetC)
	require.NoError(t, err)
	dbIncusTargetC, err = target.GetByName(ctx, incusTargetC.Name)
	require.NoError(t, err)
	require.Equal(t, incusTargetC, dbIncusTargetC)

	// Delete a target.
	err = target.DeleteByName(ctx, incusTargetA.Name)
	require.NoError(t, err)
	_, err = target.GetByName(ctx, incusTargetA.Name)
	require.ErrorIs(t, err, migration.ErrNotFound)

	// Should have two targets remaining.
	targets, err = target.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, targets, 2)

	// Can't delete a target that doesn't exist.
	err = target.DeleteByName(ctx, "BazBiz")
	require.ErrorIs(t, err, migration.ErrNotFound)

	// Can't update a target that doesn't exist.
	_, err = target.UpdateByID(ctx, incusTargetA)
	require.ErrorIs(t, err, migration.ErrNotFound)

	// Can't add a duplicate target.
	_, err = target.Create(ctx, incusTargetB)
	require.ErrorIs(t, err, migration.ErrConstraintViolation)
}
