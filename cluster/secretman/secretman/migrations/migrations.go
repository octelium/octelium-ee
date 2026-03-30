// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package migrations

import (
	"bytes"
	"context"
	"database/sql"
	"html/template"
)

const migrationTmpl = `

CREATE TABLE IF NOT EXISTS octelium_encrypted_resources (
    id BIGSERIAL PRIMARY KEY,
		uid TEXT NOT NULL,
		resource_version TEXT NOT NULL,
		key_uid TEXT NOT NULL,
		created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
		updated_at TIMESTAMP WITHOUT TIME ZONE,
		ciphertext BYTEA
);

CREATE TABLE IF NOT EXISTS octelium_data_encryption_keys (
    id BIGSERIAL PRIMARY KEY,
		uid TEXT UNIQUE NOT NULL,
		key_uid TEXT,
		key_version TEXT,
		created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
		updated_at TIMESTAMP WITHOUT TIME ZONE,
		ciphertext BYTEA,
		info JSONB
);

CREATE INDEX IF NOT EXISTS idx_octelium_encrypted_resources_uid ON octelium_encrypted_resources(uid);
CREATE INDEX IF NOT EXISTS idx_octelium_encrypted_resources_resource_version ON octelium_encrypted_resources(resource_version);

CREATE INDEX IF NOT EXISTS idx_octelium_data_encryption_keys_uid ON octelium_data_encryption_keys(uid);
`

func Migrate(ctx context.Context, db *sql.DB) error {
	t, err := template.New("migration-tmpl").Parse(migrationTmpl)
	if err != nil {
		return err
	}

	var tpl bytes.Buffer

	if err := t.Execute(&tpl, nil); err != nil {
		panic(err)
	}

	tmpl := tpl.String()

	_, err = db.Exec(tmpl)
	if err != nil {
		return err
	}
	return nil
}
