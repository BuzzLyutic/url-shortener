package shortcode

import (
	"testing"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		attempt int
	}{
		{
			name:    "simple URL",
			url:     "https://example.com",
			attempt: 0,
		},
		{
			name:    "URL with path",
			url:     "https://example.com/very/long/path/to/resource",
			attempt: 0,
		},
		{
			name:    "URL with query params",
			url:     "https://example.com/search?q=hello&page=1",
			attempt: 0,
		},
		{
			name:    "with collision attempt",
			url:     "https://example.com",
			attempt: 1,
		},
		{
			name:    "empty URL",
			url:     "",
			attempt: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Generate(tt.url, tt.attempt)

			if len(got) != Length {
				t.Errorf("Generate() returned code of length %d, want %d", len(got), Length)
			}

			if !IsValid(got) {
				t.Errorf("Generate() returned invalid code: %s", got)
			}
		})
	}
}

func TestGenerate_Deterministic(t *testing.T) {
	url := "https://example.com/test"

	first := Generate(url, 0)
	second := Generate(url, 0)

	if first != second {
		t.Errorf("Generate() is not deterministic: got %s and %s for same input", first, second)
	}
}

func TestGenerate_DifferentURLs_DifferentCodes(t *testing.T) {
	url1 := "https://example.com/page1"
	url2 := "https://example.com/page2"

	code1 := Generate(url1, 0)
	code2 := Generate(url2, 0)

	if code1 == code2 {
		t.Errorf("Different URLs produced same code: %s", code1)
	}
}

func TestGenerate_DifferentAttempts_DifferentCodes(t *testing.T) {
	url := "https://example.com"

	code0 := Generate(url, 0)
	code1 := Generate(url, 1)
	code2 := Generate(url, 2)

	if code0 == code1 || code1 == code2 || code0 == code2 {
		t.Errorf("Different attempts produced same codes: %s, %s, %s", code0, code1, code2)
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name  string
		code  string
		valid bool
	}{
		{
			name:  "valid code",
			code:  "aB3_xY9z12",
			valid: true,
		},
		{
			name:  "valid all lowercase",
			code:  "abcdefghij",
			valid: true,
		},
		{
			name:  "valid all uppercase",
			code:  "ABCDEFGHIJ",
			valid: true,
		},
		{
			name:  "valid all digits",
			code:  "0123456789",
			valid: true,
		},
		{
			name:  "valid with underscores",
			code:  "a_b_c_d_e_",
			valid: true,
		},
		{
			name:  "too short",
			code:  "abc",
			valid: false,
		},
		{
			name:  "too long",
			code:  "abcdefghijk",
			valid: false,
		},
		{
			name:  "invalid character dash",
			code:  "abc-defghi",
			valid: false,
		},
		{
			name:  "invalid character space",
			code:  "abc defghi",
			valid: false,
		},
		{
			name:  "invalid character special",
			code:  "abc@defghi",
			valid: false,
		},
		{
			name:  "empty string",
			code:  "",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValid(tt.code); got != tt.valid {
				t.Errorf("IsValid(%q) = %v, want %v", tt.code, got, tt.valid)
			}
		})
	}
}

func TestEncode(t *testing.T) {
	// Тестирование функции шифрования на корректную длину возвращаемого кода
	testCases := []uint64{0, 1, 63, 64, 1000000, ^uint64(0)}

	for _, num := range testCases {
		result := encode(num)
		if len(result) != Length {
			t.Errorf("encode(%d) returned length %d, want %d", num, len(result), Length)
		}
		if !IsValid(result) {
			t.Errorf("encode(%d) returned invalid code: %s", num, result)
		}
	}
}

// Бенчмарки

func BenchmarkGenerate(b *testing.B) {
	url := "https://example.com/very/long/path/to/resource?query=param&another=value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Generate(url, 0)
	}
}

func BenchmarkGenerate_WithAttempt(b *testing.B) {
	url := "https://example.com/very/long/path"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Generate(url, i%10)
	}
}

func BenchmarkIsValid(b *testing.B) {
	code := "aB3_xY9z12"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsValid(code)
	}
}
