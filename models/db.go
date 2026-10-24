package models

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"orcn/pkg/routing"
)

func InitDB(filepath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(filepath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Resolve duplicate provider connection names before the unique
	// (organization_id, name) index is created. Rows created before the
	// constraint may collide and would otherwise abort the migration.
	if db.Migrator().HasTable(&ProviderConnection{}) &&
		db.Migrator().HasColumn(&ProviderConnection{}, "OrganizationID") {
		if err := db.Exec(`UPDATE provider_connections
			SET name = name || '-' || substr(id, 1, 8)
			WHERE rowid NOT IN (
				SELECT MIN(rowid) FROM provider_connections GROUP BY organization_id, name
			)`).Error; err != nil {
			return nil, err
		}
	}

	err = db.AutoMigrate(&Deployment{}, &Node{}, &Job{}, &Secret{}, &Template{}, &Registry{}, &RouteEndpoint{}, &ProviderConnection{}, &Organization{}, &Machine{})
	if err != nil {
		return nil, err
	}

	// Ensure organization_id column exists with default for secrets (SQLite limitation)
	if !db.Migrator().HasColumn(&Secret{}, "OrganizationID") {
		if err := db.Exec("ALTER TABLE secrets ADD COLUMN organization_id TEXT NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'").Error; err != nil {
			return nil, err
		}
	}

	// Ensure default organization exists
	defaultOrg := Organization{
		ID:          "00000000-0000-0000-0000-000000000001",
		Name:        "default",
		Description: "Default organization for single-tenant mode",
	}
	db.FirstOrCreate(&defaultOrg, Organization{ID: defaultOrg.ID})

	if err := migrateDeploymentRouteIdentity(db); err != nil {
		return nil, err
	}

	if err := repairProviderConnectionSecrets(db); err != nil {
		return nil, err
	}

	return db, nil
}

// migrateDeploymentRouteIdentity transitions deployments from a global name to
// an organization-scoped name and backfills the immutable ingress route key.
// It is deliberately explicit because AutoMigrate does not remove obsolete
// unique indexes.
func migrateDeploymentRouteIdentity(db *gorm.DB) error {
	if err := db.Model(&Deployment{}).
		Where("organization_id IS NULL OR organization_id = ''").
		Update("organization_id", "00000000-0000-0000-0000-000000000001").Error; err != nil {
		return err
	}

	var deployments []Deployment
	if err := db.Preload("Organization").Find(&deployments).Error; err != nil {
		return err
	}
	for _, deployment := range deployments {
		if deployment.RouteKey != "" {
			continue
		}
		key, err := routing.DeploymentRouteKey(deployment.Name, deployment.Organization.Name)
		if err != nil {
			return fmt.Errorf("build route key for deployment %q: %w", deployment.ID, err)
		}
		if err := db.Model(&Deployment{}).Where("id = ?", deployment.ID).Update("route_key", key).Error; err != nil {
			return err
		}
	}

	// GORM used this conventional name for the former Name uniqueIndex. It must
	// be removed so a different organization can create the same deployment name.
	if db.Migrator().HasIndex(&Deployment{}, "idx_deployments_name") {
		if err := db.Migrator().DropIndex(&Deployment{}, "idx_deployments_name"); err != nil {
			return err
		}
	}

	return db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_deployments_route_key ON deployments(route_key)").Error
}

// repairProviderConnectionSecrets fixes rows created before Secret had a UUID
// hook. Such secrets ended up with an empty primary key, and the provider
// connections referencing them stored an empty (or dangling) credential id,
// which made credential resolution pick an arbitrary secret.
func repairProviderConnectionSecrets(db *gorm.DB) error {
	// Drop ghost provider connections that have no primary key.
	if err := db.Exec("DELETE FROM provider_connections WHERE id IS NULL OR id = ''").Error; err != nil {
		return err
	}

	// Assign a UUID to every secret that is missing one.
	type secretRow struct {
		Rowid int64 `gorm:"column:rowid"`
	}
	var rows []secretRow
	if err := db.Raw("SELECT rowid AS rowid FROM secrets WHERE id IS NULL OR id = ''").Scan(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		if err := db.Exec("UPDATE secrets SET id = ? WHERE rowid = ?", uuid.New().String(), r.Rowid).Error; err != nil {
			return err
		}
	}

	// Relink provider connections to the secret named "<provider>/<name>" when
	// their credential id is empty or points to a secret that no longer exists.
	return db.Exec(`
		UPDATE provider_connections
		SET credential_secret_id = (
			SELECT s.id FROM secrets s
			WHERE s.name = provider_connections.provider || '/' || provider_connections.name
			ORDER BY s.created_at DESC
			LIMIT 1
		)
		WHERE (
				credential_secret_id IS NULL
				OR credential_secret_id = ''
				OR NOT EXISTS (SELECT 1 FROM secrets s2 WHERE s2.id = provider_connections.credential_secret_id)
			)
			AND EXISTS (
				SELECT 1 FROM secrets s
				WHERE s.name = provider_connections.provider || '/' || provider_connections.name
			)`).Error
}
