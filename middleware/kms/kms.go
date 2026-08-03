package kms

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"io"
	"sort"
	"sync"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	kms "github.com/alibabacloud-go/kms-20160120/v4/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/credentials-go/credentials"
	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/miso"
)

const (
	envelopeVersion    = 1
	aesGcmNoPadding256 = 2
	gcmTagSize         = 16
	gcmNonceSize       = 12
)

// ASN.1 DER envelope structures compatible with alibabacloud-encryption-sdk-java.

type encryptionMessage struct {
	Head encryptionHead
	Body encryptionBody
}

type encryptionHead struct {
	Version           int
	Algorithm         int
	EncryptedDataKeys []encryptedDataKey       `asn1:"set"`
	EncryptionContext []encryptionContextEntry `asn1:"set"`
	HeaderIV          []byte
	HeaderAuthTag     []byte
}

type encryptedDataKey struct {
	KeyID   []byte
	DataKey []byte
}

type encryptionContextEntry struct {
	Key   []byte
	Value []byte
}

type encryptionBody struct {
	IV         []byte
	CipherText []byte
	AuthTag    []byte
}

var module = miso.InitAppModuleFunc(newModule)

type kmsModule struct {
	mu     sync.RWMutex
	client *kms.Client
	keyId  string
}

func newModule() *kmsModule {
	return &kmsModule{}
}

func (m *kmsModule) lazyInit() error {
	m.mu.RLock()
	if m.client != nil {
		m.mu.RUnlock()
		return nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		return nil
	}

	region := miso.GetPropStr(PropKmsRegion)
	if region == "" {
		return errs.NewErrf("kms region not configured, property '%s' is required", PropKmsRegion)
	}
	keyId := miso.GetPropStr(PropKmsKeyId)
	if keyId == "" {
		return errs.NewErrf("kms key-id not configured, property '%s' is required", PropKmsKeyId)
	}

	// Use default credential chain (auto-detects RRSA in ACK, AK/SK in dev)
	cred, err := credentials.NewCredential(nil)
	if err != nil {
		return errs.Wrapf(err, "failed to create Alicloud credential")
	}

	config := &openapi.Config{
		Credential: cred,
		RegionId:   tea.String(region),
		Endpoint:   tea.String("kms." + region + ".aliyuncs.com"),
	}

	client, err := kms.NewClient(config)
	if err != nil {
		return errs.Wrapf(err, "failed to create KMS client")
	}

	m.client = client
	m.keyId = keyId
	miso.Infof("KMS client initialized, region: %s, keyId: %s", region, keyId)
	return nil
}

func (m *kmsModule) encrypt(plaintext []byte) (string, error) {
	if err := m.lazyInit(); err != nil {
		return "", err
	}

	// 1. Generate data key via KMS
	req := &kms.GenerateDataKeyRequest{
		KeyId:         tea.String(m.keyId),
		NumberOfBytes: tea.Int32(32), // AES-256
	}
	resp, err := m.client.GenerateDataKeyWithOptions(req, &util.RuntimeOptions{})
	if err != nil {
		return "", errs.Wrapf(err, "KMS GenerateDataKey failed")
	}

	plainDEK, err := base64.StdEncoding.DecodeString(tea.StringValue(resp.Body.Plaintext))
	if err != nil {
		return "", errs.Wrapf(err, "failed to decode data key plaintext")
	}
	encryptedDEK, err := base64.StdEncoding.DecodeString(tea.StringValue(resp.Body.CiphertextBlob))
	if err != nil {
		return "", errs.Wrapf(err, "failed to decode data key ciphertext")
	}

	// 2. Generate random 12-byte headerIV
	headerIV := make([]byte, gcmNonceSize)
	if _, err := io.ReadFull(rand.Reader, headerIV); err != nil {
		return "", errs.Wrapf(err, "failed to generate header IV")
	}

	// 3. Compute header auth tag
	// Serialize authenticated fields as raw binary big-endian
	keyIDBytes := []byte(m.keyId)
	encryptedDEKBase64 := []byte(base64.StdEncoding.EncodeToString(encryptedDEK))

	var headerAAD []byte
	headerAAD = binary.BigEndian.AppendUint32(headerAAD, uint32(envelopeVersion))
	headerAAD = binary.BigEndian.AppendUint32(headerAAD, uint32(aesGcmNoPadding256))
	headerAAD = binary.BigEndian.AppendUint32(headerAAD, 0) // encryptionContext count
	headerAAD = binary.BigEndian.AppendUint32(headerAAD, 1) // encryptedDataKeys count
	headerAAD = binary.BigEndian.AppendUint32(headerAAD, uint32(len(keyIDBytes)))
	headerAAD = append(headerAAD, keyIDBytes...)
	headerAAD = binary.BigEndian.AppendUint32(headerAAD, uint32(len(encryptedDEKBase64)))
	headerAAD = append(headerAAD, encryptedDEKBase64...)

	block, err := aes.NewCipher(plainDEK)
	if err != nil {
		return "", errs.Wrapf(err, "aes.NewCipher failed")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errs.Wrapf(err, "cipher.NewGCM failed")
	}

	// AES-GCM encrypt with empty plaintext returns just the 16-byte tag
	headerAuthTag := gcm.Seal(nil, headerIV, nil, headerAAD)

	// 4. Generate random 12-byte body IV
	bodyIV := make([]byte, gcmNonceSize)
	if _, err := io.ReadFull(rand.Reader, bodyIV); err != nil {
		return "", errs.Wrapf(err, "failed to generate body IV")
	}

	// 5. Body encryption
	// AAD = encryptionContext serialized (empty context: just 4 bytes: 0)
	var contextAAD []byte
	contextAAD = binary.BigEndian.AppendUint32(contextAAD, 0)

	// AES-GCM encrypt: returns ciphertext+tag concatenated
	result := gcm.Seal(nil, bodyIV, plaintext, contextAAD)
	cipherText := result[:len(result)-gcmTagSize]
	authTag := result[len(result)-gcmTagSize:]

	// 6. Zero plaintext data key
	for i := range plainDEK {
		plainDEK[i] = 0
	}

	// 7. Build ASN.1 struct
	msg := encryptionMessage{
		Head: encryptionHead{
			Version:   envelopeVersion,
			Algorithm: aesGcmNoPadding256,
			EncryptedDataKeys: []encryptedDataKey{
				{
					KeyID:   keyIDBytes,
					DataKey: encryptedDEK,
				},
			},
			EncryptionContext: []encryptionContextEntry{},
			HeaderIV:          headerIV,
			HeaderAuthTag:     headerAuthTag,
		},
		Body: encryptionBody{
			IV:         bodyIV,
			CipherText: cipherText,
			AuthTag:    authTag,
		},
	}

	// 8. Marshal to DER and base64 encode
	derBytes, err := asn1.Marshal(msg)
	if err != nil {
		return "", errs.Wrapf(err, "failed to marshal ASN.1 envelope")
	}

	return base64.StdEncoding.EncodeToString(derBytes), nil
}

func (m *kmsModule) decrypt(encoded string) (string, error) {
	if err := m.lazyInit(); err != nil {
		return "", err
	}

	// 1. base64 decode → DER bytes
	derBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", errs.Wrapf(err, "base64 decode failed")
	}

	// 2. Parse ASN.1 DER envelope
	var msg encryptionMessage
	if _, err := asn1.Unmarshal(derBytes, &msg); err != nil {
		return "", errs.Wrapf(err, "failed to unmarshal ASN.1 envelope")
	}

	// 3. Validate version and algorithm
	if msg.Head.Version != envelopeVersion {
		return "", errs.NewErrf("unsupported envelope version: %d", msg.Head.Version)
	}
	if msg.Head.Algorithm != aesGcmNoPadding256 {
		return "", errs.NewErrf("unsupported algorithm: %d", msg.Head.Algorithm)
	}
	if len(msg.Head.EncryptedDataKeys) == 0 {
		return "", errs.NewErrf("no encrypted data keys in envelope")
	}

	// 4. Extract encrypted DEK (raw binary from ASN.1)
	encryptedDEK := msg.Head.EncryptedDataKeys[0].DataKey

	// 5. Base64-encode raw encrypted DEK → send to KMS Decrypt API
	req := &kms.DecryptRequest{
		CiphertextBlob: tea.String(base64.StdEncoding.EncodeToString(encryptedDEK)),
	}
	resp, err := m.client.DecryptWithOptions(req, &util.RuntimeOptions{})
	if err != nil {
		return "", errs.Wrapf(err, "KMS Decrypt failed")
	}

	// 6. Base64-decode response → plainDEK
	plainDEK, err := base64.StdEncoding.DecodeString(tea.StringValue(resp.Body.Plaintext))
	if err != nil {
		return "", errs.Wrapf(err, "failed to decode data key plaintext")
	}

	// 7. Verify header auth tag
	block, err := aes.NewCipher(plainDEK)
	if err != nil {
		return "", errs.Wrapf(err, "aes.NewCipher failed")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errs.Wrapf(err, "cipher.NewGCM failed")
	}

	// Re-serialize authenticated fields
	headerAAD := serializeHeaderFields(msg.Head.Version, msg.Head.Algorithm,
		msg.Head.EncryptionContext, msg.Head.EncryptedDataKeys)

	// Verify: gcm.Open with just the tag as ciphertext
	if _, err := gcm.Open(nil, msg.Head.HeaderIV, msg.Head.HeaderAuthTag, headerAAD); err != nil {
		return "", errs.Wrapf(err, "header auth tag verification failed")
	}

	// 8. Decrypt body
	contextAAD := serializeContext(msg.Head.EncryptionContext)

	// Concatenate cipherText + authTag for gcm.Open
	cipherTextPlusTag := append(msg.Body.CipherText, msg.Body.AuthTag...)

	plaintext, err := gcm.Open(nil, msg.Body.IV, cipherTextPlusTag, contextAAD)
	if err != nil {
		return "", errs.Wrapf(err, "AES-GCM body decrypt failed")
	}

	// 9. Zero plaintext data key
	for i := range plainDEK {
		plainDEK[i] = 0
	}

	return string(plaintext), nil
}

// serializeHeaderFields builds the raw binary big-endian authenticated fields
// for header auth tag computation.
func serializeHeaderFields(version, algorithm int, ctx []encryptionContextEntry, keys []encryptedDataKey) []byte {
	// Sort keys by keyId bytes ascending
	sorted := make([]encryptedDataKey, len(keys))
	copy(sorted, keys)
	sort.Slice(sorted, func(i, j int) bool {
		return string(sorted[i].KeyID) < string(sorted[j].KeyID)
	})

	var buf []byte
	buf = binary.BigEndian.AppendUint32(buf, uint32(version))
	buf = binary.BigEndian.AppendUint32(buf, uint32(algorithm))
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(ctx)))
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(sorted)))
	for _, k := range sorted {
		buf = binary.BigEndian.AppendUint32(buf, uint32(len(k.KeyID)))
		buf = append(buf, k.KeyID...)
		// DataKey in authenticated fields: base64-encoded bytes of encrypted DEK
		dekBase64 := []byte(base64.StdEncoding.EncodeToString(k.DataKey))
		buf = binary.BigEndian.AppendUint32(buf, uint32(len(dekBase64)))
		buf = append(buf, dekBase64...)
	}
	return buf
}

// serializeContext builds the raw binary big-endian serialization of
// encryption context entries for use as AAD in body encryption.
func serializeContext(ctx []encryptionContextEntry) []byte {
	// Sort by key bytes ascending
	sorted := make([]encryptionContextEntry, len(ctx))
	copy(sorted, ctx)
	sort.Slice(sorted, func(i, j int) bool {
		return string(sorted[i].Key) < string(sorted[j].Key)
	})

	var buf []byte
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(sorted)))
	for _, e := range sorted {
		buf = binary.BigEndian.AppendUint32(buf, uint32(len(e.Key)))
		buf = append(buf, e.Key...)
		buf = binary.BigEndian.AppendUint32(buf, uint32(len(e.Value)))
		buf = append(buf, e.Value...)
	}
	return buf
}

func (m *kmsModule) addHealthIndicator() {
	miso.AddHealthIndicator(miso.HealthIndicator{
		Name: "KMS Component",
		CheckHealth: func(rail miso.Rail) bool {
			m.mu.RLock()
			initialized := m.client != nil
			m.mu.RUnlock()
			if !initialized {
				rail.Errorf("KMS client not initialized")
				return false
			}
			return true
		},
	})
}

func kmsBootstrap(rail miso.Rail) error {
	m := module()
	if err := m.lazyInit(); err != nil {
		return errs.Wrapf(err, "failed to initialize KMS client")
	}
	m.addHealthIndicator()
	return nil
}

func kmsBootstrapCondition(rail miso.Rail) (bool, error) {
	return miso.GetPropBool(PropKmsEnabled), nil
}

func init() {
	// Register bootstrap callback
	miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
		Name:      "Bootstrap KMS",
		Bootstrap: kmsBootstrap,
		Condition: kmsBootstrapCondition,
		Order:     miso.BootstrapOrderL1,
	})

	// Register "kms" prop func for config value decryption
	// Config values like "kms(base64envelope)" will be resolved by calling decrypt
	if err := miso.RegisterPropFunc("kms", func(arg string) (string, error) {
		return module().decrypt(arg)
	}); err != nil {
		panic(err)
	}
}

// Encrypt plaintext using KMS envelope encryption.
// Returns base64-encoded ASN.1 DER envelope compatible with alibabacloud-encryption-sdk-java.
// Recommended storage: VARCHAR(1024) for typical config values (passwords, tokens, keys).
func Encrypt(plaintext string) (string, error) {
	return module().encrypt([]byte(plaintext))
}

// Decrypt base64-encoded envelope ciphertext using KMS.
func Decrypt(encoded string) (string, error) {
	return module().decrypt(encoded)
}
