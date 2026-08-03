package kms

import "github.com/curtisnewbie/miso/miso"

// misoconfig-section: KMS Configuration
const (
	// misoconfig-prop: enable KMS integration | false
	PropKmsEnabled = "kms.enabled"

	// misoconfig-prop: Alicloud KMS region ID (e.g., cn-hangzhou)
	PropKmsRegion = "kms.region"

	// misoconfig-prop: KMS master key ID or ARN for envelope encryption
	PropKmsKeyId = "kms.key-id"
)

// misoconfig-default-start
func init() {
	miso.SetDefProp(PropKmsEnabled, false)
}

// misoconfig-default-end
