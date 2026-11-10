package service

import (
	"slices"
	"testing"
)

func TestNormalizeTags(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "huruf besar dan spasi di tepi", in: []string{" Tugas-Banyak ", "SANTAI"}, want: []string{"tugas-banyak", "santai"}},
		{name: "duplikat setelah normalisasi", in: []string{"Seru", "seru", " SERU"}, want: []string{"seru"}},
		{name: "string kosong dibuang", in: []string{"", " ", "praktis"}, want: []string{"praktis"}},
		{name: "urutan kemunculan pertama dipertahankan", in: []string{"b", "a", "B"}, want: []string{"b", "a"}},
		{name: "input kosong", in: []string{}, want: []string{}},
		{name: "input nil", in: nil, want: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeTags(tt.in)

			if !slices.Equal(got, tt.want) {
				t.Errorf("NormalizeTags(%q) = %q, seharusnya %q", tt.in, got, tt.want)
			}
			if got == nil {
				t.Errorf("NormalizeTags(%q) mengembalikan nil", tt.in)
			}
		})
	}
}