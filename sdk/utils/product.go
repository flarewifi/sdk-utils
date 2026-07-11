package sdkutils

// ProductInfo is the on-disk schema of core/product.json: the plaintext
// per-B2B-partner product version plus Data, an AES-GCM encrypted blob (keyed
// on the shared RPC_TOKEN secret) unmarshaling to ProductFields. Written once
// per software-release build (go/builder's writeProductVersion) and read by
// the device (core/flarewifi's core/utils/product package). Shared here, not
// duplicated as anonymous structs in both modules, since go/builder cannot
// import core/flarewifi's core packages (leaf-module boundary) and the two
// sides must agree byte-for-byte on the JSON shape.
type ProductInfo struct {
	Version string `json:"version"`
	Data    string `json:"data"`
}

// ProductFields is ProductInfo.Data, decrypted and unmarshaled. brand_id and
// device_config restamp on every software-release build (they must track the
// machine's CURRENTLY installed release for update-eligibility/product-transfer
// matching). device_model is NOT here — it lives in OsRelease instead, since it
// must stay stable for the device's physical lifetime.
type ProductFields struct {
	BrandId      string `json:"brand_id"`
	DeviceConfig string `json:"device_config"`
}
