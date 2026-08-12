package kms

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// Empty encryption context must serialize to empty bytes, matching
// alibabacloud-encryption-sdk-java (CipherHeader.serializeContext returns
// new byte[0] for an empty context). A 4-byte zero count would break
// AES-GCM body decryption of ciphertext produced by the Java SDK or
// other Java-compatible implementations.
func TestSerializeContextEmpty(t *testing.T) {
	got := serializeContext(nil)
	if len(got) != 0 {
		t.Fatalf("empty context serialized to %d bytes, want 0", len(got))
	}
}

func TestSerializeContextNonEmpty(t *testing.T) {
	ctx := []encryptionContextEntry{{Key: []byte("k1"), Value: []byte("v1")}}
	got := serializeContext(ctx)

	var want []byte
	want = binary.BigEndian.AppendUint32(want, 1)
	want = binary.BigEndian.AppendUint32(want, 2)
	want = append(want, 'k', '1')
	want = binary.BigEndian.AppendUint32(want, 2)
	want = append(want, 'v', '1')

	if !bytes.Equal(got, want) {
		t.Fatalf("non-empty context serialized to %x, want %x", got, want)
	}
}

// Header authenticated fields used for the header auth tag must match the
// Java SDK layout for an empty context: version, algorithm, context count,
// key count, then per key: keyID length, keyID, base64(DEK) length, base64(DEK).
func TestSerializeHeaderFieldsMatchesJavaLayout(t *testing.T) {
	keys := []encryptedDataKey{
		{KeyID: []byte("acs:kms:cn-hangzhou:123:key/abc"), DataKey: []byte{0x01, 0x02, 0x03}},
	}
	got := serializeHeaderFields(1, 2, nil, keys)

	var want []byte
	want = binary.BigEndian.AppendUint32(want, 1)
	want = binary.BigEndian.AppendUint32(want, 2)
	want = binary.BigEndian.AppendUint32(want, 0) // encryptionContext count
	want = binary.BigEndian.AppendUint32(want, 1) // encryptedDataKeys count
	want = binary.BigEndian.AppendUint32(want, uint32(len(keys[0].KeyID)))
	want = append(want, keys[0].KeyID...)
	dekBase64 := []byte("AQID")
	want = binary.BigEndian.AppendUint32(want, uint32(len(dekBase64)))
	want = append(want, dekBase64...)

	if !bytes.Equal(got, want) {
		t.Fatalf("header fields serialized to %x, want %x", got, want)
	}
}
