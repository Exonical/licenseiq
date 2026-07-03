package persistence

import (
	"context"

	"gorm.io/gorm"
)

func Migrate(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).AutoMigrate(
		&VendorModel{},
		&VendorContactModel{},
		&ProductModel{},
		&LicenseModel{},
		&AssignmentModel{},
		&AttachmentModel{},
		&UserModel{},
		&APIKeyModel{},
		&AuditLogModel{},
		&FeatureFlagModel{},
		&FeatureFlagAuditModel{},
		&RenewalReminderLogModel{},
		&LicenseIssueLinkModel{},
	)
}
