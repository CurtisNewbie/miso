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

	// misoconfig-prop: KMS decrypt cache max cost in bytes, default to 30MB | 31457280
	PropKmsCacheMaxCost = "kms.cache.max-cost"

	// misoconfig-prop: KMS decrypt cache TTL | 300s
	PropKmsCacheTTL = "kms.cache.ttl"
)

// misoconfig-default-start
func init() {
	miso.SetDefProp(PropKmsEnabled, false)
	miso.SetDefProp(PropKmsCacheMaxCost, 31457280)
	miso.SetDefProp(PropKmsCacheTTL, "300s")
}

// misoconfig-default-end
