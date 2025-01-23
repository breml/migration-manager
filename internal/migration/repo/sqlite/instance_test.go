package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	dbschema "github.com/FuturFusion/migration-manager/internal/db"
	dbdriver "github.com/FuturFusion/migration-manager/internal/db/sqlite"
	"github.com/FuturFusion/migration-manager/internal/migration"
	"github.com/FuturFusion/migration-manager/internal/migration/repo/sqlite"
	"github.com/FuturFusion/migration-manager/internal/ptr"
	"github.com/FuturFusion/migration-manager/shared/api"
)

var (
	testSource    = migration.Source{Name: "TestSource", SourceType: api.SOURCETYPE_COMMON, Properties: []byte(`{}`)}
	testTarget    = migration.Target{Name: "TestTarget", Endpoint: "https://localhost:6443"}
	testBatch     = migration.Batch{ID: 1, Name: "TestBatch", TargetID: 1, StoragePool: "", IncludeExpression: "true", MigrationWindowStart: time.Time{}, MigrationWindowEnd: time.Time{}, DefaultNetwork: "network"}
	instanceAUUID = uuid.Must(uuid.NewRandom())

	instanceA = migration.Instance{
		UUID:                  instanceAUUID,
		InventoryPath:         "/path/UbuntuVM",
		Annotation:            "annotation",
		MigrationStatus:       api.MIGRATIONSTATUS_NOT_ASSIGNED_BATCH,
		MigrationStatusString: api.MIGRATIONSTATUS_NOT_ASSIGNED_BATCH.String(),
		LastUpdateFromSource:  time.Now().UTC(),
		SourceID:              1,
		TargetID:              ptr.To(1),
		BatchID:               nil,
		GuestToolsVersion:     123,
		Architecture:          "x86_64",
		HardwareVersion:       "hw version",
		OS:                    "Ubuntu",
		OSVersion:             "24.04",
		Devices:               nil,
		Disks: []api.InstanceDiskInfo{
			{
				Name:                      "disk",
				DifferentialSyncSupported: true,
				SizeInBytes:               123,
			},
		},
		NICs: []api.InstanceNICInfo{
			{
				Network: "net",
				Hwaddr:  "mac",
			},
		},
		Snapshots: nil,
		CPU: api.InstanceCPUInfo{
			NumberCPUs:             2,
			CPUAffinity:            []int32{},
			NumberOfCoresPerSocket: 2,
		},
		Memory: api.InstanceMemoryInfo{
			MemoryInBytes:            4294967296,
			MemoryReservationInBytes: 4294967296,
		},
		UseLegacyBios:     false,
		SecureBootEnabled: false,
		TPMPresent:        false,
		NeedsDiskImport:   false,
		SecretToken:       uuid.Must(uuid.NewRandom()),
	}

	instanceBUUID = uuid.Must(uuid.NewRandom())
	instanceB     = migration.Instance{
		UUID:                  instanceBUUID,
		InventoryPath:         "/path/WindowsVM",
		Annotation:            "annotation",
		MigrationStatus:       api.MIGRATIONSTATUS_NOT_ASSIGNED_BATCH,
		MigrationStatusString: api.MIGRATIONSTATUS_NOT_ASSIGNED_BATCH.String(),
		LastUpdateFromSource:  time.Now().UTC(),
		SourceID:              1,
		TargetID:              ptr.To(1),
		BatchID:               nil,
		GuestToolsVersion:     123,
		Architecture:          "x86_64",
		HardwareVersion:       "hw version",
		OS:                    "Windows",
		OSVersion:             "11",
		Devices:               nil,
		Disks: []api.InstanceDiskInfo{
			{
				Name:                      "disk",
				DifferentialSyncSupported: false,
				SizeInBytes:               321,
			},
		},
		NICs: []api.InstanceNICInfo{
			{
				Network: "net1",
				Hwaddr:  "mac1",
			},
			{
				Network: "net2",
				Hwaddr:  "mac2",
			},
		},
		Snapshots: nil,
		CPU: api.InstanceCPUInfo{
			NumberCPUs:             2,
			CPUAffinity:            []int32{0, 1},
			NumberOfCoresPerSocket: 2,
		},
		Memory: api.InstanceMemoryInfo{
			MemoryInBytes:            4294967296,
			MemoryReservationInBytes: 4294967296,
		},
		UseLegacyBios:     false,
		SecureBootEnabled: true,
		TPMPresent:        true,
		NeedsDiskImport:   false,
		SecretToken:       uuid.Must(uuid.NewRandom()),
	}

	instanceCUUID = uuid.Must(uuid.NewRandom())
	instanceC     = migration.Instance{
		UUID:                  instanceCUUID,
		InventoryPath:         "/path/DebianVM",
		Annotation:            "annotation",
		MigrationStatus:       api.MIGRATIONSTATUS_NOT_ASSIGNED_BATCH,
		MigrationStatusString: api.MIGRATIONSTATUS_NOT_ASSIGNED_BATCH.String(),
		LastUpdateFromSource:  time.Now().UTC(),
		SourceID:              1,
		TargetID:              nil,
		BatchID:               ptr.To(1),
		GuestToolsVersion:     123,
		Architecture:          "arm64",
		HardwareVersion:       "hw version",
		OS:                    "Debian",
		OSVersion:             "bookworm",
		Devices:               nil,
		Disks: []api.InstanceDiskInfo{
			{
				Name:                      "disk1",
				DifferentialSyncSupported: true,
				SizeInBytes:               123,
			},
			{
				Name:                      "disk2",
				DifferentialSyncSupported: true,
				SizeInBytes:               321,
			},
		},
		NICs:      nil,
		Snapshots: nil,
		CPU: api.InstanceCPUInfo{
			NumberCPUs:             4,
			CPUAffinity:            []int32{0, 1, 2, 3},
			NumberOfCoresPerSocket: 2,
		},
		Memory: api.InstanceMemoryInfo{
			MemoryInBytes:            4294967296,
			MemoryReservationInBytes: 4294967296,
		},
		UseLegacyBios:     true,
		SecureBootEnabled: false,
		TPMPresent:        false,
		NeedsDiskImport:   false,
		SecretToken:       uuid.Must(uuid.NewRandom()),
	}
)

func TestInstanceDatabaseActions(t *testing.T) {
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

	sourceSvc := migration.NewSourceService(sqlite.NewSource(db))
	targetSvc := migration.NewTargetService(sqlite.NewTarget(db))

	instance := sqlite.NewInstance(db)
	instanceSvc := migration.NewInstanceService(instance, sourceSvc)

	batchSvc := migration.NewBatchService(sqlite.NewBatch(db), instanceSvc)

	// Cannot add an instance with an invalid source.
	_, err = instance.Create(ctx, instanceA)
	require.Error(t, err)
	_, err = sourceSvc.Create(ctx, testSource)
	require.NoError(t, err)

	// Add dummy target.
	_, err = targetSvc.Create(ctx, testTarget)
	require.NoError(t, err)

	// Add dummy batch.
	_, err = batchSvc.Create(ctx, testBatch)
	require.NoError(t, err)

	// Add instanceA.
	instanceA, err = instance.Create(ctx, instanceA)
	require.NoError(t, err)

	// Add instanceB.
	instanceB, err = instance.Create(ctx, instanceB)
	require.NoError(t, err)

	// Add instanceC.
	instanceC, err = instance.Create(ctx, instanceC)
	require.NoError(t, err)

	// Cannot delete a source or target if referenced by an instance.
	err = sourceSvc.DeleteByName(context.TODO(), testSource.Name)
	require.Error(t, err)
	err = targetSvc.DeleteByName(ctx, testTarget.Name)
	require.Error(t, err)

	// Ensure we have three instances.
	instances, err := instance.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, instances, 3)

	// Should get back instanceA unchanged.
	dbInstanceA, err := instance.GetByID(ctx, instanceA.UUID)
	require.NoError(t, err)
	require.Equal(t, instanceA, dbInstanceA)

	// Test updating an instance.
	instanceB.InventoryPath = "/foo/bar"
	instanceB.CPU.NumberCPUs = 8
	instanceB.MigrationStatus = api.MIGRATIONSTATUS_BACKGROUND_IMPORT
	instanceB.MigrationStatusString = instanceB.MigrationStatus.String()
	instanceB, err = instance.UpdateByID(ctx, instanceB)
	require.NoError(t, err)
	dbInstanceB, err := instance.GetByID(ctx, instanceB.UUID)
	require.NoError(t, err)
	require.Equal(t, instanceB, dbInstanceB)

	// Delete an instance.
	err = instance.DeleteByID(ctx, instanceA.UUID)
	require.NoError(t, err)
	_, err = instance.GetByID(ctx, instanceA.UUID)
	require.Error(t, err)

	// Should have two instances remaining.
	instances, err = instance.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, instances, 2)

	// Can't delete an instance that doesn't exist.
	randomUUID, _ := uuid.NewRandom()
	err = instance.DeleteByID(ctx, randomUUID)
	require.Error(t, err)

	// Can't update an instance that doesn't exist.
	_, err = instance.UpdateByID(ctx, instanceA)
	require.Error(t, err)

	// Can't add a duplicate instance.
	_, err = instance.Create(ctx, instanceB)
	require.Error(t, err)

	// Can't delete a source that has at least one associated instance.
	err = sourceSvc.DeleteByName(ctx, testSource.Name)
	require.Error(t, err)

	// Can't delete a target that has at least one associated instance.
	err = targetSvc.DeleteByName(ctx, testTarget.Name)
	require.Error(t, err)
}

var overridesA = migration.Overrides{UUID: instanceAUUID, LastUpdate: time.Now().UTC(), Comment: "A comment", NumberCPUs: 8, MemoryInBytes: 4096, DisableMigration: true}

func TestInstanceOverridesDatabaseActions(t *testing.T) {
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

	sourceSvc := migration.NewSourceService(sqlite.NewSource(db))
	targetSvc := migration.NewTargetService(sqlite.NewTarget(db))

	instance := sqlite.NewInstance(db)

	// Cannot add an overrides if there's no corresponding instance.
	_, err = instance.CreateOverrides(ctx, overridesA)
	require.Error(t, err)

	// Add the corresponding instance.
	_, err = sourceSvc.Create(ctx, testSource)
	require.NoError(t, err)
	_, err = targetSvc.Create(ctx, testTarget)
	require.NoError(t, err)
	_, err = instance.Create(ctx, instanceA)
	require.NoError(t, err)

	// Add the overrides.
	overridesA, err = instance.CreateOverrides(ctx, overridesA)
	require.NoError(t, err)

	// Should get back overridesA unchanged.
	dbOverridesA, err := instance.GetOverridesByID(ctx, instanceA.UUID)
	require.NoError(t, err)
	require.Equal(t, overridesA, dbOverridesA)

	// The Instance's returned overrides should match what we set.
	dbInstanceA, err := instance.GetByID(ctx, instanceA.UUID)
	require.NoError(t, err)
	require.Equal(t, *dbInstanceA.Overrides, overridesA)

	// Test updating an overrides.
	overridesA.Comment = "An update"
	overridesA.DisableMigration = false
	overridesA, err = instance.UpdateOverridesByID(ctx, overridesA)
	require.NoError(t, err)
	dbOverridesA, err = instance.GetOverridesByID(ctx, instanceA.UUID)
	require.NoError(t, err)
	require.Equal(t, overridesA, dbOverridesA)

	// Can't add a duplicate overrides.
	_, err = instance.CreateOverrides(ctx, overridesA)
	require.Error(t, err)

	// Delete an overrides.
	err = instance.DeleteOverridesByID(ctx, instanceA.UUID)
	require.NoError(t, err)
	_, err = instance.GetOverridesByID(ctx, instanceA.UUID)
	require.Error(t, err)

	// Can't delete an overrides that doesn't exist.
	randomUUID := uuid.Must(uuid.NewRandom())
	err = instance.DeleteOverridesByID(ctx, randomUUID)
	require.Error(t, err)

	// Can't update an overrides that doesn't exist.
	_, err = instance.UpdateOverridesByID(ctx, overridesA)
	require.Error(t, err)

	// Ensure deletion of instance fails, if an overrides is present
	// (cascading delete is handled by the business logic and not the DB layer).
	_, err = instance.CreateOverrides(ctx, overridesA)
	require.NoError(t, err)
	_, err = instance.GetOverridesByID(ctx, instanceA.UUID)
	require.NoError(t, err)
	err = instance.DeleteByID(ctx, instanceA.UUID)
	require.Error(t, err)
}
