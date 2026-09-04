package upgrader

// Test-only aliases retained for the pre-contract-v8 synthetic native-repin
// regression fixtures. Production compatibility uses explicit immutable native
// source anchors instead of a rolling "previous release" model.
const (
	previousNativeReleaseVersion = nativeV060ReleaseVersion
	previousNativeContractSHA256 = nativeContractV7SHA256
)
