package migrate

import (
	"fmt"
	"io/fs"
	"log"
	"regexp"
	"sort"
	"strconv"

	"gorm.io/gorm"
)

type Migration struct {
	Version int
	Name    string
	Up      string
	Down    string
}

type Migrator struct {
	db         *gorm.DB
	migrations []Migration
}

func NewMigrator(db *gorm.DB) *Migrator {
	return &Migrator{db: db}
}

func (m *Migrator) LoadMigrations(migrationsFS fs.FS) error {
	return fs.WalkDir(migrationsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		matches := regexp.MustCompile(`^(\d+)_([^.]+)\.(up|down)\.sql$`).FindStringSubmatch(d.Name())
		if len(matches) != 4 {
			return nil
		}

		version, err := strconv.Atoi(matches[1])
		if err != nil {
			return fmt.Errorf("invalid migration version: %w", err)
		}

		content, err := fs.ReadFile(migrationsFS, path)
		if err != nil {
			return fmt.Errorf("failed to read migration file: %w", err)
		}

		name := matches[2]
		dir := matches[3]

		var migration *Migration
		for i := range m.migrations {
			if m.migrations[i].Version == version {
				migration = &m.migrations[i]
				break
			}
		}

		if migration == nil {
			migration = &Migration{
				Version: version,
				Name:    name,
			}
			m.migrations = append(m.migrations, *migration)
		}

		if dir == "up" {
			migration.Up = string(content)
		} else {
			migration.Down = string(content)
		}

		return nil
	})
}

func (m *Migrator) createMigrationsTable() error {
	return m.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`).Error
}

func (m *Migrator) AppliedMigrations() (map[int]bool, error) {
	if err := m.createMigrationsTable(); err != nil {
		return nil, err
	}

	rows, err := m.db.Raw("SELECT version FROM schema_migrations ORDER BY version").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, nil
}

func (m *Migrator) Migrate() error {
	applied, err := m.AppliedMigrations()
	if err != nil {
		return err
	}

	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})

	for _, migration := range m.migrations {
		if applied[migration.Version] {
			continue
		}

		tx := m.db.Begin()
		if tx.Error != nil {
			return tx.Error
		}

		if err := tx.Exec(migration.Up).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}

		if err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", migration.Version).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		log.Printf("Applied migration %d: %s", migration.Version, migration.Name)
	}

	return nil
}

func (m *Migrator) Rollback(version int) error {
	applied, err := m.AppliedMigrations()
	if err != nil {
		return err
	}

	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version > m.migrations[j].Version
	})

	for _, migration := range m.migrations {
		if migration.Version <= version || !applied[migration.Version] {
			continue
		}

		tx := m.db.Begin()
		if tx.Error != nil {
			return tx.Error
		}

		if migration.Down == "" {
			log.Printf("Skipping rollback for migration %d (no down migration)", migration.Version)
			continue
		}

		if err := tx.Exec(migration.Down).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to rollback migration %d: %w", migration.Version, err)
		}

		if err := tx.Exec("DELETE FROM schema_migrations WHERE version = ?", migration.Version).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to remove migration %d record: %w", migration.Version, err)
		}

		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("failed to commit rollback %d: %w", migration.Version, err)
		}

		log.Printf("Rolled back migration %d: %s", migration.Version, migration.Name)
	}

	return nil
}
