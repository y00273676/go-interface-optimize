package after

import "testing"

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		New()
	}
}

func BenchmarkNewUseReflect(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewUseReflect()
	}
}

func BenchmarkNewQuickReflect(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewQuickReflect()
	}
}
