package catalog

import "testing"

func TestVerifyHash(t *testing.T) {
	data := []byte("hello")
	good := sha256Hex(data)

	tests := []struct {
		name    string
		data    []byte
		want    string
		wantErr bool
	}{
		{"match", data, good, false},
		{"match uppercase", data, upper(good), false},
		{"mismatch", data, sha256Hex([]byte("other")), true},
		{"empty want fails closed", data, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyHash(tt.data, tt.want)
			if (err != nil) != tt.wantErr {
				t.Fatalf("VerifyHash() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManifestVerifyCatalog(t *testing.T) {
	catalogBytes := []byte(`{"openai/gpt-5":{}}`)
	m := Manifest{CatalogSHA256: sha256Hex(catalogBytes)}

	if err := m.VerifyCatalog(catalogBytes); err != nil {
		t.Fatalf("VerifyCatalog() unexpected error: %v", err)
	}
	if err := m.VerifyCatalog([]byte("tampered")); err == nil {
		t.Fatal("VerifyCatalog() expected error on tampered bytes")
	}
}

func TestManifestVerifyProviderSlice(t *testing.T) {
	slice := []byte(`{"openai/gpt-5":{}}`)
	m := Manifest{Providers: []ManifestProvider{
		{ID: "openai", SHA256: sha256Hex(slice)},
	}}

	if err := m.VerifyProviderSlice("openai", slice); err != nil {
		t.Fatalf("VerifyProviderSlice() unexpected error: %v", err)
	}
	if err := m.VerifyProviderSlice("openai", []byte("tampered")); err == nil {
		t.Fatal("VerifyProviderSlice() expected error on tampered bytes")
	}
	if err := m.VerifyProviderSlice("unknown", slice); err == nil {
		t.Fatal("VerifyProviderSlice() expected error for provider not in manifest")
	}
}

func upper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'f' {
			b[i] = c - 32
		}
	}
	return string(b)
}
