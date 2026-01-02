package features

import (
	"strings"
	"testing"
)

func TestDetectParameterType(t *testing.T) {
	tests := []struct {
		key      string
		value    string
		expected ParameterType
		desc     string
	}{
		// Normal strings
		{"NAME", "John Doe", StringType, "normal name"},
		{"AGE", "25", StringType, "normal number"},
		{"LIST", "a,b,c", StringType, "comma separated"},

		// Keywords in key
		{"PASSWORD", "secret123", SecureStringType, "password in key"},
		{"API_KEY", "value", SecureStringType, "api in key"},
		{"SECRET_TOKEN", "token", SecureStringType, "secret in key"},
		{"DB_PASSWORD", "pass", SecureStringType, "password in key"},
		{"CERT", "cert", SecureStringType, "cert in key"},
		{"SECURE_CONFIG", "config", SecureStringType, "secure in key"},

		// Long base64-like
		{"TOKEN", strings.Repeat("A", 25), SecureStringType, "long alnum string"},
		{"KEY", "SGVsbG8gV29ybGQ=", SecureStringType, "base64 string"},

		// URLs with creds
		{"DATABASE_URL", "https://user:password@example.com/db", SecureStringType, "url with creds"},
		{"API_URL", "http://admin:secret@api.example.com", SecureStringType, "http url with creds"},

		// JWT-like
		{"JWT_TOKEN", "header.payload.signature", SecureStringType, "jwt three parts"},
		{"ACCESS_TOKEN", "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJ0ZXN0IiwiZXhwIjoxNjMzNjY0MDAwfQ.signature", SecureStringType, "actual jwt"},

		// Certificates
		{"CERT_PEM", "-----BEGIN CERTIFICATE-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...\n-----END CERTIFICATE-----", SecureStringType, "certificate"},
		{"PUBLIC_KEY", "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...\n-----END PUBLIC KEY-----", SecureStringType, "public key"},

		// MD5 hash (hex, detected as long alnum)
		{"MD5_HASH", "9e107d9d372bb6826bd81d3542a419d6", SecureStringType, "md5 hash"},

		// Edge cases
		{"", "value", StringType, "empty key"},
		{"KEY", "", SecureStringType, "empty value with key keyword"},
		{"PASSWORD", "short", SecureStringType, "password key even short value"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := detectParameterType(tt.key, tt.value)
			if result != tt.expected {
				t.Errorf("detectParameterType(%q, %q) = %q; want %q", tt.key, tt.value, result, tt.expected)
			}
		})
	}
}
